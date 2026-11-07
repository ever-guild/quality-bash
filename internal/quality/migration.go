package quality

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

type Migration struct {
	Reason      string   `json:"reason"`
	At          string   `json:"at"`
	Previous    Baseline `json:"previous"`
	Replacement Baseline `json:"replacement"`
}

func MigrateBaseline(path string, m Measurement, fingerprint, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("migration requires an explanation of incompatible scope or methodology")
	}
	if err := m.ValidateFresh(fingerprint); err != nil {
		return err
	}
	unlock, err := lockBaseline(path)
	if err != nil {
		return err
	}
	defer unlock()
	var previous Baseline
	if err = ReadJSON(path, &previous); err != nil {
		return err
	}
	if err := previous.Check(previous); err != nil {
		return err
	}
	if reflect.DeepEqual(previous.Files, m.Baseline.Files) && reflect.DeepEqual(comparableMethod(previous.Method), comparableMethod(m.Baseline.Method)) {
		return fmt.Errorf("baseline is compatible; use baseline update, which forbids lowering thresholds")
	}
	record := Migration{Reason: reason, At: time.Now().UTC().Format(time.RFC3339Nano), Previous: previous, Replacement: m.Baseline}
	if err = WriteJSON(path+".migration.json", record); err != nil {
		return err
	}
	return WriteJSON(path, m.Baseline)
}

// UpgradeBaseline changes only the representation, never the measured thresholds.
func UpgradeBaseline(path string) error {
	unlock, err := lockBaseline(path)
	if err != nil {
		return err
	}
	defer unlock()
	var previous Baseline
	if err := ReadJSON(path, &previous); err != nil {
		return err
	}
	if previous.Schema != 1 {
		return fmt.Errorf("schema upgrade requires a v1 baseline")
	}
	if err := previous.Check(previous); err != nil {
		return err
	}
	replacement := previous
	replacement.Schema = 2
	replacement.Method = comparableMethod(previous.Method)
	// Validate the emitted document before replacing either file.
	data, err := json.Marshal(replacement)
	if err != nil {
		return err
	}
	var validated Baseline
	if err := json.Unmarshal(data, &validated); err != nil {
		return err
	}
	record := Migration{Reason: "schema v1 to v2; thresholds preserved", At: time.Now().UTC().Format(time.RFC3339Nano), Previous: previous, Replacement: replacement}
	if err := WriteJSON(path+".migration.json", record); err != nil {
		return err
	}
	return WriteJSON(path, replacement)
}
