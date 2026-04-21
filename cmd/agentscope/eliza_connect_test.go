package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResolveElizaAPIKeyUsesPromptWhenRequested(t *testing.T) {
	original := readSecret
	defer func() {
		readSecret = original
	}()

	readSecret = func(prompt string) (string, error) {
		if prompt != "Eliza API key (X-API-KEY): " {
			t.Fatalf("unexpected prompt %q", prompt)
		}
		return "secret-token", nil
	}

	key, err := resolveElizaAPIKey("", true)
	if err != nil {
		t.Fatalf("resolveElizaAPIKey returned error: %v", err)
	}
	if key != "secret-token" {
		t.Fatalf("expected secret-token, got %q", key)
	}
}

func TestLoadKeychainSecret(t *testing.T) {
	original := runSecurityCommand
	defer func() {
		runSecurityCommand = original
	}()

	runSecurityCommand = func(args ...string) ([]byte, error) {
		expected := []string{"find-generic-password", "-s", "agentscope.eliza", "-a", "alice", "-w"}
		if strings.Join(args, " ") != strings.Join(expected, " ") {
			t.Fatalf("unexpected args %v", args)
		}
		return []byte("secret-token\n"), nil
	}

	secret, err := loadKeychainSecret("agentscope.eliza", "alice")
	if err != nil {
		t.Fatalf("loadKeychainSecret returned error: %v", err)
	}
	if secret != "secret-token" {
		t.Fatalf("expected secret-token, got %q", secret)
	}
}

func TestStoreKeychainSecret(t *testing.T) {
	original := runSecurityCommand
	defer func() {
		runSecurityCommand = original
	}()

	runSecurityCommand = func(args ...string) ([]byte, error) {
		expected := []string{"add-generic-password", "-U", "-s", "agentscope.eliza", "-a", "alice", "-w", "secret-token"}
		if strings.Join(args, " ") != strings.Join(expected, " ") {
			t.Fatalf("unexpected args %v", args)
		}
		return nil, nil
	}

	if err := storeKeychainSecret("agentscope.eliza", "alice", "secret-token"); err != nil {
		t.Fatalf("storeKeychainSecret returned error: %v", err)
	}
}

func TestPingElizaServerAddsHeader(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/server/ping" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("X-API-KEY"); got != "secret-token" {
			t.Fatalf("expected X-API-KEY secret-token, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	message, err := pingElizaServer(server.Client(), server.URL, "secret-token")
	if err != nil {
		t.Fatalf("pingElizaServer returned error: %v", err)
	}
	if message != "ok" {
		t.Fatalf("expected ok, got %q", message)
	}
}

func TestBuildElizaLoginArgsMatchesOfficialFlags(t *testing.T) {
	t.Parallel()

	args := buildElizaLoginArgs("https://custom.elizacloud.ai", true, 600)
	want := []string{"login", "--cloud-url", "https://custom.elizacloud.ai", "--no-browser", "--timeout", "600"}

	if len(args) != len(want) {
		t.Fatalf("expected %d args, got %d", len(want), len(args))
	}
	for index := range want {
		if args[index] != want[index] {
			t.Fatalf("expected arg %d to be %q, got %q", index, want[index], args[index])
		}
	}
}

func TestRunConnectElizaPrintsHealthyConnection(t *testing.T) {
	originalPrompt := readSecret
	originalLogin := execElizaLogin
	originalSecurity := runSecurityCommand
	defer func() {
		readSecret = originalPrompt
		execElizaLogin = originalLogin
		runSecurityCommand = originalSecurity
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-KEY") != "prompted-key" {
			t.Fatalf("expected prompted key, got %q", r.Header.Get("X-API-KEY"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	readSecret = func(prompt string) (string, error) {
		return "prompted-key", nil
	}
	execElizaLogin = func(stdout, stderr io.Writer, stdin io.Reader, cloudURL string, noBrowser bool, timeoutSeconds int) error {
		return nil
	}
	runSecurityCommand = func(args ...string) ([]byte, error) {
		return nil, errors.New("unexpected keychain call")
	}

	var stdout bytes.Buffer
	err := runConnectEliza([]string{
		"-server-url", server.URL,
		"-prompt",
		"-cloud-login",
		"-skip-ping=false",
	}, &stdout, io.Discard, strings.NewReader(""))
	if err != nil {
		t.Fatalf("runConnectEliza returned error: %v", err)
	}

	output := stdout.String()
	for _, expected := range []string{
		"Eliza Connection",
		"Auth: X-API-KEY configured",
		"Health: ok",
		"Cloud Login: completed",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected output to contain %q, got %q", expected, output)
		}
	}
}

func TestRunConnectElizaLoadsKeyFromKeychain(t *testing.T) {
	originalSecurity := runSecurityCommand
	defer func() {
		runSecurityCommand = originalSecurity
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-KEY") != "keychain-token" {
			t.Fatalf("expected keychain token, got %q", r.Header.Get("X-API-KEY"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	runSecurityCommand = func(args ...string) ([]byte, error) {
		return []byte("keychain-token"), nil
	}

	var stdout bytes.Buffer
	err := runConnectEliza([]string{
		"-server-url", server.URL,
		"-use-keychain",
		"-keychain-account", "alice",
	}, &stdout, io.Discard, strings.NewReader(""))
	if err != nil {
		t.Fatalf("runConnectEliza returned error: %v", err)
	}

	if !strings.Contains(stdout.String(), "Keychain Read: agentscope.eliza/alice") {
		t.Fatalf("expected keychain read marker, got %q", stdout.String())
	}
}
