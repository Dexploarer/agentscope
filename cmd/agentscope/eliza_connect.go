package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/term"
)

const defaultElizaCloudURL = "https://www.elizacloud.ai"

var (
	execElizaLogin = func(stdout, stderr io.Writer, stdin io.Reader, cloudURL string, noBrowser bool, timeoutSeconds int) error {
		args := buildElizaLoginArgs(cloudURL, noBrowser, timeoutSeconds)
		cmd := exec.Command("elizaos", args...)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		cmd.Stdin = stdin
		return cmd.Run()
	}

	readSecret = func(prompt string) (string, error) {
		tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err != nil {
			return "", fmt.Errorf("open tty for prompt: %w", err)
		}
		defer tty.Close()

		if _, err := fmt.Fprint(tty, prompt); err != nil {
			return "", fmt.Errorf("write prompt: %w", err)
		}

		secret, err := term.ReadPassword(int(tty.Fd()))
		if err != nil {
			return "", fmt.Errorf("read secret: %w", err)
		}
		if _, err := fmt.Fprintln(tty); err != nil {
			return "", fmt.Errorf("finish prompt: %w", err)
		}

		return strings.TrimSpace(string(secret)), nil
	}

	runSecurityCommand = func(args ...string) ([]byte, error) {
		cmd := exec.Command("security", args...)
		return cmd.CombinedOutput()
	}
)

type elizaAuthConfig struct {
	ServerURL       string
	APIKey          string
	CloudURL        string
	Prompt          bool
	CloudLogin      bool
	NoBrowser       bool
	SkipPing        bool
	UseKeychain     bool
	SaveKeychain    bool
	KeychainService string
	KeychainAccount string
	TimeoutSeconds  int
}

type elizaConnectionResult struct {
	Config         elizaAuthConfig
	ResolvedAPIKey string
	Health         string
}

func defaultElizaAuthConfig() elizaAuthConfig {
	return elizaAuthConfig{
		ServerURL: envOrDefault("ELIZA_SERVER_URL", "http://localhost:3000"),
		APIKey: firstNonEmpty(
			os.Getenv("ELIZA_API_KEY"),
			os.Getenv("ELIZA_SERVER_AUTH_TOKEN"),
			os.Getenv("ELIZAOS_API_KEY"),
			os.Getenv("ELIZAOS_CLOUD_API_KEY"),
		),
		CloudURL:        envOrDefault("ELIZA_CLOUD_URL", defaultElizaCloudURL),
		KeychainService: "agentscope.eliza",
		KeychainAccount: envOrDefault("USER", "default"),
		TimeoutSeconds:  300,
	}
}

func bindElizaAuthFlags(fs *flag.FlagSet, cfg *elizaAuthConfig) {
	fs.StringVar(&cfg.ServerURL, "server-url", cfg.ServerURL, "Eliza server base URL")
	fs.StringVar(&cfg.APIKey, "api-key", cfg.APIKey, "API key sent as X-API-KEY")
	fs.StringVar(&cfg.CloudURL, "cloud-url", cfg.CloudURL, "Eliza Cloud URL for official login")
	fs.BoolVar(&cfg.Prompt, "prompt", cfg.Prompt, "prompt for API key if none is configured")
	fs.BoolVar(&cfg.CloudLogin, "cloud-login", cfg.CloudLogin, "run the official `elizaos login` browser flow first")
	fs.BoolVar(&cfg.NoBrowser, "no-browser", cfg.NoBrowser, "do not auto-open the browser during `elizaos login`")
	fs.BoolVar(&cfg.SkipPing, "skip-ping", cfg.SkipPing, "skip the server health check")
	fs.BoolVar(&cfg.UseKeychain, "use-keychain", cfg.UseKeychain, "load the API key from macOS Keychain")
	fs.BoolVar(&cfg.SaveKeychain, "save-keychain", cfg.SaveKeychain, "save the resolved API key to macOS Keychain")
	fs.StringVar(&cfg.KeychainService, "keychain-service", cfg.KeychainService, "macOS Keychain service name")
	fs.StringVar(&cfg.KeychainAccount, "keychain-account", cfg.KeychainAccount, "macOS Keychain account name")
	fs.IntVar(&cfg.TimeoutSeconds, "timeout", cfg.TimeoutSeconds, "timeout in seconds for `elizaos login`")
}

