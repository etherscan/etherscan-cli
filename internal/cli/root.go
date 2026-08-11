package cli

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/etherscan/etherscan-cli/internal/brand"
	"github.com/etherscan/etherscan-cli/internal/chains"
	"github.com/etherscan/etherscan-cli/internal/client"
	"github.com/etherscan/etherscan-cli/internal/config"
	"github.com/etherscan/etherscan-cli/internal/output"
	"github.com/etherscan/etherscan-cli/internal/tui"
	"github.com/etherscan/etherscan-cli/internal/updater"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

type globalState struct {
	apiKey   string
	chain    string
	baseURL  string
	out      string
	compact  bool
	timeout  time.Duration
	rate     float64
	verbose  bool
	debug    bool
	yes      bool
	file     string
	maxPages int
	all      bool
}

// removedOutputFlags maps flags dropped in favour of -o/--output to their replacement. Without
// this, a script still passing --json gets cobra's bare "unknown flag" and has to go read --help
// to find out what replaced it.
var removedOutputFlags = map[string]string{"--json": "-o json", "--csv": "-o csv"}

type updateManager interface {
	Check(context.Context, string, bool) (updater.Result, error)
	Skip(string) error
	DetectMethod() string
	Upgrade(context.Context, string, string, io.Writer, io.Writer) (bool, error)
	Uninstall(context.Context, string, io.Writer, io.Writer) (bool, error)
}

func NewRootCommand(info BuildInfo) *cobra.Command {
	return newRootCommand(info, updater.NewService())
}

func newRootCommand(info BuildInfo, updates updateManager) *cobra.Command {
	state := &globalState{timeout: 30 * time.Second, rate: 3, maxPages: 20}
	root := &cobra.Command{
		Use:           "etherscan",
		Short:         "Command-line client for the Etherscan V2 API",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// A bare invocation prints the Quick Start guide (coloured at a terminal,
			// plain when piped). The interactive explorer is launched explicitly with
			// `etherscan tui`; `etherscan --help` lists every command.
			printSplash(cmd.OutOrStdout(), info)
			return nil
		},
	}
	root.PersistentFlags().StringVar(&state.apiKey, "api-key", "", "API key for this command (overrides login/ETHERSCAN_API_KEY)")
	root.PersistentFlags().StringVar(&state.apiKey, "apikey", "", "alias for --api-key")
	root.PersistentFlags().StringVar(&state.chain, "chain", "", "chain name or chainid")
	root.PersistentFlags().StringVar(&state.baseURL, "base-url", "", "API base URL")
	root.PersistentFlags().StringVarP(&state.out, "output", "o", "", "output format: json (default), table, csv")
	root.PersistentFlags().BoolVar(&state.compact, "compact", false, "compact JSON output")
	root.PersistentFlags().DurationVar(&state.timeout, "timeout", 30*time.Second, "request timeout")
	root.PersistentFlags().Float64Var(&state.rate, "rate-limit", 3, "client-side request rate limit per second (free-tier API V2 default; raise for higher tiers)")
	root.PersistentFlags().BoolVarP(&state.verbose, "verbose", "v", false, "log request URL and timing to stderr")
	root.PersistentFlags().BoolVar(&state.debug, "debug", false, "dump raw response bodies to stderr")
	root.PersistentFlags().BoolVar(&state.yes, "yes", false, "skip confirmation for sensitive submit actions")
	root.PersistentFlags().BoolVar(&state.all, "all", false, "auto-paginate list commands")
	root.PersistentFlags().IntVar(&state.maxPages, "max-pages", 20, "maximum pages for --all")
	hideFlags(root, "apikey", "base-url", "compact", "max-pages", "rate-limit", "timeout", "verbose", "debug", "yes")
	// Registered on root only: cobra's FlagErrorFunc walks up to the parent when a subcommand
	// has none, so this covers every command in the tree.
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		for flag, replacement := range removedOutputFlags {
			if err.Error() == "unknown flag: "+flag {
				return fmt.Errorf("%s was removed; use %s instead", flag, replacement)
			}
		}
		return err
	})

	root.AddCommand(loginCommand(state), logoutCommand(state), uninstallCommand(state, updates), configCommand(state), chainsCommand(state), whoamiCommand(state), versionCommand(info), updateCommand(info, updates), tuiCommand(state, info, updates), completionCommand(root))
	addEndpointCommands(root, state)
	return root
}

func hideFlags(cmd *cobra.Command, names ...string) {
	for _, name := range names {
		_ = cmd.PersistentFlags().MarkHidden(name)
	}
}

func addEndpointCommands(root *cobra.Command, state *globalState) {
	groups := map[string]*cobra.Command{}
	for _, spec := range endpoints() {
		if spec.Module == "getapilimit" {
			root.AddCommand(endpointCommand(state, spec))
			continue
		}
		group, ok := groups[spec.Module]
		if !ok {
			group = &cobra.Command{Use: spec.Module, Short: "Etherscan " + spec.Module + " commands"}
			groups[spec.Module] = group
			root.AddCommand(group)
		}
		group.AddCommand(endpointCommand(state, spec))
	}
}

