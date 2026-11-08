package quality

import (
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
	if reflect.DeepEqual(previous.Files, m.Baseline.Files) && reflect.DeepEqual(previous.Method, m.Baseline.Method) {
		return fmt.Errorf("baseline is compatible; use baseline-update, which forbids lowering thresholds")
	}
	record := Migration{Reason: reason, At: time.Now().UTC().Format(time.RFC3339Nano), Previous: previous, Replacement: m.Baseline}
	if err = WriteJSON(path+".migration.json", record); err != nil {
		return err
	}
	return WriteJSON(path, m.Baseline)
}
