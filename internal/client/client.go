package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const DefaultBaseURL = "https://api.etherscan.io/v2/api"

var emptyMessages = []string{
	"no records found",
	"no transactions found",
	"no transaction found",
	"no data found",
}

type Options struct {
	BaseURL    string
	APIKey     string
	ChainID    string
	Timeout    time.Duration
	RateLimit  float64
	Retries    int
	Verbose    bool
	Debug      bool
	HTTPClient *http.Client
	Stderr     io.Writer
}

type Client struct {
	baseURL string
	apiKey  string
	chainID string
	http    *http.Client
	retries int
	verbose bool
	debug   bool
	stderr  io.Writer
	limiter *Limiter
}

type Envelope struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Result  json.RawMessage `json:"result"`
}

type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Result struct {
	Raw      json.RawMessage
	Envelope *Envelope
	RPC      *RPCResponse
	Empty    bool
}

func New(opts Options) *Client {
	if opts.BaseURL == "" {
		opts.BaseURL = DefaultBaseURL
	}
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.Retries == 0 {
		opts.Retries = 3
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: opts.Timeout}
	}
	if opts.RateLimit <= 0 {
		opts.RateLimit = 3
	}
	return &Client{
		baseURL: strings.TrimRight(opts.BaseURL, "/"),
		apiKey:  opts.APIKey,
		chainID: opts.ChainID,
		http:    opts.HTTPClient,
		retries: opts.Retries,
		verbose: opts.Verbose,
		debug:   opts.Debug,
		stderr:  opts.Stderr,
		limiter: NewLimiter(opts.RateLimit),
	}
}

func (c *Client) Get(ctx context.Context, module, action string, params map[string]string, retryable bool) (Result, error) {
	values := url.Values{}
	values.Set("module", module)
	values.Set("action", action)
	if c.chainID != "" {
		values.Set("chainid", c.chainID)
	}
	if c.apiKey != "" {
		values.Set("apikey", c.apiKey)
	}
	for k, v := range params {
		if strings.TrimSpace(v) != "" {
			values.Set(k, v)
		}
	}
	endpoint, err := url.Parse(c.baseURL)
	if err != nil {
		return Result{}, err
	}
	endpoint.RawQuery = mergeQuery(endpoint.Query(), values).Encode()
	decode := decodeEnvelope
	if module == "proxy" {
		decode = decodeRPC
	}
	return c.do(ctx, http.MethodGet, endpoint.String(), "", retryable, decode)
}

// ChainList calls the dedicated supported-chains endpoint. It is the one Etherscan
// endpoint that is not module/action-shaped: a bare GET on <base>/chainlist with no
// query parameters (no API key or chainid required) and a non-envelope response.
func (c *Client) ChainList(ctx context.Context) (Result, error) {
	endpoint := strings.TrimSuffix(c.baseURL, "/api") + "/chainlist"
	return c.do(ctx, http.MethodGet, endpoint, "", true, decodeChainList)
}

func (c *Client) PostForm(ctx context.Context, module, action string, params map[string]string, retryable bool) (Result, error) {
	values := url.Values{}
	values.Set("module", module)
	values.Set("action", action)
	if c.chainID != "" {
		values.Set("chainid", c.chainID)
	}
	if c.apiKey != "" {
		values.Set("apikey", c.apiKey)
	}
	for k, v := range params {
		if strings.TrimSpace(v) != "" {
			values.Set(k, v)
		}
	}
	endpoint, err := url.Parse(c.baseURL)
	if err != nil {
		return Result{}, err
	}
	return c.do(ctx, http.MethodPost, endpoint.String(), values.Encode(), retryable, decodeEnvelope)
}

func (c *Client) do(ctx context.Context, method, endpoint, body string, retryable bool, decode func([]byte) (Result, error)) (Result, error) {
	attempts := 1
	if retryable {
		attempts = c.retries
	}
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := c.limiter.Wait(ctx); err != nil {
			return Result{}, err
		}
		start := time.Now()
		var reader io.Reader
		if body != "" {
			reader = strings.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
		if err != nil {
			return Result{}, err
		}
		if body != "" {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		if c.verbose && c.stderr != nil {
			fmt.Fprintf(c.stderr, "%s %s\n", method, RedactURL(endpoint))
		}
		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			if !retryable || !transientErr(err) || attempt == attempts {
				return Result{}, fmt.Errorf("%s %s failed: %s", method, RedactURL(endpoint), RedactSecrets(err.Error()))
			}
			sleep(ctx, backoff(attempt, ""))
			continue
		}
		raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
		resp.Body.Close()
		if c.verbose && c.stderr != nil {
			fmt.Fprintf(c.stderr, "%s completed in %s\n", resp.Status, time.Since(start).Round(time.Millisecond))
		}
		if c.debug && c.stderr != nil {
			fmt.Fprintf(c.stderr, "%s\n", raw)
		}
		if readErr != nil {
			return Result{}, readErr
		}
		if shouldRetryStatus(resp.StatusCode) && retryable && attempt < attempts {
			sleep(ctx, backoff(attempt, resp.Header.Get("Retry-After")))
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return Result{}, fmt.Errorf("request failed: %s: %s", resp.Status, strings.TrimSpace(string(raw)))
		}
		return decode(raw)
	}
	return Result{}, lastErr
}