func endpointCommand(state *globalState, spec EndpointSpec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   spec.Use,
		Short: spec.Short,
		Args: func(cmd *cobra.Command, args []string) error {
			positionals := requiredParams(spec)
			if len(args) > len(positionals) {
				return fmt.Errorf("too many arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			params, err := collectParams(cmd, spec, args)
			if err != nil {
				return err
			}
			for name, value := range spec.FixedParams {
				params[name] = value
			}
			if err := populateFileParam(state, params); err != nil {
				return err
			}
			if err := validateParams(spec, params); err != nil {
				return err
			}
			rt, err := runtime(state)
			if err != nil {
				return err
			}
			if err := validateEndpointChain(spec, rt.chain); err != nil {
				return err
			}
			if spec.Sensitive && !state.yes {
				if err := confirm(cmd.Context(), fmt.Sprintf("Submit %s/%s?", spec.Module, spec.Action), cmd.InOrStdin(), cmd.ErrOrStderr()); err != nil {
					return err
				}
			}
			if state.all && spec.Paginated {
				return runAllPages(cmd.Context(), rt, spec, params, state.maxPages)
			}
			if err := validatePagination(spec, params); err != nil {
				return err
			}
			result, err := call(cmd.Context(), rt.client, spec, params)
			if err != nil {
				return err
			}
			return output.Write(os.Stdout, result.Raw, rt.format, state.compact, spec.Columns)
		},
	}
	for _, param := range spec.Params {
		value := new(string)
		canonical := flagName(param.Name)
		cmd.Flags().StringVar(value, canonical, "", param.Usage)
		if legacy := legacyFlagName(param.Name); legacy != "" && legacy != canonical {
			cmd.Flags().StringVar(value, legacy, "", param.Usage)
			_ = cmd.Flags().MarkDeprecated(legacy, "use --"+canonical+" instead")
			_ = cmd.Flags().MarkHidden(legacy)
		}
	}
	if spec.AcceptsFile {
		cmd.Flags().StringVar(&state.file, "file", "", "read source/payload content from file")
	}
	return cmd
}

func collectParams(cmd *cobra.Command, spec EndpointSpec, args []string) (map[string]string, error) {
	params := map[string]string{}
	positionals := requiredParams(spec)
	for i, value := range args {
		params[positionals[i].Name] = value
	}
	for _, param := range spec.Params {
		value, err := cmd.Flags().GetString(flagName(param.Name))
		if err != nil {
			return nil, err
		}
		if value != "" || params[param.Name] == "" {
			params[param.Name] = value
		}
	}
	return params, nil
}

type resolvedRuntime struct {
	client *client.Client
	format output.Format
	chain  chains.Chain
}

// errNoAPIKey is returned by runtime() when no key is resolved. An API key is
// required: keyless requests are throttled server-side to ~1 req/3s and fail with
// confusing NOTOK errors, so the CLI fails fast with the fix in the message instead.
var errNoAPIKey = errors.New("no API key configured; run 'etherscan login' or set ETHERSCAN_API_KEY")

// resolveKey returns the effective API key: the --api-key flag wins, then the
// env > config precedence implemented by config.GetAPIKey.
func resolveKey(state *globalState, cfg config.File) string {
	if state.apiKey != "" {
		return state.apiKey
	}
	key, _ := config.GetAPIKey(cfg)
	return key
}

func runtime(state *globalState) (resolvedRuntime, error) {
	cfg, _, err := config.Load()
	if err != nil {
		return resolvedRuntime{}, err
	}
	key := resolveKey(state, cfg)
	if key == "" {
		return resolvedRuntime{}, errNoAPIKey
	}
	return buildRuntime(state, cfg, key)
}

// buildRuntime constructs the shared runtime from already-loaded configuration.
// Most CLI commands call runtime(), which rejects an empty key first. The TUI is
// the sole caller allowed to pass an empty key so users can browse before setup.
func buildRuntime(state *globalState, cfg config.File, key string) (resolvedRuntime, error) {
	chainInput := firstNonEmpty(state.chain, os.Getenv("ETHERSCAN_CHAIN"), cfg.DefaultChain, "ethereum")
	chain, err := chains.Resolve(chainInput)
	if err != nil {
		return resolvedRuntime{}, err
	}
	baseURL := firstNonEmpty(state.baseURL, os.Getenv("ETHERSCAN_BASE_URL"), cfg.BaseURL, client.DefaultBaseURL)
	if !strings.HasPrefix(baseURL, "https://") {
		fmt.Fprintf(os.Stderr, "warning: non-HTTPS base URL: %s\n", baseURL)
	}
	format, err := output.ParseFormat(firstNonEmpty(state.out, cfg.DefaultOutput, string(output.DefaultFormat)))
	if err != nil {
		return resolvedRuntime{}, err
	}
	return resolvedRuntime{
		client: client.New(client.Options{BaseURL: baseURL, APIKey: key, ChainID: chain.ID, Timeout: state.timeout, RateLimit: state.rate, Verbose: state.verbose, Debug: state.debug, Stderr: os.Stderr}),
		format: format,
		chain:  chain,
	}, nil
}

// rebindRuntimeChain changes only the active chain. The cloned client shares the
// existing limiter and transport, and the runtime keeps its resolved key, base
// URL, output format, and other session settings.
func rebindRuntimeChain(rt *resolvedRuntime, nameOrID string) (chains.Chain, error) {
	chain, err := chains.Resolve(nameOrID)
	if err != nil {
		return chains.Chain{}, err
	}
	rt.client = rt.client.ForChain(chain.ID)
	rt.chain = chain
	return chain, nil
}

func call(ctx context.Context, c *client.Client, spec EndpointSpec, params map[string]string) (client.Result, error) {
	clean := map[string]string{}
	for k, v := range params {
		if strings.TrimSpace(v) != "" {
			clean[k] = v
		}
	}
	if spec.Post {
		return c.PostForm(ctx, spec.Module, spec.Action, clean, !spec.NoRetry)
	}
	return c.Get(ctx, spec.Module, spec.Action, clean, !spec.NoRetry)
}

func runAllPages(ctx context.Context, rt resolvedRuntime, spec EndpointSpec, params map[string]string, maxPages int) error {
	offset := params["offset"]
	if offset == "" {
		offset = "100"
		params["offset"] = offset
	}
	var combined []map[string]string
	reachedEnd := false
	limit := max(1, maxPages)
	for page := 1; page <= limit; page++ {
		params["page"] = fmt.Sprint(page)
		result, err := call(ctx, rt.client, spec, params)
		if err != nil {
			return err
		}
		rows, scalar, err := output.Rows(result.Raw)
		if err != nil {
			return err
		}
		if scalar != "" {
			return output.Write(os.Stdout, result.Raw, rt.format, false, spec.Columns)
		}
		combined = append(combined, rows...)
		fmt.Fprintf(os.Stderr, "fetched page %d (%d rows)\n", page, len(rows))
		if len(rows) == 0 || len(rows) < atoi(offset) {
			reachedEnd = true
			break
		}
	}
	if !reachedEnd {
		fmt.Fprintf(os.Stderr, "warning: stopped at --max-pages=%d (%d rows); results may be truncated. Increase --max-pages or narrow --startblock/--endblock.\n", limit, len(combined))
	}
	raw, err := json.Marshal(combined)
	if err != nil {
		return err
	}
	return output.Write(os.Stdout, raw, rt.format, false, spec.Columns)
}

// validateKeyLive checks a key against the API (getapilimit) before it is saved,
// so a typo'd key is rejected at login/setup time rather than on first use.
func validateKeyLive(ctx context.Context, state *globalState, key, chainID, baseURL string) error {
	validator := client.New(client.Options{BaseURL: baseURL, APIKey: key, ChainID: chainID, Timeout: state.timeout, RateLimit: state.rate})
	if _, err := validator.Get(ctx, "getapilimit", "getapilimit", nil, false); err != nil {
		return fmt.Errorf("API key validation failed: %w", err)
	}
	return nil
}

func loginCommand(state *globalState) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Store an Etherscan API key",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := config.Load()
			if err != nil {
				return err
			}
			// The chain and base URL are resolved up front because the save closure
			// below captures them, and it has to exist before the prompt is drawn.
			chain, err := chains.Resolve(firstNonEmpty(state.chain, cfg.DefaultChain, "ethereum"))
			if err != nil {
				return err
			}
			baseURL := firstNonEmpty(state.baseURL, cfg.BaseURL, client.DefaultBaseURL)

			var savedPath, savedLabel string
			save := func(ctx context.Context, key string) (string, error) {
				key = strings.TrimSpace(key)
				if key == "" {
					return "", errors.New("empty API key")
				}
				if err := checkKeyShape(key); err != nil {
					return "", err
				}
				if err := validateKeyLive(ctx, state, key, chain.ID, baseURL); err != nil {
					return "", err
				}
				// Re-read so a concurrent config edit is not clobbered, then check
				// for cancellation one last time: validation is the slow step, and a
				// key must never land on disk after the user has backed out.
				latest, _, err := config.Load()
				if err != nil {
					return "", err
				}
				if err := ctx.Err(); err != nil {
					return "", err
				}
				latest.BaseURL = baseURL
				latest.DefaultChain = chain.Name
				config.StoreAPIKey(key, &latest)
				path, err := config.Save(latest)
				if err != nil {
					return "", err
				}
				savedPath, savedLabel = path, maskKey(key)
				return savedLabel, nil
			}

			ctx := cmd.Context()
			switch {
			case state.apiKey != "":
				// An explicit --api-key never prompts, so scripts keep working.
				if _, err := save(ctx, state.apiKey); err != nil {
					return err
				}
			case keyPromptTTY():
				if _, err := runKeyPrompt(ctx, cmd.InOrStdin(), cmd.ErrOrStderr(), save); err != nil {
					return err
				}
			default:
				key, err := readSecret("Etherscan API key: ", cmd.InOrStdin(), cmd.ErrOrStderr())
				if err != nil {
					return err
				}
				if _, err := save(ctx, key); err != nil {
					return err
				}
			}
			fmt.Fprintln(cmd.OutOrStdout(), loginSavedLine(savedPath, savedLabel, stdoutIsTTY()))
			return nil
		},
	}
}

