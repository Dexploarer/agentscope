package agent

import (
	"bytes"
	_ "embed"
)

var (
	//go:embed sample_snapshot.json
	sampleSnapshotData []byte

	//go:embed sample_events.ndjson
	sampleEventsData []byte
)

func SampleSnapshot() (Snapshot, error) {
	return LoadSnapshot(bytes.NewReader(sampleSnapshotData))
}

func SampleEvents() ([]Event, error) {
	return LoadEvents(bytes.NewReader(sampleEventsData))
}

func SampleEventsNDJSON() []byte {
	return bytes.Clone(sampleEventsData)
}
