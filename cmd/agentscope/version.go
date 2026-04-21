package main

import (
	"fmt"
	"io"
	"strings"
)

var (
	buildVersion = "dev"
	buildCommit  = "unknown"
	buildTime    = ""
)

func runVersion(stdout io.Writer) error {
	_, err := fmt.Fprintln(stdout, renderVersion())
	return err
}

func renderVersion() string {
	parts := []string{
		fmt.Sprintf("version=%s", strings.TrimSpace(buildVersion)),
		fmt.Sprintf("commit=%s", strings.TrimSpace(buildCommit)),
	}
	if strings.TrimSpace(buildTime) != "" {
		parts = append(parts, fmt.Sprintf("built=%s", strings.TrimSpace(buildTime)))
	}
	return strings.Join(parts, " ")
}