func logoutCommand(state *globalState) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the stored API key",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := config.Load()
			if err != nil {
				return err
			}
			removed := config.DeleteAPIKey(&cfg)
			if _, err := config.Save(cfg); err != nil {
				return err
			}
			if removed {
				fmt.Fprintln(os.Stdout, "Logged out (API key removed).")
			} else {
				fmt.Fprintln(os.Stdout, "No stored API key.")
			}
			if os.Getenv("ETHERSCAN_API_KEY") != "" {
				fmt.Fprintln(os.Stderr, "note: ETHERSCAN_API_KEY is still set and overrides stored keys; unset it in your shell to fully sign out.")
			}
			return nil
		},
	}
}

func uninstallCommand(state *globalState, updates updateManager) *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove Etherscan CLI and its saved configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			method := updates.DetectMethod()
			prompt, err := uninstallPrompt(method)
			if err != nil {
				return err
			}
			if !state.yes {
				if err := confirm(cmd.Context(), prompt, cmd.InOrStdin(), cmd.ErrOrStderr()); err != nil {
					return err
				}
			}
			background, err := updates.Uninstall(cmd.Context(), method, cmd.OutOrStdout(), cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			if background {
				fmt.Fprintln(cmd.OutOrStdout(), "Uninstall scheduled; removal will finish after this process exits.")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Etherscan CLI uninstalled.")
			}
			if method == updater.MethodScript {
				fmt.Fprintln(cmd.ErrOrStderr(), "note: manually created aliases or symlinks may still need to be removed.")
			}
			if os.Getenv("ETHERSCAN_API_KEY") != "" {
				fmt.Fprintln(cmd.ErrOrStderr(), "note: ETHERSCAN_API_KEY is still set in your environment; unset it in your shell to fully remove the key.")
			}
			return nil
		},
	}
}

func uninstallPrompt(method string) (string, error) {
	configPath, err := config.DefaultPath()
	if err != nil {
		return "", err
	}
	var action string
	switch method {
	case updater.MethodHomebrew:
		action = "  run     brew uninstall etherscan/etherscan-cli/etherscan"
	case updater.MethodNPM:
		packageName, err := updater.NPMPackageName()
		if err != nil {
			return "", err
		}
		action = "  run     npm uninstall -g " + packageName
	case updater.MethodScript:
		executable, err := os.Executable()
		if err != nil {
			return "", fmt.Errorf("locate current executable: %w", err)
		}
		action = fmt.Sprintf("  binary  %s\n  PATH    removed only with installer provenance and no shared files", executable)
	default:
		return "", fmt.Errorf("unsupported uninstall method %q", method)
	}
	return fmt.Sprintf("This will remove Etherscan CLI:\n%s\n  config  %s\n\nProceed?", action, filepath.Dir(configPath)), nil
}

func configCommand(state *globalState) *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "Manage CLI configuration"}
	cmd.AddCommand(&cobra.Command{Use: "list", Short: "Show configuration", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, path, err := config.Load()
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, "config:", path)
		key, src := config.GetAPIKey(cfg)
		fmt.Fprintf(os.Stdout, "default_chain=%s\n", cfg.DefaultChain)
		// Print the effective format: DefaultOutput is empty unless explicitly set.
		fmt.Fprintf(os.Stdout, "default_output=%s\n", firstNonEmpty(cfg.DefaultOutput, string(output.DefaultFormat)))
		fmt.Fprintf(os.Stdout, "base_url=%s\n", cfg.BaseURL)
		fmt.Fprintf(os.Stdout, "api_key=%s (%s)\n", config.Redact(key), src)
		return nil
	}})
	cmd.AddCommand(&cobra.Command{Use: "set key=value", Short: "Set a config value", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _, err := config.Load()
		if err != nil {
			return err
		}
		if err := config.Set(&cfg, args[0]); err != nil {
			return err
		}
		path, err := config.Save(cfg)
		if err == nil {
			fmt.Fprintln(os.Stdout, path)
		}
		return err
	}})
	cmd.AddCommand(&cobra.Command{Use: "get key", Short: "Get a config value", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _, err := config.Load()
		if err != nil {
			return err
		}
		switch args[0] {
		case "api_key":
			key, _ := config.GetAPIKey(cfg)
			fmt.Fprintln(os.Stdout, config.Redact(key))
		case "default_chain":
			fmt.Fprintln(os.Stdout, cfg.DefaultChain)
		case "default_output":
			fmt.Fprintln(os.Stdout, firstNonEmpty(cfg.DefaultOutput, string(output.DefaultFormat)))
		case "base_url":
			fmt.Fprintln(os.Stdout, cfg.BaseURL)
		default:
			return fmt.Errorf("unknown config key %q", args[0])
		}
		return nil
	}})
	return cmd
}