func runConnectEliza(args []string, stdout, stderr io.Writer, stdin io.Reader) error {
	fs := flag.NewFlagSet("connect-eliza", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	cfg := defaultElizaAuthConfig()
	bindElizaAuthFlags(fs, &cfg)

	if err := fs.Parse(args); err != nil {
		return err
	}

	result, err := connectEliza(cfg, stdout, stderr, stdin)
	if err != nil {
		return err
	}

	printElizaConnectionSummary(stdout, result)
	fmt.Fprintln(stdout, "Next: start `agentscope monitor -listen unix:///tmp/agentscope.sock` and load the AgentScope Eliza plugin in your runtime.")
	return nil
}

func connectEliza(cfg elizaAuthConfig, stdout, stderr io.Writer, stdin io.Reader) (elizaConnectionResult, error) {
	result := elizaConnectionResult{Config: cfg}

	if cfg.CloudLogin {
		if err := execElizaLogin(stdout, stderr, stdin, cfg.CloudURL, cfg.NoBrowser, cfg.TimeoutSeconds); err != nil {
			return elizaConnectionResult{}, fmt.Errorf("elizaos login failed: %w", err)
		}
	}

	apiKey := strings.TrimSpace(cfg.APIKey)
	if cfg.UseKeychain && apiKey == "" {
		keychainKey, err := loadKeychainSecret(cfg.KeychainService, cfg.KeychainAccount)
		if err != nil {
			return elizaConnectionResult{}, err
		}
		apiKey = keychainKey
	}

	resolvedKey, err := resolveElizaAPIKey(apiKey, cfg.Prompt)
	if err != nil {
		return elizaConnectionResult{}, err
	}
	if cfg.SaveKeychain && resolvedKey != "" {
		if err := storeKeychainSecret(cfg.KeychainService, cfg.KeychainAccount, resolvedKey); err != nil {
			return elizaConnectionResult{}, err
		}
	}

	result.ResolvedAPIKey = resolvedKey
	if cfg.SkipPing {
		result.Health = "skipped"
		return result, nil
	}

	message, err := pingElizaServer(http.DefaultClient, cfg.ServerURL, resolvedKey)
	if err != nil {
		return elizaConnectionResult{}, err
	}
	result.Health = message
	return result, nil
}

func printElizaConnectionSummary(w io.Writer, result elizaConnectionResult) {
	cfg := result.Config

	fmt.Fprintln(w, "Eliza Connection")
	fmt.Fprintf(w, "Server: %s\n", strings.TrimSpace(cfg.ServerURL))
	if result.ResolvedAPIKey == "" {
		fmt.Fprintln(w, "Auth: none")
	} else {
		fmt.Fprintln(w, "Auth: X-API-KEY configured")
	}
	if cfg.UseKeychain {
		fmt.Fprintf(w, "Keychain Read: %s/%s\n", strings.TrimSpace(cfg.KeychainService), strings.TrimSpace(cfg.KeychainAccount))
	}
	if cfg.SaveKeychain && result.ResolvedAPIKey != "" {
		fmt.Fprintf(w, "Keychain Save: %s/%s\n", strings.TrimSpace(cfg.KeychainService), strings.TrimSpace(cfg.KeychainAccount))
	}
	if cfg.CloudLogin {
		fmt.Fprintf(w, "Cloud Login: completed against %s\n", strings.TrimSpace(cfg.CloudURL))
	}
	fmt.Fprintf(w, "Health: %s\n", result.Health)
}

func resolveElizaAPIKey(explicit string, prompt bool) (string, error) {
	key := strings.TrimSpace(explicit)
	if key != "" {
		return key, nil
	}
	if !prompt {
		return "", nil
	}

	key, err := readSecret("Eliza API key (X-API-KEY): ")
	if err != nil {
		return "", err
	}
	if key == "" {
		return "", errors.New("empty API key")
	}
	return key, nil
}

func pingElizaServer(client *http.Client, serverURL, apiKey string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(serverURL), "/")
	if base == "" {
		return "", errors.New("server URL is required")
	}

	request, err := http.NewRequest(http.MethodGet, base+"/api/server/ping", nil)
	if err != nil {
		return "", fmt.Errorf("build health request: %w", err)
	}
	if strings.TrimSpace(apiKey) != "" {
		request.Header.Set("X-API-KEY", strings.TrimSpace(apiKey))
	}

	httpClient := client
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	response, err := httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("ping eliza server: %w", err)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
		return "ok", nil
	case http.StatusUnauthorized:
		return "", errors.New("eliza server rejected X-API-KEY (401 Unauthorized)")
	default:
		return "", fmt.Errorf("eliza server health check failed: %s", response.Status)
	}
}

func buildElizaLoginArgs(cloudURL string, noBrowser bool, timeoutSeconds int) []string {
	args := []string{"login"}
	if strings.TrimSpace(cloudURL) != "" {
		args = append(args, "--cloud-url", strings.TrimSpace(cloudURL))
	}
	if noBrowser {
		args = append(args, "--no-browser")
	}
	if timeoutSeconds > 0 {
		args = append(args, "--timeout", fmt.Sprintf("%d", timeoutSeconds))
	}
	return args
}

func loadKeychainSecret(service, account string) (string, error) {
	service = strings.TrimSpace(service)
	account = strings.TrimSpace(account)
	if service == "" || account == "" {
		return "", errors.New("keychain service and account are required")
	}

	output, err := runSecurityCommand(
		"find-generic-password",
		"-s", service,
		"-a", account,
		"-w",
	)
	if err != nil {
		return "", fmt.Errorf("load keychain secret %s/%s: %w", service, account, err)
	}

	secret := strings.TrimSpace(string(output))
	if secret == "" {
		return "", fmt.Errorf("load keychain secret %s/%s: empty secret", service, account)
	}
	return secret, nil
}

func storeKeychainSecret(service, account, secret string) error {
	service = strings.TrimSpace(service)
	account = strings.TrimSpace(account)
	secret = strings.TrimSpace(secret)
	if service == "" || account == "" {
		return errors.New("keychain service and account are required")
	}
	if secret == "" {
		return errors.New("keychain secret is required")
	}

	_, err := runSecurityCommand(
		"add-generic-password",
		"-U",
		"-s", service,
		"-a", account,
		"-w", secret,
	)
	if err != nil {
		return fmt.Errorf("store keychain secret %s/%s: %w", service, account, err)
	}
	return nil
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value != "" {
		return value
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
