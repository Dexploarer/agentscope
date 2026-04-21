package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"

	"agentscope/internal/agent"
	"agentscope/internal/tui"
)

type streamServer struct {
	listener   net.Listener
	removePath string
	events     chan tui.StreamEnvelope
	done       chan struct{}
	once       sync.Once
	wg         sync.WaitGroup

	mu    sync.Mutex
	conns map[net.Conn]struct{}
}

func startSocketStream(target string) (<-chan tui.StreamEnvelope, io.Closer, string, error) {
	listener, removePath, display, err := listenTarget(target)
	if err != nil {
		return nil, nil, "", err
	}

	server := &streamServer{
		listener:   listener,
		removePath: removePath,
		events:     make(chan tui.StreamEnvelope),
		done:       make(chan struct{}),
		conns:      make(map[net.Conn]struct{}),
	}

	server.wg.Add(1)
	go server.run()

	return server.events, server, display, nil
}

func (s *streamServer) run() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			switch {
			case errors.Is(err, net.ErrClosed):
				return
			case isClosed(s.done):
				return
			default:
				select {
				case s.events <- tui.StreamEnvelope{Err: fmt.Errorf("accept live stream: %w", err)}:
				case <-s.done:
				}
				continue
			}
		}

		s.mu.Lock()
		s.conns[conn] = struct{}{}
		s.mu.Unlock()

		s.wg.Add(1)
		go s.handleConn(conn)
	}
}

func (s *streamServer) handleConn(conn net.Conn) {
	defer s.wg.Done()
	defer s.dropConn(conn)

	err := agent.StreamEvents(conn, func(event agent.Event) error {
		select {
		case s.events <- tui.StreamEnvelope{Event: event}:
			return nil
		case <-s.done:
			return io.EOF
		}
	})
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
		select {
		case s.events <- tui.StreamEnvelope{Err: fmt.Errorf("stream %s: %w", conn.RemoteAddr(), err)}:
		case <-s.done:
		}
	}
}

func (s *streamServer) dropConn(conn net.Conn) {
	s.mu.Lock()
	delete(s.conns, conn)
	s.mu.Unlock()
	_ = conn.Close()
}

func (s *streamServer) Close() error {
	var closeErr error

	s.once.Do(func() {
		close(s.done)
		closeErr = s.listener.Close()

		s.mu.Lock()
		for conn := range s.conns {
			_ = conn.Close()
		}
		s.mu.Unlock()

		s.wg.Wait()
		close(s.events)

		if s.removePath != "" {
			_ = os.Remove(s.removePath)
		}
	})

	return closeErr
}

func listenTarget(target string) (net.Listener, string, string, error) {
	scheme, address, err := normalizeStreamTarget(target)
	if err != nil {
		return nil, "", "", err
	}

	switch scheme {
	case "unix":
		if err := removeUnixSocket(address); err != nil {
			return nil, "", "", err
		}
		listener, err := net.Listen("unix", address)
		if err != nil {
			return nil, "", "", fmt.Errorf("listen on unix socket %q: %w", address, err)
		}
		return listener, address, "unix://" + address, nil
	case "tcp":
		listener, err := net.Listen("tcp", address)
		if err != nil {
			return nil, "", "", fmt.Errorf("listen on tcp address %q: %w", address, err)
		}
		return listener, "", "tcp://" + listener.Addr().String(), nil
	default:
		return nil, "", "", fmt.Errorf("unsupported live stream scheme %q", scheme)
	}
}

func normalizeStreamTarget(target string) (scheme string, address string, err error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", "", errors.New("live stream target is required")
	}

	if strings.HasPrefix(target, "unix://") {
		address = strings.TrimPrefix(target, "unix://")
		if address == "" {
			return "", "", errors.New("unix socket path is required")
		}
		return "unix", address, nil
	}
	if strings.HasPrefix(target, "tcp://") {
		address = strings.TrimPrefix(target, "tcp://")
		if address == "" {
			return "", "", errors.New("tcp address is required")
		}
		return "tcp", address, nil
	}
	if strings.Contains(target, "://") {
		return "", "", fmt.Errorf("unsupported live stream target %q", target)
	}

	switch {
	case strings.Contains(target, "/"), strings.HasSuffix(target, ".sock"), !strings.Contains(target, ":"):
		return "unix", target, nil
	default:
		return "tcp", target, nil
	}
}

func removeUnixSocket(path string) error {
	err := os.Remove(path)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, os.ErrNotExist):
		return nil
	default:
		return fmt.Errorf("remove existing socket %q: %w", path, err)
	}
}

func openConsoleInput() (io.Reader, io.Closer, error) {
	tty, err := os.Open("/dev/tty")
	if err != nil {
		return nil, nil, fmt.Errorf("event stream is using stdin; open /dev/tty for keyboard input: %w", err)
	}
	return tty, tty, nil
}

func resolveLiveProgramInput(
	once bool,
	opener func() (io.Reader, io.Closer, error),
) (io.Reader, io.Closer, error) {
	reader, closer, err := opener()
	if err != nil {
		if once {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	return reader, closer, nil
}

func isClosed(done <-chan struct{}) bool {
	select {
	case <-done:
		return true
	default:
		return false
	}
}