func chainsCommand(state *globalState) *cobra.Command {
	return &cobra.Command{Use: "chains", Short: "List supported chains", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		format, err := chainsFormat(state)
		if err != nil {
			return err
		}
		rows := make([]map[string]string, 0, len(chains.All()))
		for _, c := range chains.All() {
			freeTier := "available"
			if !c.FreeTier {
				freeTier = "paid only"
			}
			rows = append(rows, map[string]string{
				"id":        c.ID,
				"name":      c.DisplayName,
				"slug":      c.Name,
				"free_tier": freeTier,
				"testnet":   fmt.Sprint(c.Testnet),
				"symbol":    c.Symbol,
				"explorer":  c.Explorer,
			})
		}
		return output.WriteRows(os.Stdout, rows, format, []string{"id", "name", "slug", "free_tier", "testnet", "symbol", "explorer"})
	}}
}

func whoamiCommand(state *globalState) *cobra.Command {
	return &cobra.Command{Use: "whoami", Short: "Show the active chain and saved API key", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _, err := config.Load()
		if err != nil {
			return err
		}
		chain, err := chains.Resolve(firstNonEmpty(state.chain, os.Getenv("ETHERSCAN_CHAIN"), cfg.DefaultChain, "ethereum"))
		if err != nil {
			return err
		}
		key := state.apiKey
		if key == "" {
			key, _ = config.GetAPIKey(cfg)
		}
		keyDisplay := "(none — run 'etherscan login')"
		if key != "" {
			keyDisplay = maskKey(key)
		}
		fmt.Fprintf(os.Stdout, "chain:   %s (%s)\napi key: %s\n", chain.Name, chain.ID, keyDisplay)
		fmt.Fprintln(os.Stderr, "(run 'etherscan apilimit' for credit usage)")
		return nil
	}}
}

func versionCommand(info BuildInfo) *cobra.Command {
	return &cobra.Command{Use: "version", Short: "Print version", Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stdout, info.Version)
	}}
}

func updateCommand(info BuildInfo, updates updateManager) *cobra.Command {
	var method string
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update Etherscan CLI to the latest stable release",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if method != "" && !updater.ValidMethod(method) {
				return fmt.Errorf("unsupported update method %q (use homebrew, npm, or script)", method)
			}
			result, err := updates.Check(cmd.Context(), info.Version, true)
			if err != nil {
				return err
			}
			if !result.UpdateAvailable {
				fmt.Fprintf(cmd.OutOrStdout(), "Etherscan CLI %s is already up to date.\n", result.Current)
				return nil
			}
			detectedMethod := updates.DetectMethod()
			if detectedMethod == updater.MethodNPM {
				fmt.Fprintln(cmd.OutOrStdout(), "This installation is managed by npm. Run:")
				fmt.Fprintln(cmd.OutOrStdout(), "  npm install -g @etherscan/cli@latest")
				return nil
			}
			if method == "" {
				method = detectedMethod
			}
			if method == updater.MethodNPM {
				fmt.Fprintln(cmd.OutOrStdout(), "This installation is managed by npm. Run:")
				fmt.Fprintln(cmd.OutOrStdout(), "  npm install -g @etherscan/cli@latest")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updating Etherscan CLI %s -> %s using %s...\n", result.Current, result.Latest, method)
			background, err := updates.Upgrade(cmd.Context(), method, result.Latest, cmd.OutOrStdout(), cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			if background {
				fmt.Fprintln(cmd.OutOrStdout(), "The update will finish after this process exits.")
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Etherscan CLI %s installed. Restart the CLI to use it.\n", result.Latest)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&method, "method", "", "update method: homebrew, npm, or script")
	return cmd
}

func offerUpdate(ctx context.Context, updates updateManager, current string, in io.Reader, out, errOut io.Writer) (bool, error) {
	result, err := updates.Check(ctx, current, false)
	if err != nil || !result.UpdateAvailable {
		return false, nil
	}
	fmt.Fprintf(out, "\nUpdate available! %s -> %s\n", result.Current, result.Latest)
	if result.ReleaseURL != "" {
		fmt.Fprintf(out, "Release notes: %s\n", result.ReleaseURL)
	}
	fmt.Fprintln(out, "\n1. Update now")
	fmt.Fprintln(out, "2. Later")
	fmt.Fprintln(out, "3. Skip this version")
	fmt.Fprint(out, "\nChoose [2]: ")
	choice, readErr := bufio.NewReader(in).ReadString('\n')
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return false, nil
	}
	// Enter (empty) defaults to Later so a reflexive keypress on the way into the
	// explorer never kicks off a self-update; only an explicit "1" updates.
	switch strings.TrimSpace(choice) {
	case "1":
		method := updates.DetectMethod()
		if method == updater.MethodNPM {
			fmt.Fprintln(out, "This installation is managed by npm. Run:")
			fmt.Fprintln(out, "  npm install -g @etherscan/cli@latest")
			return true, nil
		}
		fmt.Fprintf(out, "Updating with %s...\n", method)
		background, err := updates.Upgrade(ctx, method, result.Latest, out, errOut)
		if err != nil {
			return true, err
		}
		if background {
			fmt.Fprintln(out, "The update will finish after this process exits.")
		} else {
			fmt.Fprintf(out, "Etherscan CLI %s installed. Restart the CLI to use it.\n", result.Latest)
		}
		return true, nil
	case "3":
		if err := updates.Skip(result.Latest); err != nil {
			return false, nil
		}
		fmt.Fprintf(out, "Skipped Etherscan CLI %s. You will be notified about the next release.\n\n", result.Latest)
		return false, nil
	default:
		fmt.Fprintln(out)
		return false, nil
	}
}

func tuiCommand(state *globalState, info BuildInfo, updates updateManager) *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch the interactive explorer",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !interactiveTTY() {
				return errors.New("tui requires an interactive terminal")
			}
			exit, err := offerUpdate(cmd.Context(), updates, info.Version, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
			if err != nil || exit {
				return err
			}
			return launchTUI(cmd.Context(), state, info)
		},
	}
}

