package cli

import (
	"bufio"
	"context"
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

	"github.com/etherscan/etherscan-cli/internal/chains"
	"github.com/etherscan/etherscan-cli/internal/client"
	"github.com/etherscan/etherscan-cli/internal/config"
	"github.com/etherscan/etherscan-cli/internal/output"
	"github.com/etherscan/etherscan-cli/internal/tui"
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
	json     bool
	compact  bool
	csv      bool
	timeout  time.Duration
	rate     float64
	verbose  bool
	debug    bool
	yes      bool
	file     string
	maxPages int
	all      bool
}

func NewRootCommand(info BuildInfo) *cobra.Command {
	state := &globalState{timeout: 30 * time.Second, rate: 3, maxPages: 20}
	root := &cobra.Command{
		Use:           "etherscan",
		Short:         "Command-line client for the Etherscan V2 API",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// A bare invocation at an interactive terminal opens the explorer; when
			// piped/redirected (agents, scripts, CI) it prints a plain text splash so
			// nothing hangs waiting for keypresses.
			if interactiveTTY() {
				return launchTUI(cmd.Context(), state, info)
			}
			printSplash(cmd.OutOrStdout(), info)
			return nil
		},
	}
	root.PersistentFlags().StringVar(&state.apiKey, "api-key", "", "API key for this command (overrides login/ETHERSCAN_API_KEY)")
	root.PersistentFlags().StringVar(&state.apiKey, "apikey", "", "alias for --api-key")
	root.PersistentFlags().StringVar(&state.chain, "chain", "", "chain name or chainid")
	root.PersistentFlags().StringVar(&state.baseURL, "base-url", "", "API base URL")
	root.PersistentFlags().StringVarP(&state.out, "output", "o", "", "output format: table, json, csv")
	root.PersistentFlags().BoolVar(&state.json, "json", false, "print raw result as JSON")
	root.PersistentFlags().BoolVar(&state.compact, "compact", false, "compact JSON output")
	root.PersistentFlags().BoolVar(&state.csv, "csv", false, "print result as CSV")
	root.PersistentFlags().DurationVar(&state.timeout, "timeout", 30*time.Second, "request timeout")
	root.PersistentFlags().Float64Var(&state.rate, "rate-limit", 3, "client-side request rate limit per second (free-tier API V2 default; raise for higher tiers)")
	root.PersistentFlags().BoolVarP(&state.verbose, "verbose", "v", false, "log request URL and timing to stderr")
	root.PersistentFlags().BoolVar(&state.debug, "debug", false, "dump raw response bodies to stderr")
	root.PersistentFlags().BoolVar(&state.yes, "yes", false, "skip confirmation for sensitive submit actions")
	root.PersistentFlags().BoolVar(&state.all, "all", false, "auto-paginate list commands")
	root.PersistentFlags().IntVar(&state.maxPages, "max-pages", 20, "maximum pages for --all")
	hideFlags(root, "apikey", "base-url", "compact", "max-pages", "rate-limit", "timeout", "verbose", "debug", "yes")

	root.AddCommand(loginCommand(state), logoutCommand(state), uninstallCommand(state), configCommand(state), chainsCommand(), whoamiCommand(state), versionCommand(info), tuiCommand(state, info), completionCommand(root))
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
			if spec.MainnetOnly && !chains.IsMainnetID(rt.chain.ID) {
				return fmt.Errorf("%s/%s is only supported on Ethereum mainnet", spec.Module, spec.Action)
			}
			if spec.Sensitive && !state.yes {
				if err := confirm(cmd.Context(), fmt.Sprintf("Submit %s/%s?", spec.Module, spec.Action)); err != nil {
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
		cmd.Flags().String(flagName(param.Name), "", param.Usage)
	}
	if spec.Action == "verifysourcecode" {
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
	chainInput := firstNonEmpty(state.chain, os.Getenv("ETHERSCAN_CHAIN"), cfg.DefaultChain, "ethereum")
	chain, err := chains.Resolve(chainInput)
	if err != nil {
		return resolvedRuntime{}, err
	}
	baseURL := firstNonEmpty(state.baseURL, os.Getenv("ETHERSCAN_BASE_URL"), cfg.BaseURL, client.DefaultBaseURL)
	if !strings.HasPrefix(baseURL, "https://") {
		fmt.Fprintf(os.Stderr, "warning: non-HTTPS base URL: %s\n", baseURL)
	}
	format := output.Format(firstNonEmpty(outputFlag(state), cfg.DefaultOutput, "table"))
	return resolvedRuntime{
		client: client.New(client.Options{BaseURL: baseURL, APIKey: key, ChainID: chain.ID, Timeout: state.timeout, RateLimit: state.rate, Verbose: state.verbose, Debug: state.debug, Stderr: os.Stderr}),
		format: format,
		chain:  chain,
	}, nil
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
			key := state.apiKey
			if key == "" {
				key, err = readSecret("Etherscan API key: ")
				if err != nil {
					return err
				}
			}
			if key == "" {
				return errors.New("empty API key")
			}
			if err := checkKeyShape(key); err != nil {
				return err
			}
			chain, err := chains.Resolve(firstNonEmpty(state.chain, cfg.DefaultChain, "ethereum"))
			if err != nil {
				return err
			}
			baseURL := firstNonEmpty(state.baseURL, cfg.BaseURL, client.DefaultBaseURL)
			if err := validateKeyLive(cmd.Context(), state, key, chain.ID, baseURL); err != nil {
				return err
			}
			cfg.BaseURL = baseURL
			cfg.DefaultChain = chain.Name
			config.StoreAPIKey(key, &cfg)
			path, err := config.Save(cfg)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "API Key saved to %s! Key: %s\n", path, maskKey(key))
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

func uninstallCommand(state *globalState) *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove all etherscan CLI configuration (API key and settings)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.DefaultPath()
			if err != nil {
				return err
			}
			dir := filepath.Dir(path)
			if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
				fmt.Fprintln(os.Stdout, "Nothing to remove; no configuration found.")
				return nil
			} else if err != nil {
				return err
			}
			if !state.yes {
				if err := confirm(cmd.Context(), fmt.Sprintf("Remove all configuration in %s?", dir)); err != nil {
					return err
				}
			}
			if err := os.RemoveAll(dir); err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "Removed %s\n", dir)
			if os.Getenv("ETHERSCAN_API_KEY") != "" {
				fmt.Fprintln(os.Stderr, "note: ETHERSCAN_API_KEY is still set in your environment; unset it in your shell to fully remove the key.")
			}
			return nil
		},
	}
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
		fmt.Fprintf(os.Stdout, "default_output=%s\n", cfg.DefaultOutput)
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
			fmt.Fprintln(os.Stdout, cfg.DefaultOutput)
		case "base_url":
			fmt.Fprintln(os.Stdout, cfg.BaseURL)
		default:
			return fmt.Errorf("unknown config key %q", args[0])
		}
		return nil
	}})
	return cmd
}

func chainsCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "chains", Short: "Manage supported chains"}
	cmd.AddCommand(&cobra.Command{Use: "list", Short: "List built-in chains", Run: func(cmd *cobra.Command, args []string) {
		rows := make([]map[string]string, 0, len(chains.All()))
		for _, c := range chains.All() {
			rows = append(rows, map[string]string{
				"id":       c.ID,
				"name":     c.Name,
				"symbol":   c.Symbol,
				"testnet":  fmt.Sprint(c.Testnet),
				"explorer": c.Explorer,
			})
		}
		_ = output.WriteRows(os.Stdout, rows, output.Table, []string{"id", "name", "symbol", "testnet", "explorer"})
	}})
	return cmd
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

func tuiCommand(state *globalState, info BuildInfo) *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch the interactive explorer",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !interactiveTTY() {
				return errors.New("tui requires an interactive terminal")
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
// endpoint list plus an executor that reuses the existing client/call path. With
// no key resolved it runs the first-launch setup screen first; a key saved there
// lands in the config file, so the runtime resolution below picks it up.
func launchTUI(ctx context.Context, state *globalState, info BuildInfo) error {
	cfg, _, err := config.Load()
	if err != nil {
		return err
	}
	if resolveKey(state, cfg) == "" {
		if err := runSetup(ctx, state); err != nil {
			return err
		}
		cfg, _, _ = config.Load()
	}
	rt, err := runtime(state)
	if err != nil {
		return err
	}
	keyLabel := "none"
	if key := resolveKey(state, cfg); key != "" {
		keyLabel = maskKey(key)
	}
	eps, index := tuiEndpoints()
	return tui.Run(ctx, tui.Config{
		Endpoints: eps,
		Exec:      tuiExec(rt, index),
		Validate:  tuiValidate(rt, index),
		ChainName: rt.chain.Name,
		ChainID:   rt.chain.ID,
		KeyLabel:  keyLabel,
	})
}

