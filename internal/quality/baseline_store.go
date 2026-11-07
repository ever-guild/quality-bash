package quality

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Measurement struct {
	Fingerprint string   `json:"fingerprint"`
	Baseline    Baseline `json:"baseline"`
	Qualified   bool     `json:"qualified"`
}

func ReadJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := rejectDuplicateKeys(data); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(target); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	var extra any
	if err = decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("%s: trailing JSON data", path)
	}
	return nil
}
func WriteJSON(path string, value any) error {
	if err := rejectSymlinkComponents(path); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	directory, err := openOutputDirectory(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer directory.Close()
	// Linux procfs refers to the retained directory inode, not its mutable name.
	anchored := fmt.Sprintf("/proc/self/fd/%d", directory.Fd())
	file, err := os.CreateTemp(anchored, ".quality-bash-*")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(name, filepath.Join(anchored, filepath.Base(path)))
}
func (m Measurement) ValidateFresh(fingerprint string) error {
	if !m.Qualified {
		return fmt.Errorf("report is not qualified")
	}
	if fingerprint == "" || m.Fingerprint != fingerprint {
		return fmt.Errorf("stale report: sources, tests, helpers or settings changed; rerun measurements")
	}
	if (m.Baseline.Schema != 1 && m.Baseline.Schema != 2) || len(m.Baseline.Files) == 0 {
		return fmt.Errorf("invalid report scope or schema")
	}
	if err := m.Baseline.Coverage.Validate(); err != nil {
		return err
	}
	return m.Baseline.Mutation.Validate()
}
func UpdateBaseline(path string, m Measurement, fingerprint string, create bool) error {
	if err := m.ValidateFresh(fingerprint); err != nil {
		return err
	}
	unlock, err := lockBaseline(path)
	if err != nil {
		return err
	}
	defer unlock()
	var previous Baseline
	err = ReadJSON(path, &previous)
	if os.IsNotExist(err) {
		if !create {
			return fmt.Errorf("baseline does not exist: use --create explicitly")
		}
		// Exclusive creation prevents concurrent initializers from overwriting a gate.
		data, err := json.MarshalIndent(m.Baseline, "", "  ")
		if err != nil {
			return err
		}
		file, err := openOutputFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return err
		}
		_, writeErr := file.Write(append(data, '\n'))
		closeErr := file.Close()
		if writeErr != nil {
			return writeErr
		}
		return closeErr
	}
	if err != nil {
		return err
	}
	if create {
		return fmt.Errorf("baseline already exists; omit --create")
	}
	if previous.Schema == 1 && m.Baseline.Schema == 2 {
		return fmt.Errorf("baseline schema v1: run baseline update --upgrade-schema first")
	}
	if err = previous.Check(m.Baseline); err != nil {
		return err
	}
	return WriteJSON(path, m.Baseline)
}