// interactiveTTY reports whether both stdin (for keypresses) and stdout (for the
// screen) are terminals. The TUI needs both; if either is redirected/piped we
// fall back to the text splash so nothing hangs against a non-interactive stream.
func interactiveTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

// launchTUI resolves the runtime once and hands the interactive explorer the
// endpoint list plus an executor that reuses the existing client/call path. An
// empty key is allowed here so first-time users can explore locally; the TUI asks
// for and validates a key only when an API-backed endpoint is submitted.
func launchTUI(ctx context.Context, state *globalState, info BuildInfo) error {
	cfg, _, err := config.Load()
	if err != nil {
		return err
	}
	key := resolveKey(state, cfg)
	rt, err := buildRuntime(state, cfg, key)
	if err != nil {
		return err
	}
	baseURL := firstNonEmpty(state.baseURL, os.Getenv("ETHERSCAN_BASE_URL"), cfg.BaseURL, client.DefaultBaseURL)
	keyLabel := "none"
	if key != "" {
		keyLabel = maskKey(key)
	}
	eps, index := tuiEndpoints()
	// rt is mutable so the chain switcher can rebind the client mid-session; the
	// executor and validator capture &rt and therefore see the switched chain.
	switchChain := func(nameOrID string) (string, string, error) {
		chain, err := rebindRuntimeChain(&rt, nameOrID)
		if err != nil {
			return "", "", err
		}
		return chain.DisplayName, chain.ID, nil
	}
	saveKey := func(ctx context.Context, key string) (string, error) {
		key = strings.TrimSpace(key)
		if key == "" {
			return "", errors.New("empty API key")
		}
		if err := checkKeyShape(key); err != nil {
			return "", err
		}
		if err := validateKeyLive(ctx, state, key, rt.chain.ID, baseURL); err != nil {
			return "", err
		}
		latest, _, err := config.Load()
		if err != nil {
			return "", err
		}
		latest.BaseURL = baseURL
		latest.DefaultChain = rt.chain.Name
		config.StoreAPIKey(key, &latest)
		if _, err := config.Save(latest); err != nil {
			return "", err
		}
		rt.client = rt.client.WithAPIKey(key)
		return maskKey(key), nil
	}
	return tui.Run(ctx, tui.Config{
		Endpoints:   eps,
		Exec:        tuiExec(&rt, index),
		Validate:    tuiValidate(&rt, index),
		ChainName:   rt.chain.DisplayName,
		ChainID:     rt.chain.ID,
		KeyLabel:    keyLabel,
		HasAPIKey:   key != "",
		SaveAPIKey:  saveKey,
		Chains:      tuiChains(),
		SwitchChain: switchChain,
	})
}

// tuiChains maps the chain registry into the TUI's display list for the switcher.
func tuiChains() []tui.ChainInfo {
	all := chains.All()
	out := make([]tui.ChainInfo, 0, len(all))
	for _, c := range all {
		out = append(out, tui.ChainInfo{
			Name: c.Name, DisplayName: c.DisplayName, ID: c.ID,
			Aliases: append([]string(nil), c.Aliases...), Testnet: c.Testnet, PaidOnly: !c.FreeTier,
		})
	}
	return out
}

// tuiValidate builds the pre-call guard shared by the TUI form (inline errors on
// submit) and the executor: the mainnet-only check and validateParams — the SAME
// guards the normal CLI path (endpointCommand RunE) applies, so the TUI cannot
// bypass the validation the CLI enforces (e.g. empty comma-list entries).
// chainlist has no module/action or params on the wire, so it passes trivially.
func tuiValidate(rt *resolvedRuntime, index map[string]EndpointSpec) func(module, action string, params map[string]string) error {
	return func(module, action string, params map[string]string) error {
		if module == "getapilimit" && action == "chainlist" {
			return nil
		}
		spec, ok := index[module+"/"+action]
		if !ok {
			return fmt.Errorf("unknown endpoint %s/%s", module, action)
		}
		if err := validateEndpointChain(spec, rt.chain); err != nil {
			return err
		}
		return validateParams(spec, params)
	}
}

func validateEndpointChain(spec EndpointSpec, chain chains.Chain) error {
	if spec.MainnetOnly && !chains.IsMainnetID(chain.ID) {
		return fmt.Errorf("%s/%s is only supported on Ethereum mainnet", spec.Module, spec.Action)
	}
	if len(spec.AllowedChainIDs) == 0 {
		return nil
	}
	for _, id := range spec.AllowedChainIDs {
		if chain.ID == id {
			return nil
		}
	}
	names := make([]string, len(spec.AllowedChainIDs))
	for i, id := range spec.AllowedChainIDs {
		if allowed, err := chains.Resolve(id); err == nil {
			names[i] = allowed.DisplayName
		} else {
			names[i] = id
		}
	}
	return fmt.Errorf("etherscan contract %s is only supported on %s", strings.Fields(spec.Use)[0], strings.Join(names, " and "))
}

// tuiExec builds the explorer's executor. It re-runs tuiValidate before issuing
// the request (defense in depth — the form validates on submit, but the executor
// must not rely on it).
func tuiExec(rt *resolvedRuntime, index map[string]EndpointSpec) tui.Exec {
	validate := tuiValidate(rt, index)
	return func(ctx context.Context, module, action string, params map[string]string) (json.RawMessage, error) {
		if err := validate(module, action, params); err != nil {
			return nil, err
		}
		if module == "getapilimit" && action == "chainlist" {
			res, err := rt.client.ChainList(ctx)
			if err != nil {
				return nil, err
			}
			return res.Raw, nil
		}
		res, err := call(ctx, rt.client, index[module+"/"+action], params)
		if err != nil {
			return nil, err
		}
		return res.Raw, nil
	}
}

// tuiModuleOrder is the Etherscan docs order (https://docs.etherscan.io/introduction).
// The TUI sidebar follows it so users can cross-reference the docs; the CLI command
// tree is unaffected.
var tuiModuleOrder = []string{
	"account", "block", "contract", "gastracker", "proxy", "logs",
	"stats", "transaction", "token", "nametag", "usage",
}

