package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"testing"
	"time"
)

func TestNormalizeStreamTarget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		target     string
		wantScheme string
		wantAddr   string
	}{
		{name: "unix explicit", target: "unix:///tmp/agentscope.sock", wantScheme: "unix", wantAddr: "/tmp/agentscope.sock"},
		{name: "unix implicit", target: "agentscope.sock", wantScheme: "unix", wantAddr: "agentscope.sock"},
		{name: "tcp explicit", target: "tcp://127.0.0.1:7777", wantScheme: "tcp", wantAddr: "127.0.0.1:7777"},
		{name: "tcp implicit", target: "127.0.0.1:7777", wantScheme: "tcp", wantAddr: "127.0.0.1:7777"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			scheme, address, err := normalizeStreamTarget(test.target)
			if err != nil {
				t.Fatalf("normalizeStreamTarget returned error: %v", err)
			}
			if scheme != test.wantScheme {
				t.Fatalf("expected scheme %q, got %q", test.wantScheme, scheme)
			}
			if address != test.wantAddr {
				t.Fatalf("expected address %q, got %q", test.wantAddr, address)
			}
		})
	}
}

func TestStartSocketStreamReceivesUnixEvents(t *testing.T) {
	t.Parallel()

	socketPath := fmt.Sprintf("/tmp/agentscope-%d-%d.sock", os.Getpid(), time.Now().UnixNano())
	stream, closer, display, err := startSocketStream(socketPath)
	if err != nil {
		t.Fatalf("startSocketStream returned error: %v", err)
	}
	defer closer.Close()

	if want := "unix://" + socketPath; display != want {
		t.Fatalf("expected display target %q, got %q", want, display)
	}

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("Dial returned error: %v", err)
	}
	defer conn.Close()

	if _, err := fmt.Fprintln(conn, `{"time":"2026-04-19T12:00:00Z","agent":"router","channel":"intake","kind":"channel_open","status":"open","message":"Opened intake"}`); err != nil {
		t.Fatalf("write event: %v", err)
	}

	select {
	case envelope := <-stream:
		if envelope.Err != nil {
			t.Fatalf("unexpected stream error: %v", envelope.Err)
		}
		if envelope.Event.Channel != "intake" {
			t.Fatalf("expected intake channel, got %q", envelope.Event.Channel)
		}
		if envelope.Event.Kind != "channel_open" {
			t.Fatalf("expected channel_open kind, got %q", envelope.Event.Kind)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for live event")
	}
}

func TestResolveLiveProgramInputAllowsOnceWithoutTTY(t *testing.T) {
	t.Parallel()

	reader, closer, err := resolveLiveProgramInput(true, func() (io.Reader, io.Closer, error) {
		return nil, nil, errors.New("no tty")
	})
	if err != nil {
		t.Fatalf("resolveLiveProgramInput returned error: %v", err)
	}
	if reader != nil || closer != nil {
		t.Fatal("expected nil reader and closer in once mode")
	}
}

func TestResolveLiveProgramInputRequiresTTYWhenInteractive(t *testing.T) {
	t.Parallel()

	_, _, err := resolveLiveProgramInput(false, func() (io.Reader, io.Closer, error) {
		return nil, nil, errors.New("no tty")
	})
	if err == nil {
		t.Fatal("expected error when once mode is disabled")
	}
}
