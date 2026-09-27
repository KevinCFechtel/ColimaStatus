package colima

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type profileJSON struct {
	Name    string         `json:"name"`
	Status  string         `json:"status"`
	Arch    string         `json:"arch"`
	CPUs    flexibleNumber `json:"cpus"`
	Memory  flexibleNumber `json:"memory"`
	Disk    flexibleNumber `json:"disk"`
	Runtime string         `json:"runtime"`
	Address string         `json:"address"`
}

// flexibleNumber decodes a JSON number that a future Colima may report as a
// string, and tolerates a value it cannot interpret by staying zero. These
// fields are cosmetic detail in the menu; failing the whole status read over
// one of them would hide the profile state, which is the part that matters.
type flexibleNumber int64

func (number *flexibleNumber) UnmarshalJSON(data []byte) error {
	text := strings.TrimSpace(string(data))
	if text == "" || text == "null" {
		return nil
	}
	if unquoted, err := strconv.Unquote(text); err == nil {
		text = strings.TrimSpace(unquoted)
	}
	if value, err := strconv.ParseInt(text, 10, 64); err == nil {
		*number = flexibleNumber(value)
		return nil
	}
	if value, err := strconv.ParseFloat(text, 64); err == nil {
		*number = flexibleNumber(value)
	}
	return nil
}

// ParseProfiles accepts both Colima's JSON object stream and a JSON array so
// the app remains compatible with old and future list output variants. A single
// unreadable entry is skipped rather than failing the whole read; an error is
// only returned when nothing at all could be interpreted.
func ParseProfiles(reader io.Reader) ([]Profile, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("Colima status could not be read: %w", err)
	}
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, nil
	}

	entries, err := splitProfileEntries(data)
	if err != nil {
		return nil, err
	}

	profiles := make([]Profile, 0, len(entries))
	var firstEntryErr error
	for _, entry := range entries {
		var raw profileJSON
		if err := json.Unmarshal(entry, &raw); err != nil {
			if firstEntryErr == nil {
				firstEntryErr = err
			}
			continue
		}
		profiles = append(profiles, Profile{
			Name:      raw.Name,
			State:     parseState(raw.Status),
			RawStatus: strings.TrimSpace(raw.Status),
			Arch:      raw.Arch,
			CPUs:      int(raw.CPUs),
			Memory:    int64(raw.Memory),
			Disk:      int64(raw.Disk),
			Runtime:   raw.Runtime,
			Address:   raw.Address,
		})
	}

	if len(profiles) == 0 && firstEntryErr != nil {
		return nil, fmt.Errorf("Colima status is not valid JSON: %w", firstEntryErr)
	}
	return profiles, nil
}

// splitProfileEntries separates the raw output into one JSON document per
// profile without interpreting their contents yet.
func splitProfileEntries(data []byte) ([]json.RawMessage, error) {
	if data[0] == '[' {
		var entries []json.RawMessage
		if err := json.Unmarshal(data, &entries); err != nil {
			return nil, fmt.Errorf("Colima status is not valid JSON: %w", err)
		}
		return entries, nil
	}

	var entries []json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(data))
	for {
		var entry json.RawMessage
		if err := decoder.Decode(&entry); err != nil {
			if err == io.EOF {
				break
			}
			if len(entries) == 0 {
				return nil, fmt.Errorf("Colima status is not valid JSON: %w", err)
			}
			// Trailing garbage after readable entries is ignored: what was
			// already decoded still describes the profiles.
			break
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func parseState(status string) State {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "running":
		return StateRunning
	case "stopped":
		return StateStopped
	case "broken":
		return StateBroken
	default:
		return StateUnknown
	}
}