// tuiModuleGroup maps wire modules to the docs nav group they appear under when
// the two differ. The TUI sidebar shows the docs group name; requests and result
// headers keep the wire module.
var tuiModuleGroup = map[string]string{
	"getapilimit": "usage",
}

// tuiActionOrder is the docs sidebar order WITHIN each module (scraped from the
// rendered nav on docs.etherscan.io, 2026-07-11). Actions whose docs page lives in
// a different docs group than their API module (token balances, L2 transfers,
// daily-series stats) are appended after the module's own group, in their group's
// order. Actions not listed here (e.g. undocumented stats series) sink to the end
// of their module, keeping registry order among themselves.
var tuiActionOrder = map[string][]string{
	"account": {
		"balance", "balancemulti", "balancehistory", "txlist", "tokentx",
		"tokennfttx", "token1155tx", "txlistinternal", "getminedblocks",
		"txsBeaconWithdrawal", "fundedby",
		// tokens docs group
		"tokenbalance", "tokenbalancehistory", "addresstokenbalance",
		"addresstokennftbalance", "addresstokennftinventory",
		// L2 deposits/withdrawals docs group
		"txnbridge", "getdeposittxs", "getwithdrawaltxs",
	},
	"block":      {"getblockreward", "getblocktxnscount", "getblockcountdown", "getblocknobytime"},
	"contract":   {"getabi", "getsourcecode", "getcontractcreation", "verifysourcecode", "checkverifystatus", "verifyproxycontract", "checkproxyverification"},
	"gastracker": {"gasestimate", "gasoracle"},
	"proxy": {
		"eth_blockNumber", "eth_getBlockByNumber", "eth_getUncleByBlockNumberAndIndex",
		"eth_getBlockTransactionCountByNumber", "eth_getTransactionByHash",
		"eth_getTransactionByBlockNumberAndIndex", "eth_getTransactionCount",
		"eth_sendRawTransaction", "eth_getTransactionReceipt", "eth_call",
		"eth_getCode", "eth_getStorageAt", "eth_gasPrice", "eth_estimateGas",
	},
	"stats": {
		"ethsupply", "ethsupply2", "ethprice", "chainsize", "nodecount",
		"dailytxnfee", "dailynewaddress", "dailynetutilization", "dailyavghashrate",
		"dailytx", "dailyavgnetdifficulty", "ethdailyprice",
		// blocks docs group
		"dailyavgblocksize", "dailyblkcount", "dailyblockrewards",
		"dailyavgblocktime", "dailyuncleblkcount",
		// gas tracker docs group
		"dailyavggaslimit", "dailygasused", "dailyavggasprice",
		// tokens docs group
		"tokensupply", "tokensupplyhistory",
	},
	"transaction": {"getstatus", "gettxreceiptstatus"},
	"token":       {"topholders", "tokenholderlist", "tokenholdercount", "tokeninfo"},
	"usage":       {"getapilimit", "chainlist"},
}

// tuiEndpoints adapts the registry into the tui package's own types, excluding
// write/sensitive actions so the explorer stays read-only. Endpoints are titled
// by their API action (not the friendly CLI alias — the TUI is an API explorer)
// and ordered by docs module order. It returns the browsable list plus an index
// for the executor to look specs up by module/action.
func tuiEndpoints() ([]tui.Endpoint, map[string]EndpointSpec) {
	var list []tui.Endpoint
	index := map[string]EndpointSpec{}
	for _, spec := range endpoints() {
		if spec.Post || spec.Sensitive {
			continue
		}
		var params []tui.Param
		for _, pr := range spec.Params {
			label := pr.Usage
			if label == "" {
				label = pr.Name
			}
			params = append(params, tui.Param{Name: pr.Name, Label: label, Required: pr.Required})
		}
		list = append(list, tui.Endpoint{
			Module:    spec.Module,
			Action:    spec.Action,
			Title:     spec.Action,
			Desc:      spec.Short,
			Params:    params,
			Columns:   spec.Columns,
			Paginated: spec.Paginated,
			Group:     tuiModuleGroup[spec.Module],
		})
		index[spec.Module+"/"+spec.Action] = spec
	}
	// chainlist is the one endpoint with no module/action on the wire (dedicated
	// /v2/chainlist URL). It is placed in the usage group for navigation only —
	// tuiExec routes it to client.ChainList, never through the spec index.
	list = append(list, tui.Endpoint{
		Module:  "getapilimit",
		Action:  "chainlist",
		Title:   "chainlist",
		Desc:    "List all supported Etherscan chains",
		Columns: []string{"chainname", "chainid", "blockexplorer", "status"},
		Bare:    true,
		Group:   tuiModuleGroup["getapilimit"],
	})
	groupOf := func(e tui.Endpoint) string {
		if e.Group != "" {
			return e.Group
		}
		return e.Module
	}
	rank := func(group string) int {
		for i, m := range tuiModuleOrder {
			if m == group {
				return i
			}
		}
		return len(tuiModuleOrder) // unknown modules sink to the end
	}
	actionRank := func(group, action string) int {
		order := tuiActionOrder[group]
		for i, a := range order {
			if a == action {
				return i
			}
		}
		return len(order) // unlisted actions sink to the end of their module
	}
	sort.SliceStable(list, func(i, j int) bool {
		gi, gj := groupOf(list[i]), groupOf(list[j])
		if mi, mj := rank(gi), rank(gj); mi != mj {
			return mi < mj
		}
		return actionRank(gi, list[i].Action) < actionRank(gj, list[j].Action)
	})
	return list, index
}

// printSplash is shown on a bare `etherscan` invocation: the branded Quick Start
// guide instead of dumping the entire auto-generated command tree. It is coloured
// when stdout is a real terminal and plain when piped (CI, agents), so redirected
// output stays free of escape codes.
func printSplash(w io.Writer, info BuildInfo) {
	width, height := stdoutSize()
	fmt.Fprintln(w, renderSplash(info, stdoutIsTTY(), width, height))
}

// renderSplash adds a single breathing row before the artwork at an interactive
// terminal. Redirected output stays byte-compatible with the existing plain
// Quick Start render and does not gain a leading blank line.
func renderSplash(info BuildInfo, interactive bool, width, height int) string {
	splash := renderQuickStart(info, interactive, width, height)
	if interactive {
		return "\n" + splash
	}
	return splash
}