// runSetup runs the TUI first-launch key screen. Its Save closure applies the
// same shape check, live validation, and persistence as `etherscan login`, so a
// key accepted here behaves identically to one saved via login.
func runSetup(ctx context.Context, state *globalState) error {
	save := func(ctx context.Context, key string) error {
		key = strings.TrimSpace(key)
		if key == "" {
			return errors.New("empty API key")
		}
		if err := checkKeyShape(key); err != nil {
			return err
		}
		cfg, _, err := config.Load()
		if err != nil {
			return err
		}
		chain, err := chains.Resolve(firstNonEmpty(state.chain, cfg.DefaultChain, "ethereum"))
		if err != nil {
			return err
		}
		baseURL := firstNonEmpty(state.baseURL, cfg.BaseURL, client.DefaultBaseURL)
		if err := validateKeyLive(ctx, state, key, chain.ID, baseURL); err != nil {
			return err
		}
		cfg.BaseURL = baseURL
		cfg.DefaultChain = chain.Name
		config.StoreAPIKey(key, &cfg)
		_, err = config.Save(cfg)
		return err
	}
	if err := tui.RunSetup(ctx, tui.SetupConfig{Save: save}); err != nil {
		if errors.Is(err, tui.ErrSetupAborted) {
			return errNoAPIKey
		}
		return err
	}
	return nil
}

// tuiValidate builds the pre-call guard shared by the TUI form (inline errors on
// submit) and the executor: the mainnet-only check and validateParams — the SAME
// guards the normal CLI path (endpointCommand RunE) applies, so the TUI cannot
// bypass the validation the CLI enforces (e.g. empty comma-list entries).
// chainlist has no module/action or params on the wire, so it passes trivially.
func tuiValidate(rt resolvedRuntime, index map[string]EndpointSpec) func(module, action string, params map[string]string) error {
	return func(module, action string, params map[string]string) error {
		if module == "getapilimit" && action == "chainlist" {
			return nil
		}
		spec, ok := index[module+"/"+action]
		if !ok {
			return fmt.Errorf("unknown endpoint %s/%s", module, action)
		}
		if spec.MainnetOnly && !chains.IsMainnetID(rt.chain.ID) {
			return fmt.Errorf("%s/%s is only supported on Ethereum mainnet", spec.Module, spec.Action)
		}
		return validateParams(spec, params)
	}
}

// tuiExec builds the explorer's executor. It re-runs tuiValidate before issuing
// the request (defense in depth — the form validates on submit, but the executor
// must not rely on it).
func tuiExec(rt resolvedRuntime, index map[string]EndpointSpec) tui.Exec {
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

// printSplash is shown on a bare `etherscan` invocation: a short branded banner
// with a few example commands and a pointer to full help, instead of dumping the
// entire auto-generated command tree.
func printSplash(w io.Writer, info BuildInfo) {
	fmt.Fprintf(w, "Etherscan CLI %s\n", info.Version)
	fmt.Fprintln(w, "Command-line client for the Etherscan V2 API.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  etherscan login")
	fmt.Fprintln(w, "  etherscan account balance 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	fmt.Fprintln(w, "  etherscan gastracker oracle")
	fmt.Fprintln(w, "  etherscan --chain base account balance 0xADDRESS --json")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'etherscan --help' to see all commands.")
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

// readSecret reads a secret from stdin. On an interactive terminal it echoes a
// "*" per character so the user gets visual confirmation of input (without
// revealing the value); piped/non-interactive stdin is read as a normal line so
// automation still works. The prompt and asterisks go to stderr to keep stdout clean.
func readSecret(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}
		return strings.TrimSpace(line), nil
	}
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer term.Restore(fd, oldState)
	in := bufio.NewReader(os.Stdin)
	var buf []byte
	for {
		b, err := in.ReadByte()
		if err != nil {
			return "", err
		}
		switch {
		case b == '\r' || b == '\n':
			fmt.Fprint(os.Stderr, "\r\n")
			return strings.TrimSpace(string(buf)), nil
		case b == 3: // Ctrl-C
			fmt.Fprint(os.Stderr, "\r\n")
			return "", errors.New("cancelled")
		case b == 8 || b == 127: // Backspace / Delete
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				fmt.Fprint(os.Stderr, "\b \b")
			}
		case b < 32: // ignore other control bytes
		default:
			buf = append(buf, b)
			fmt.Fprint(os.Stderr, "*")
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
	if state.file == "" {
		return nil
	}
	data, err := os.ReadFile(state.file)
	if err != nil {
		return err
	}
	if params["sourceCode"] == "" {
		params["sourceCode"] = string(data)
	}
	return nil
}

func confirm(ctx context.Context, prompt string) error {
	fmt.Fprintf(os.Stderr, "%s [y/N] ", prompt)
	done := make(chan string, 1)
	go func() {
		reader := bufio.NewReader(os.Stdin)
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

func outputFlag(state *globalState) string {
	if state.json {
		return string(output.JSON)
	}
	if state.csv {
		return string(output.CSV)
	}
	return state.out
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
	replacer := strings.NewReplacer("_", "-", "sourceCode", "source-code", "fromBlock", "from-block", "toBlock", "to-block", "gasPrice", "gas-price")
	return replacer.Replace(name)
}

func atoi(value string) int {
	n, _ := strconv.Atoi(value)
	return n
}