func decodeEnvelope(raw []byte) (Result, error) {
	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return Result{}, err
	}
	if env.Status == "1" {
		return Result{Raw: env.Result, Envelope: &env}, nil
	}
	if isEmptyMessage(env.Message) || isEmptyResult(env.Result) {
		return Result{Raw: []byte("[]"), Envelope: &env, Empty: true}, nil
	}
	// Surface Etherscan's documented error message (the `result` field) verbatim: it is the
	// canonical, self-guiding string (e.g. "Max rate limit reached, please use API Key for higher
	// rate limit", "Unable to locate ContractCode at 0x..."). `env.Result` is raw JSON, so unquote
	// the normal string case and don't prefix it with the "NOTOK" status word.
	var msg string
	if json.Unmarshal(env.Result, &msg) != nil {
		msg = string(env.Result)
	}
	msg = strings.TrimSpace(msg)
	if msg != "" && msg != "null" {
		return Result{}, errors.New(msg)
	}
	return Result{}, errors.New(env.Message)
}

func decodeRPC(raw []byte) (Result, error) {
	// Rate-limit / invalid-key / unsupported-chainid / throttle checks run BEFORE the proxy
	// module is dispatched server-side, so they arrive as the standard Etherscan envelope
	// ({"status","message","result"}) rather than JSON-RPC — over HTTP 200. Detect that shape
	// (has "status", no "jsonrpc") and reuse decodeEnvelope so the error surfaces correctly
	// instead of being returned as a successful result.
	var probe struct {
		JSONRPC string `json:"jsonrpc"`
		Status  string `json:"status"`
	}
	if json.Unmarshal(raw, &probe) == nil && probe.JSONRPC == "" && probe.Status != "" {
		return decodeEnvelope(raw)
	}

	var rpc RPCResponse
	if err := json.Unmarshal(raw, &rpc); err != nil {
		return Result{}, err
	}
	if rpc.Error != nil {
		return Result{}, fmt.Errorf("json-rpc error %d: %s", rpc.Error.Code, rpc.Error.Message)
	}
	return Result{Raw: rpc.Result, RPC: &rpc}, nil
}

// decodeChainList handles the chainlist endpoint's non-envelope response shape:
// {"comments": ..., "totalcount": N, "result": [...]}. There is no status/message
// pair, so decodeEnvelope would misread a valid response as empty.
func decodeChainList(raw []byte) (Result, error) {
	var body struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return Result{}, err
	}
	if len(body.Result) == 0 {
		return Result{}, errors.New("unexpected chainlist response: missing result")
	}
	return Result{Raw: body.Result}, nil
}

func DecodeResult[T any](raw json.RawMessage) (T, error) {
	var out T
	if len(raw) == 0 {
		return out, nil
	}
	err := json.Unmarshal(raw, &out)
	return out, err
}

func mergeQuery(a, b url.Values) url.Values {
	for key, values := range b {
		for _, value := range values {
			a.Set(key, value)
		}
	}
	return a
}

func isEmptyMessage(message string) bool {
	msg := strings.ToLower(message)
	for _, known := range emptyMessages {
		if strings.Contains(msg, known) {
			return true
		}
	}
	return false
}

func isEmptyResult(raw json.RawMessage) bool {
	value := strings.Trim(strings.ToLower(string(raw)), `" `)
	// A blank result is intentionally NOT treated as "empty success": error
	// responses (e.g. an invalid API key) can come back as status "0" with a blank
	// result, and swallowing those would mask real errors — letting `login` accept a
	// bad key, or hiding rate-limit/timeout failures. Genuine empty result sets arrive
	// as an empty array, or with a "no … found" message handled by isEmptyMessage.
	return value == "[]" || strings.Contains(value, "no records found") || strings.Contains(value, "no transactions found")
}

func shouldRetryStatus(code int) bool {
	return code == http.StatusTooManyRequests || code >= 500
}

func transientErr(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
}

func backoff(attempt int, retryAfter string) time.Duration {
	if retryAfter != "" {
		if seconds, err := strconv.Atoi(retryAfter); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	base := time.Duration(150*(1<<uint(attempt-1))) * time.Millisecond
	return base + time.Duration(rand.Intn(100))*time.Millisecond
}

func sleep(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

func RedactURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	values := parsed.Query()
	if values.Get("apikey") != "" {
		values.Set("apikey", "REDACTED")
	}
	parsed.RawQuery = values.Encode()
	return parsed.String()
}

func RedactSecrets(value string) string {
	const key = "apikey="
	searchFrom := 0
	for {
		lower := strings.ToLower(value)
		if searchFrom >= len(lower) {
			return value
		}
		relative := strings.Index(lower[searchFrom:], key)
		if relative == -1 {
			return value
		}
		start := searchFrom + relative
		end := start + len(key)
		for end < len(value) {
			switch value[end] {
			case '&', ' ', '\t', '\r', '\n', '"', '\'':
				value = value[:start+len(key)] + "REDACTED" + value[end:]
				searchFrom = start + len(key) + len("REDACTED")
				goto next
			default:
				end++
			}
		}
		value = value[:start+len(key)] + "REDACTED"
		searchFrom = len(value)
	next:
	}
}

type Limiter struct {
	mu       sync.Mutex
	interval time.Duration
	next     time.Time
}

func NewLimiter(rate float64) *Limiter {
	return &Limiter{interval: time.Duration(float64(time.Second) / rate)}
}

func (l *Limiter) Wait(ctx context.Context) error {
	l.mu.Lock()
	now := time.Now()
	wait := time.Duration(0)
	if !l.next.IsZero() && now.Before(l.next) {
		wait = l.next.Sub(now)
	}
	if wait == 0 {
		l.next = now.Add(l.interval)
	} else {
		l.next = l.next.Add(l.interval)
	}
	l.mu.Unlock()
	if wait == 0 {
		return nil
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