// stdoutIsTTY reports whether standard output is an interactive terminal, used to
// decide whether the Quick Start banner should be coloured.
func stdoutIsTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// stdoutSize returns the terminal width and height of standard output, or 0, 0
// when it is not a terminal or the size cannot be determined. Callers treat an
// unknown size as "render everything": piped output has no viewport to overflow.
func stdoutSize() (width, height int) {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 0, 0
	}
	return w, h
}

// checkKeyShape fast-fails a key containing whitespace (a clear paste mistake) before
// any network call. Length is deliberately not checked: the server validates the key,
// so a doubled paste is rejected there without coupling the CLI to Etherscan's key format.
func checkKeyShape(key string) error {
	if strings.ContainsAny(key, " \t\r\n") {
		return errors.New("API key looks malformed (contains whitespace)")
	}
	return nil
}

// maskKey renders a key for display: first 3 + last 3 characters, asterisks between.
// Keys of 6 or fewer characters are fully masked.
func maskKey(key string) string {
	if len(key) <= 6 {
		return strings.Repeat("*", len(key))
	}
	return key[:3] + strings.Repeat("*", len(key)-6) + key[len(key)-3:]
}

// keyPromptTTY reports whether the branded key prompt can be drawn. It needs
// stdin for keypresses and stderr for its frames; stdout is required too so that
// `etherscan login > file` falls back to the plain reader rather than rendering
// a full-colour prompt against a half-redirected session. Setting
// ETHERSCAN_PLAIN_PROMPT (or TERM=dumb) forces the fallback.
func keyPromptTTY() bool {
	if os.Getenv("ETHERSCAN_PLAIN_PROMPT") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	return term.IsTerminal(int(os.Stdin.Fd())) &&
		term.IsTerminal(int(os.Stdout.Fd())) &&
		term.IsTerminal(int(os.Stderr.Fd()))
}

// loginSavedLine renders the post-login confirmation. The wording is unchanged
// so anything scraping this output keeps working, and the plain form is
// byte-identical to previous releases; a check mark and brand colour are added
// only when stdout is a terminal.
func loginSavedLine(path, maskedKey string, color bool) string {
	plain := fmt.Sprintf("API Key saved to %s! Key: %s", path, maskedKey)
	if !color {
		return plain
	}
	return ansiColor(brand.GreenHex, "✓", false) + " API Key saved to " +
		ansiColor(brand.DimHex, path, false) + "! Key: " +
		ansiColor(brand.AccentHex, maskedKey, false)
}

// readSecret reads a secret from in. When in is an interactive terminal it echoes
// a "*" per character so the user gets visual confirmation of input (without
// revealing the value); piped/non-interactive input is read as a normal line so
// automation still works. The prompt and asterisks go to out (stderr) to keep
// stdout clean.
//
// This is the fallback for every case the branded prompt cannot serve: piped
// stdin, a redirected stream, TERM=dumb, and terminals such as Git Bash's MSYS
// pty where IsTerminal reports false. The raw-mode masking half matters there —
// without it the terminal would echo the key in clear text.
func readSecret(prompt string, in io.Reader, out io.Writer) (string, error) {
	fmt.Fprint(out, prompt)
	f, isFile := in.(*os.File)
	if !isFile || !term.IsTerminal(int(f.Fd())) {
		line, err := bufio.NewReader(in).ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}
		return strings.TrimSpace(line), nil
	}
	fd := int(f.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer term.Restore(fd, oldState)
	r := bufio.NewReader(in)
	var buf []byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		switch {
		case b == '\r' || b == '\n':
			fmt.Fprint(out, "\r\n")
			return strings.TrimSpace(string(buf)), nil
		case b == 3: // Ctrl-C
			fmt.Fprint(out, "\r\n")
			return "", errKeyPromptCanceled
		case b == 8 || b == 127: // Backspace / Delete
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				fmt.Fprint(out, "\b \b")
			}
		case b < 32: // ignore other control bytes
		default:
			buf = append(buf, b)
			fmt.Fprint(out, "*")
		}
	}
}

func completionCommand(root *cobra.Command) *cobra.Command {
	return &cobra.Command{Use: "completion [bash|zsh|fish|powershell]", Short: "Generate shell completion", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return root.GenBashCompletion(os.Stdout)
		case "zsh":
			return root.GenZshCompletion(os.Stdout)
		case "fish":
			return root.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return root.GenPowerShellCompletion(os.Stdout)
		default:
			return fmt.Errorf("unsupported shell %q", args[0])
		}
	}}
}

func requiredParams(spec EndpointSpec) []ParamSpec {
	var out []ParamSpec
	for _, p := range spec.Params {
		if p.Arg {
			out = append(out, p)
		}
	}
	return out
}

func isPositional(spec EndpointSpec, param ParamSpec) bool {
	for _, p := range requiredParams(spec) {
		if p.Name == param.Name {
			return true
		}
	}
	return false
}

func validateParams(spec EndpointSpec, params map[string]string) error {
	for _, p := range spec.Params {
		v := params[p.Name]
		if p.Required && strings.TrimSpace(v) == "" {
			return fmt.Errorf("missing required %s", p.Name)
		}
		switch p.Kind {
		case KindAddress:
			if err := client.ValidateAddress(p.Name, v); err != nil {
				return err
			}
		case KindAddresses:
			maxList := p.MaxList
			if maxList == 0 {
				maxList = 20
			}
			if err := client.ValidateCommaAddresses(p.Name, v, maxList); err != nil {
				return err
			}
		case KindTxHash:
			if err := client.ValidateTxHash(p.Name, v); err != nil {
				return err
			}
		case KindUint:
			if err := client.ValidateUint(p.Name, v); err != nil {
				return err
			}
		case KindDate:
			if err := client.ValidateDate(p.Name, v); err != nil {
				return err
			}
		case KindSort:
			if err := client.ValidateSort(v); err != nil {
				return err
			}
		case KindHex:
			if err := client.ValidateHex(p.Name, v); err != nil {
				return err
			}
		case KindZeroOne:
			if v != "" && v != "0" && v != "1" {
				return fmt.Errorf("%s must be 0 or 1", p.Name)
			}
		case KindConstructorArgs:
			normalized, err := normalizeConstructorArguments(v)
			if err != nil {
				return err
			}
			params[p.Name] = normalized
		case KindLicense:
			if err := validateLicenseType(v); err != nil {
				return err
			}
		}
	}
	if spec.Module == "contract" && spec.Action == "verifysourcecode" {
		if err := validateSourceVerification(spec, params); err != nil {
			return err
		}
	}
	if len(spec.RequireOneOf) > 0 {
		any := false
		for _, name := range spec.RequireOneOf {
			if strings.TrimSpace(params[name]) != "" {
				any = true
				break
			}
		}
		if !any {
			return fmt.Errorf("at least one of %s is required", strings.Join(spec.RequireOneOf, ", "))
		}
	}
	if spec.AdvancedFilter {
		if err := validateAdvancedFilter(params); err != nil {
			return err
		}
	}
	return nil
}

