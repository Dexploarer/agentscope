package agent

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode"
)

var ansiSequencePattern = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

func LoadSnapshot(r io.Reader) (Snapshot, error) {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()

	var snapshot Snapshot
	if err := decoder.Decode(&snapshot); err != nil {
		return Snapshot{}, fmt.Errorf("decode snapshot: %w", err)
	}
	if err := expectEOF(decoder); err != nil {
		return Snapshot{}, fmt.Errorf("decode snapshot: %w", err)
	}

	snapshot.normalize()
	if err := snapshot.validate(); err != nil {
		return Snapshot{}, err
	}

	return snapshot, nil
}

func LoadEvents(r io.Reader) ([]Event, error) {
	var events []Event

	if err := StreamEvents(r, func(event Event) error {
		events = append(events, event)
		return nil
	}); err != nil {
		return nil, err
	}

	return events, nil
}

func StreamEvents(r io.Reader, handle func(Event) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lineNumber := 0

	for scanner.Scan() {
		lineNumber++

		raw := bytes.TrimSpace(scanner.Bytes())
		if len(raw) == 0 {
			continue
		}

		event, err := decodeEvent(raw)
		if err != nil {
			return fmt.Errorf("event line %d: %w", lineNumber, err)
		}
		if err := handle(event); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan events: %w", err)
	}

	return nil
}

func (s *Snapshot) normalize() {
	s.Workspace = sanitizeText(s.Workspace)

	for index := range s.Agents {
		s.Agents[index].normalize()
	}
	for index := range s.Channels {
		s.Channels[index].normalize()
	}
	for index := range s.Events {
		s.Events[index].normalize()
	}
}

func (s Snapshot) validate() error {
	if s.Workspace == "" {
		return errors.New("snapshot workspace is required")
	}
	if s.UpdatedAt.IsZero() {
		return errors.New("snapshot updatedAt is required")
	}
	if s.Queue.Pending < 0 || s.Queue.Running < 0 || s.Queue.Failed < 0 {
		return errors.New("queue counts must be zero or greater")
	}

	for index, current := range s.Agents {
		if err := current.validate(); err != nil {
			return fmt.Errorf("agent %d: %w", index, err)
		}
	}
	for index, current := range s.Channels {
		if err := current.validate(); err != nil {
			return fmt.Errorf("channel %d: %w", index, err)
		}
	}
	for index, current := range s.Events {
		if err := current.validate(); err != nil {
			return fmt.Errorf("event %d: %w", index, err)
		}
	}

	return nil
}

func (a *Agent) normalize() {
	a.Name = sanitizeText(a.Name)
	a.Role = sanitizeText(a.Role)
	a.Status = strings.ToLower(sanitizeText(a.Status))
	a.Model = sanitizeText(a.Model)
	a.LastEvent = sanitizeText(a.LastEvent)
}

func (a Agent) validate() error {
	if a.Name == "" {
		return errors.New("name is required")
	}
	if a.Role == "" {
		return errors.New("role is required")
	}
	if a.Status == "" {
		return errors.New("status is required")
	}
	if a.Tasks < 0 {
		return errors.New("tasks must be zero or greater")
	}
	return nil
}

func (c *Channel) normalize() {
	c.Name = sanitizeText(c.Name)
	c.Topic = sanitizeText(c.Topic)
	c.Status = strings.ToLower(sanitizeText(c.Status))
	c.LastEvent = sanitizeText(c.LastEvent)
	for index := range c.Members {
		c.Members[index] = sanitizeText(c.Members[index])
	}
	c.Members = compactStrings(c.Members)
}

func (c Channel) validate() error {
	if c.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

func (e *Event) normalize() {
	e.Agent = sanitizeText(e.Agent)
	e.Channel = sanitizeText(e.Channel)
	e.Kind = strings.ToLower(sanitizeText(e.Kind))
	e.Status = strings.ToLower(sanitizeText(e.Status))
	e.Topic = sanitizeText(e.Topic)
	e.Source = sanitizeText(e.Source)
	e.RunID = sanitizeText(e.RunID)
	e.RoomID = sanitizeText(e.RoomID)
	e.WorldID = sanitizeText(e.WorldID)
	for index := range e.Members {
		e.Members[index] = sanitizeText(e.Members[index])
	}
	e.Members = compactStrings(e.Members)
	e.Message = sanitizeText(e.Message)
}

func (e Event) validate() error {
	if e.Time.IsZero() {
		return errors.New("time is required")
	}
	if e.Agent == "" {
		return errors.New("agent is required")
	}
	if e.Kind == "" {
		return errors.New("kind is required")
	}
	if e.Message == "" {
		return errors.New("message is required")
	}
	return nil
}

func decodeEvent(raw []byte) (Event, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()

	var event Event
	if err := decoder.Decode(&event); err != nil {
		return Event{}, fmt.Errorf("decode: %w", err)
	}
	if err := expectEOF(decoder); err != nil {
		return Event{}, fmt.Errorf("decode: %w", err)
	}

	event.normalize()
	if err := event.validate(); err != nil {
		return Event{}, fmt.Errorf("validate: %w", err)
	}

	return event, nil
}

func sanitizeText(value string) string {
	value = ansiSequencePattern.ReplaceAllString(value, "")
	value = strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			return ' '
		case unicode.IsPrint(r):
			return r
		default:
			return -1
		}
	}, value)

	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func compactStrings(values []string) []string {
	compacted := values[:0]
	for _, current := range values {
		if current == "" {
			continue
		}
		compacted = append(compacted, current)
	}
	return compacted
}

func expectEOF(decoder *json.Decoder) error {
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("unexpected trailing JSON value")
		}
		return err
	}
	return nil
}
