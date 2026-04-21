package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckListenTargetUnix(t *testing.T) {
	t.Parallel()

	check := checkListenTarget("unix:///tmp/agentscope.sock")
	if check.Err != nil {
		t.Fatalf("expected unix listen target to pass, got %v", check.Err)
	}
	if check.status() != "OK" {
		t.Fatalf("expected OK status, got %s", check.status())
	}
}

func TestCheckBridgePackage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	packageJSON := filepath.Join(dir, "package.json")
	if err := os.WriteFile(packageJSON, []byte(`{"name":"@agentscope/elizaos-bridge"}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	check := checkBridgePackage(dir)
	if check.Err != nil {
		t.Fatalf("expected bridge package to pass, got %v", check.Err)
	}
}

func TestCheckDoctorAuthUsesKeychain(t *testing.T) {
	original := runSecurityCommand
	defer func() {
		runSecurityCommand = original
	}()

	runSecurityCommand = func(args ...string) ([]byte, error) {
		return []byte("keychain-token"), nil
	}

	check, token := checkDoctorAuth(elizaAuthConfig{
		UseKeychain:     true,
		KeychainService: "agentscope.eliza",
		KeychainAccount: "alice",
	})
	if check.Err != nil {
		t.Fatalf("expected keychain auth to pass, got %v", check.Err)
	}
	if token != "keychain-token" {
		t.Fatalf("expected keychain-token, got %q", token)
	}
}

func TestCheckDoctorServer(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-KEY") != "secret-token" {
			t.Fatalf("expected secret-token, got %q", r.Header.Get("X-API-KEY"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	check := checkDoctorServer(elizaAuthConfig{ServerURL: server.URL}, "secret-token")
	if check.Err != nil {
		t.Fatalf("expected server check to pass, got %v", check.Err)
	}
}

func TestRunDoctorElizaFailsOnMissingCLI(t *testing.T) {
	originalLookPath := lookPathCommand
	originalStat := statPath
	defer func() {
		lookPathCommand = originalLookPath
		statPath = originalStat
	}()

	lookPathCommand = func(file string) (string, error) {
		return "", errors.New("not found")
	}
	statPath = os.Stat

	var output bytes.Buffer
	err := runDoctorEliza([]string{
		"-skip-ping",
		"-bridge-path", t.TempDir(),
	}, &output, io.Discard, strings.NewReader(""))
	if err == nil {
		t.Fatal("expected doctor to fail")
	}
	if !strings.Contains(output.String(), "[FAIL] elizaos CLI") {
		t.Fatalf("expected output to contain CLI failure, got %q", output.String())
	}
}