// validatePagination enforces the API's page/offset pairing rule. The Etherscan
// API only paginates when BOTH page and offset are present (and numeric > 0); given
// just one it silently ignores it and returns the full default window. Applies only
// to endpoints that declare both params. Not enforced under --all, where runAllPages
// supplies page itself and --offset is the page size — so it is called only on the
// single-call path.
func validatePagination(spec EndpointSpec, params map[string]string) error {
	if !spec.Paginated || !hasParam(spec, "page") || !hasParam(spec, "offset") {
		return nil
	}
	pageSet := strings.TrimSpace(params["page"]) != ""
	offsetSet := strings.TrimSpace(params["offset"]) != ""
	if pageSet != offsetSet {
		return errors.New("--page and --offset must be used together.")
	}
	return nil
}

// hasParam reports whether the spec declares a parameter with the given name.
func hasParam(spec EndpointSpec, name string) bool {
	for _, p := range spec.Params {
		if p.Name == name {
			return true
		}
	}
	return false
}

// validateAdvancedFilter enforces the from/to/fromto_opr rules and normalizes the
// operator to the upper-case AND/OR the API expects. No-op when neither is set.
func validateAdvancedFilter(params map[string]string) error {
	from := strings.TrimSpace(params["from"])
	to := strings.TrimSpace(params["to"])
	if from == "" && to == "" {
		return nil
	}
	// The server rejects address combined with the from/to filter mode.
	if strings.TrimSpace(params["address"]) != "" {
		return errors.New("address cannot be combined with --from/--to filters")
	}
	opr := strings.ToUpper(strings.TrimSpace(params["fromto_opr"]))
	if opr != "AND" && opr != "OR" {
		return errors.New("--fromto-opr is required and must be 'and' or 'or' when --from or --to is set")
	}
	if opr == "AND" && (from == "" || to == "") {
		return errors.New("--from and --to are both required when --fromto-opr is 'and'")
	}
	params["fromto_opr"] = opr
	return nil
}

func populateFileParam(state *globalState, params map[string]string) error {
	if state.file == "" || params["sourceCode"] != "" {
		return nil
	}
	file, err := os.Open(state.file)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxVerificationSourceBytes+1))
	if err != nil {
		return err
	}
	if len(data) > maxVerificationSourceBytes {
		return fmt.Errorf("verification source exceeds the %d-byte limit", maxVerificationSourceBytes)
	}
	if params["sourceCode"] == "" {
		params["sourceCode"] = string(data)
	}
	return nil
}

const (
	maxVerificationSourceBytes  = 3_000_000
	maxConstructorArgumentChars = 250_000
)

func normalizeConstructorArguments(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
		value = value[2:]
	}
	if len(value) > maxConstructorArgumentChars {
		return "", fmt.Errorf("constructorArguments exceeds the %d-character limit", maxConstructorArgumentChars)
	}
	if len(value)%2 != 0 {
		return "", errors.New("constructorArguments must contain an even number of hexadecimal characters")
	}
	if _, err := hex.DecodeString(value); err != nil {
		return "", errors.New("constructorArguments must be hexadecimal without a 0x prefix")
	}
	return value, nil
}

func validateLicenseType(value string) error {
	if value == "" {
		return nil
	}
	license, err := strconv.Atoi(value)
	if err != nil || license < 1 || license > 14 {
		return errors.New("licenseType must be an integer from 1 to 14")
	}
	return nil
}

func validateSourceVerification(spec EndpointSpec, params map[string]string) error {
	if len(params["sourceCode"]) > maxVerificationSourceBytes {
		return fmt.Errorf("verification source exceeds the %d-byte limit", maxVerificationSourceBytes)
	}
	format := params["codeformat"]
	allowed := false
	for _, candidate := range spec.AllowedCodeFormats {
		if format == candidate {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("unsupported codeformat %q for etherscan contract %s", format, strings.Fields(spec.Use)[0])
	}
	switch format {
	case "solidity-single-file", "vyper-json":
		if params["optimizationUsed"] == "" {
			return fmt.Errorf("missing required optimizationUsed for %s", format)
		}
	case "solidity-standard-json-input", "stylus":
	}
	return nil
}

func confirm(ctx context.Context, prompt string, input io.Reader, output io.Writer) error {
	fmt.Fprintf(output, "%s [y/N] ", prompt)
	done := make(chan string, 1)
	go func() {
		reader := bufio.NewReader(input)
		line, _ := reader.ReadString('\n')
		done <- strings.TrimSpace(strings.ToLower(line))
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case answer := <-done:
		if answer == "y" || answer == "yes" {
			return nil
		}
		return errors.New("cancelled")
	}
}

// chainsFormat resolves the output format for `chains` without going through
// runtime(): the chain list comes from the built-in registry, so the command must
// work before `etherscan login`. A config load failure degrades to the flag value
// or the built-in default rather than failing the listing.
func chainsFormat(state *globalState) (output.Format, error) {
	cfg, _, _ := config.Load()
	return output.ParseFormat(firstNonEmpty(state.out, cfg.DefaultOutput, string(output.DefaultFormat)))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func flagName(name string) string {
	replacer := strings.NewReplacer(
		"_", "-",
		"sourceCode", "source-code",
		"fromBlock", "from-block",
		"toBlock", "to-block",
		"gasPrice", "gas-price",
		"optimizationUsed", "optimization-used",
		"constructorArguments", "constructor-arguments",
		"evmVersion", "evm-version",
		"licenseType", "license-type",
		"zksolcVersion", "zksolc-version",
	)
	return replacer.Replace(name)
}

func legacyFlagName(name string) string {
	switch name {
	case "optimizationUsed", "constructorArguments", "evmVersion", "licenseType", "zksolcVersion":
		return name
	default:
		return ""
	}
}

func atoi(value string) int {
	n, _ := strconv.Atoi(value)
	return n
}
