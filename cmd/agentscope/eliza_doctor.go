package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	lookPathCommand = exec.LookPath
	statPath        = os.Stat
)

type doctorCheck struct {
	Name    string
	Detail  string
	Err     error
	Skipped bool
}

func (c doctorCheck) status() string {
	switch {
	case c.Skipped:
		return "SKIP"
	case c.Err != nil:
		return "FAIL"
	default:
		return "OK"
	}
}

func runDoctorEliza(args []string, stdout, stderr io.Writer, stdin io.Reader) error {
	fs := flag.NewFlagSet("doctor-eliza", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	auth := defaultElizaAuthConfig()
	bindElizaAuthFlags(fs, &auth)

	listenTarget := "unix:///tmp/agentscope.sock"
	bridgePath := preferredBridgePackagePath(projectRoot())

	fs.StringVar(&listenTarget, "listen", listenTarget, "socket target used by the AgentScope monitor")
	fs.StringVar(&bridgePath, "bridge-path", bridgePath, "path to the local Eliza bridge package")

	if err := fs.Parse(args); err != nil {
		return err
	}

	checks := runDoctorChecks(auth, strings.TrimSpace(listenTarget), strings.TrimSpace(bridgePath))

	fmt.Fprintln(stdout, "AgentScope Eliza Doctor")
	for _, check := range checks {
		line := fmt.Sprintf("[%s] %s", check.status(), check.Name)
		if check.Detail != "" {
			line += " - " + check.Detail
		}
		if check.Err != nil {
			line += " - " + check.Err.Error()
		}
		fmt.Fprintln(stdout, line)
	}

	for _, check := range checks {
		if check.Err != nil {
			return errors.New("doctor-eliza found one or more failing checks")
		}
	}
	return nil
}

func runDoctorChecks(auth elizaAuthConfig, listenTarget, bridgePath string) []doctorCheck {
	checks := []doctorCheck{
		checkElizaCLIInstalled(),
		checkBridgePackage(bridgePath),
		checkListenTarget(listenTarget),
	}

	authCheck, apiKey := checkDoctorAuth(auth)
	checks = append(checks, authCheck)

	serverCheck := checkDoctorServer(auth, apiKey)
	checks = append(checks, serverCheck)

	return checks
}

func checkElizaCLIInstalled() doctorCheck {
	path, err := lookPathCommand("elizaos")
	if err != nil {
		return doctorCheck{
			Name:   "elizaos CLI",
			Detail: "install the official CLI or make sure it is on PATH",
			Err:    err,
		}
	}

	return doctorCheck{
		Name:   "elizaos CLI",
		Detail: path,
	}
}

func checkBridgePackage(path string) doctorCheck {
	if path == "" {
		return doctorCheck{Name: "bridge package", Detail: "no bridge path provided", Skipped: true}
	}

	info, err := statPath(path)
	if err != nil {
		return doctorCheck{
			Name:   "bridge package",
			Detail: path,
			Err:    err,
		}
	}
	if !info.IsDir() {
		return doctorCheck{
			Name:   "bridge package",
			Detail: path,
			Err:    errors.New("path is not a directory"),
		}
	}

	packageJSON := filepath.Join(path, "package.json")
	if _, err := statPath(packageJSON); err != nil {
		return doctorCheck{
			Name:   "bridge package",
			Detail: packageJSON,
			Err:    err,
		}
	}

	return doctorCheck{
		Name:   "bridge package",
		Detail: packageJSON,
	}
}

func checkListenTarget(target string) doctorCheck {
	if target == "" {
		return doctorCheck{Name: "listen target", Detail: "empty listen target", Err: errors.New("listen target is required")}
	}

	scheme, address, err := normalizeStreamTarget(target)
	if err != nil {
		return doctorCheck{Name: "listen target", Detail: target, Err: err}
	}

	switch scheme {
	case "unix":
		dir := filepath.Dir(address)
		info, err := statPath(dir)
		if err != nil {
			return doctorCheck{Name: "listen target", Detail: dir, Err: err}
		}
		if !info.IsDir() {
			return doctorCheck{Name: "listen target", Detail: dir, Err: errors.New("socket parent is not a directory")}
		}
		return doctorCheck{Name: "listen target", Detail: "unix socket under " + dir}
	case "tcp":
		return doctorCheck{Name: "listen target", Detail: "tcp listener " + address}
	default:
		return doctorCheck{Name: "listen target", Detail: target, Err: fmt.Errorf("unsupported target scheme %q", scheme)}
	}
}

func checkDoctorAuth(auth elizaAuthConfig) (doctorCheck, string) {
	if strings.TrimSpace(auth.APIKey) != "" {
		return doctorCheck{Name: "auth source", Detail: "explicit API key configured"}, strings.TrimSpace(auth.APIKey)
	}
	if auth.UseKeychain {
		secret, err := loadKeychainSecret(auth.KeychainService, auth.KeychainAccount)
		if err != nil {
			return doctorCheck{
				Name:   "auth source",
				Detail: fmt.Sprintf("keychain %s/%s", auth.KeychainService, auth.KeychainAccount),
				Err:    err,
			}, ""
		}
		return doctorCheck{
			Name:   "auth source",
			Detail: fmt.Sprintf("keychain %s/%s", auth.KeychainService, auth.KeychainAccount),
		}, secret
	}
	if auth.Prompt {
		return doctorCheck{
			Name:    "auth source",
			Detail:  "interactive prompt required at connect time",
			Skipped: true,
		}, ""
	}
	if auth.CloudLogin {
		return doctorCheck{
			Name:    "auth source",
			Detail:  "cloud login will provision credentials when run",
			Skipped: true,
		}, ""
	}
	return doctorCheck{
		Name:    "auth source",
		Detail:  "set -api-key, -use-keychain, -prompt, or -cloud-login",
		Skipped: true,
	}, ""
}

func checkDoctorServer(auth elizaAuthConfig, apiKey string) doctorCheck {
	if auth.SkipPing {
		return doctorCheck{Name: "server health", Detail: "ping skipped", Skipped: true}
	}
	if apiKey == "" {
		return doctorCheck{
			Name:    "server health",
			Detail:  "no noninteractive key available for ping",
			Skipped: true,
		}
	}

	message, err := pingElizaServer(nil, auth.ServerURL, apiKey)
	if err != nil {
		return doctorCheck{
			Name:   "server health",
			Detail: strings.TrimSpace(auth.ServerURL),
			Err:    err,
		}
	}
	return doctorCheck{
		Name:   "server health",
		Detail: strings.TrimSpace(auth.ServerURL) + " (" + message + ")",
	}
}
