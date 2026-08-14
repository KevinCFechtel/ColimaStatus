// Command icnspack creates a modern macOS ICNS container from a complete
// iconset directory. Recent iconutil versions may reject the same valid files
// that they can successfully extract, so the project packs the PNG chunks
// directly.
package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type iconEntry struct {
	kind string
	file string
}

var iconEntries = []iconEntry{
	{kind: "icp4", file: "icon_16x16.png"},
	{kind: "ic11", file: "icon_16x16@2x.png"},
	{kind: "icp5", file: "icon_32x32.png"},
	{kind: "ic12", file: "icon_32x32@2x.png"},
	{kind: "ic07", file: "icon_128x128.png"},
	{kind: "ic13", file: "icon_128x128@2x.png"},
	{kind: "ic08", file: "icon_256x256.png"},
	{kind: "ic14", file: "icon_256x256@2x.png"},
	{kind: "ic09", file: "icon_512x512.png"},
	{kind: "ic10", file: "icon_512x512@2x.png"},
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 2 {
		return errors.New("usage: icnspack <input.iconset> <output.icns>")
	}

	container := []byte{'i', 'c', 'n', 's', 0, 0, 0, 0}
	for _, entry := range iconEntries {
		data, err := os.ReadFile(filepath.Join(args[0], entry.file))
		if err != nil {
			return fmt.Errorf("read %s: %w", entry.file, err)
		}

		chunkSize := len(data) + 8
		if chunkSize > int(^uint32(0)) {
			return fmt.Errorf("%s is too large", entry.file)
		}
		container = append(container, entry.kind...)
		container = binary.BigEndian.AppendUint32(container, uint32(chunkSize))
		container = append(container, data...)
	}

	if len(container) > int(^uint32(0)) {
		return errors.New("ICNS container is too large")
	}
	binary.BigEndian.PutUint32(container[4:8], uint32(len(container)))

	if err := os.WriteFile(args[1], container, 0o644); err != nil {
		return fmt.Errorf("write ICNS: %w", err)
	}
	return nil
}
