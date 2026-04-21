package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"agentscope/internal/agent"
	"agentscope/internal/tui"
	"agentscope/internal/ui"
	tea "charm.land/bubbletea/v2"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr, os.Stdin); err != nil {
		fmt.Fprintf(os.Stderr, "agentscope: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer, stdin io.Reader) error {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return runConsole(args, stdout, stdin)
	}

	switch args[0] {
	case "console":
		return runConsole(args[1:], stdout, stdin)
	case "bootstrap-eliza":
		return runBootstrapEliza(args[1:], stdout, stderr, stdin)
	case "connect-eliza":
		return runConnectEliza(args[1:], stdout, stderr, stdin)
	case "doctor-eliza":
		return runDoctorEliza(args[1:], stdout, stderr, stdin)
	case "dashboard":
		return runDashboard(args[1:], stdout, stdin)
	case "events", "stream":
		return runEvents(args[1:], stdout, stdin)
	case "monitor":
		return runConsole(args[1:], stdout, stdin)
	case "sample-events":
		_, err := stdout.Write(agent.SampleEventsNDJSON())
		return err
	case "version":
		return runVersion(stdout)
	case "help", "-h", "--help":
		printUsage(stdout)
		return nil
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runDashboard(args []string, stdout io.Writer, stdin io.Reader) error {
	fs := flag.NewFlagSet("dashboard", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var dataPath string
	var width int

	fs.StringVar(&dataPath, "data", "", "path to a JSON snapshot file, or - for stdin")
	fs.IntVar(&width, "width", 0, "render width; defaults to $COLUMNS or 110")

	if err := fs.Parse(args); err != nil {
		return err
	}

	snapshot, err := loadSnapshot(dataPath, stdin)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(stdout, ui.RenderDashboard(snapshot, resolveWidth(width)))
	return err
}

func runConsole(args []string, stdout io.Writer, stdin io.Reader) error {
	fs := flag.NewFlagSet("console", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var dataPath string
	var filePath string
	var listenTarget string
	var width int
	var height int
	var once bool
	var workspace string

	fs.StringVar(&dataPath, "data", "", "path to a JSON snapshot file, or - for stdin")
	fs.StringVar(&filePath, "file", "", "path to an NDJSON events file, or - for stdin")
	fs.StringVar(&listenTarget, "listen", "", "listen for live NDJSON events on unix:///path.sock or tcp://host:port")
	fs.IntVar(&width, "width", 0, "render width; defaults to $COLUMNS or 140")
	fs.IntVar(&height, "height", 0, "render height; defaults to $LINES or 40")
	fs.BoolVar(&once, "once", false, "quit automatically when a finite live stream ends")
	fs.StringVar(&workspace, "workspace", "live-fleet", "workspace name when starting from an empty board")

	if err := fs.Parse(args); err != nil {
		return err
	}

	stdinHasData, err := hasPipedInput(stdin)
	if err != nil {
		return err
	}
	if listenTarget != "" && (filePath != "" || stdinHasData) {
		return errors.New("use either -listen or stdin/-file for live events, not both")
	}

	var snapshot agent.Snapshot
	switch {
	case dataPath != "":
		snapshot, err = loadSnapshot(dataPath, stdin)
	case filePath == "" && listenTarget == "" && !stdinHasData:
		snapshot, err = agent.SampleSnapshot()
	default:
		snapshot = agent.NewSnapshot(workspace)
	}
	if err != nil {
		return err
	}

	var (
		stream       <-chan tui.StreamEnvelope
		streamCloser io.Closer
		inputCloser  io.Closer
		programInput io.Reader = os.Stdin
		connection             = "embedded snapshot"
	)

	switch {
	case listenTarget != "":
		stream, streamCloser, connection, err = startSocketStream(listenTarget)
		if err != nil {
			return err
		}
	case filePath != "" || stdinHasData:
		reader, currentCloser, usingStdin, err := openEventsReader(filePath, stdin)
		if err != nil {
			return err
		}
		streamCloser = currentCloser
		stream = startEventStream(reader)
		if usingStdin {
			connection = "stdin"
			programInput, inputCloser, err = resolveLiveProgramInput(once, openConsoleInput)
			if err != nil {
				return err
			}
		} else {
			connection = filePath
		}
	case dataPath != "":
		connection = dataPath
	}
	if streamCloser != nil {
		defer streamCloser.Close()
	}
	if inputCloser != nil {
		defer inputCloser.Close()
	}

	model, err := tui.New(snapshot, tui.Options{
		Width:           resolveWidth(width),
		Height:          resolveHeight(height),
		Live:            stream != nil,
		Stream:          stream,
		Connection:      connection,
		QuitOnStreamEnd: once,
	})
	if err != nil {
		return err
	}

	options := []tea.ProgramOption{
		tea.WithOutput(stdout),
		tea.WithWindowSize(resolveWidth(width), resolveHeight(height)),
	}
	if programInput == nil {
		options = append(options, tea.WithInput(nil))
	} else {
		options = append(options, tea.WithInput(programInput))
	}

	program := tea.NewProgram(model, options...)
	_, err = program.Run()
	return err
}

func runEvents(args []string, stdout io.Writer, stdin io.Reader) error {
	fs := flag.NewFlagSet("events", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var filePath string

	fs.StringVar(&filePath, "file", "", "path to an NDJSON events file, or - for stdin")
	if err := fs.Parse(args); err != nil {
		return err
	}

	reader, closer, _, err := openEventsReader(filePath, stdin)
	if err != nil {
		return err
	}
	if closer != nil {
		defer closer.Close()
	}

	events, err := agent.LoadEvents(reader)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return errors.New("no events found")
	}

	for index, event := range events {
		if index > 0 {
			if _, err := fmt.Fprintln(stdout); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(stdout, ui.RenderEvent(event)); err != nil {
			return err
		}
	}

	return nil
}

func openEventsReader(filePath string, stdin io.Reader) (io.Reader, io.Closer, bool, error) {
	switch {
	case filePath == "":
		hasData, err := hasPipedInput(stdin)
		if err != nil {
			return nil, nil, false, err
		}
		if !hasData {
			return nil, nil, false, errors.New("events expects NDJSON via stdin or -file")
		}
		return stdin, nil, true, nil
	case filePath == "-":
		return stdin, nil, true, nil
	default:
		file, err := os.Open(filePath)
		if err != nil {
			return nil, nil, false, fmt.Errorf("open events file: %w", err)
		}
		return file, file, false, nil
	}
}

func resolveWidth(flagWidth int) int {
	if flagWidth > 0 {
		return flagWidth
	}

	if columns, err := strconv.Atoi(strings.TrimSpace(os.Getenv("COLUMNS"))); err == nil && columns > 0 {
		return columns
	}

	return 140
}

func resolveHeight(flagHeight int) int {
	if flagHeight > 0 {
		return flagHeight
	}

	if lines, err := strconv.Atoi(strings.TrimSpace(os.Getenv("LINES"))); err == nil && lines > 0 {
		return lines
	}

	return 40
}

func loadSnapshot(dataPath string, stdin io.Reader) (agent.Snapshot, error) {
	switch {
	case dataPath == "":
		return agent.SampleSnapshot()
	case dataPath == "-":
		return agent.LoadSnapshot(stdin)
	default:
		file, err := os.Open(dataPath)
		if err != nil {
			return agent.Snapshot{}, fmt.Errorf("open snapshot: %w", err)
		}
		defer file.Close()

		snapshot, err := agent.LoadSnapshot(file)
		if err != nil {
			return agent.Snapshot{}, err
		}
		return snapshot, nil
	}
}

func hasPipedInput(r io.Reader) (bool, error) {
	file, ok := r.(*os.File)
	if !ok {
		return true, nil
	}

	info, err := file.Stat()
	if err != nil {
		return false, fmt.Errorf("inspect stdin: %w", err)
	}

	return info.Mode()&os.ModeCharDevice == 0, nil
}

func startEventStream(reader io.Reader) <-chan tui.StreamEnvelope {
	ch := make(chan tui.StreamEnvelope)

	go func() {
		defer close(ch)

		err := agent.StreamEvents(reader, func(event agent.Event) error {
			ch <- tui.StreamEnvelope{Event: event}
			return nil
		})
		if err != nil {
			ch <- tui.StreamEnvelope{Err: err}
		}
	}()

	return ch
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "agentscope renders a styled CLI surface for agent fleets.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  agentscope [console] [-data snapshot.json] [-file events.ndjson | -listen unix:///tmp/agentscope.sock] [-once] [-width 140] [-height 40]")
	fmt.Fprintln(w, "  agentscope bootstrap-eliza [-server-url http://localhost:3000] [-prompt | -use-keychain | -cloud-login] [-listen unix:///tmp/agentscope.sock]")
	fmt.Fprintln(w, "  agentscope connect-eliza [-server-url http://localhost:3000] [-api-key TOKEN | -prompt] [-cloud-login]")
	fmt.Fprintln(w, "  agentscope doctor-eliza [-server-url http://localhost:3000] [-api-key TOKEN | -use-keychain] [-listen unix:///tmp/agentscope.sock]")
	fmt.Fprintln(w, "  agentscope dashboard [-data snapshot.json] [-width 120]")
	fmt.Fprintln(w, "  agentscope events [-file events.ndjson]")
	fmt.Fprintln(w, "  agentscope monitor [-data snapshot.json] [-file events.ndjson | -listen tcp://127.0.0.1:7777] [-once] [-width 140] [-height 40]")
	fmt.Fprintln(w, "  agentscope sample-events")
	fmt.Fprintln(w, "  agentscope version")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "The default console command launches the Bubble Tea control room with embedded sample data when no live stream is provided.")
}
