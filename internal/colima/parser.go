package colima

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type profileJSON struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Arch    string `json:"arch"`
	CPUs    int    `json:"cpus"`
	Memory  int64  `json:"memory"`
	Disk    int64  `json:"disk"`
	Runtime string `json:"runtime"`
	Address string `json:"address"`
}

// ParseProfiles accepts both Colima's JSON object stream and a JSON array so
// the app remains compatible with old and future list output variants.
func ParseProfiles(reader io.Reader) ([]Profile, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("Colima status could not be read: %w", err)
	}
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, nil
	}

	var rawProfiles []profileJSON
	if data[0] == '[' {
		if err := json.Unmarshal(data, &rawProfiles); err != nil {
			return nil, fmt.Errorf("Colima status is not valid JSON: %w", err)
		}
	} else {
		decoder := json.NewDecoder(bytes.NewReader(data))
		for {
			var raw profileJSON
			if err := decoder.Decode(&raw); err != nil {
				if err == io.EOF {
					break
				}
				return nil, fmt.Errorf("Colima status is not valid JSON: %w", err)
			}
			rawProfiles = append(rawProfiles, raw)
		}
	}

	profiles := make([]Profile, 0, len(rawProfiles))
	for _, raw := range rawProfiles {
		profiles = append(profiles, Profile{
			Name:      raw.Name,
			State:     parseState(raw.Status),
			RawStatus: raw.Status,
			Arch:      raw.Arch,
			CPUs:      raw.CPUs,
			Memory:    raw.Memory,
			Disk:      raw.Disk,
			Runtime:   raw.Runtime,
			Address:   raw.Address,
		})
	}
	return profiles, nil
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
