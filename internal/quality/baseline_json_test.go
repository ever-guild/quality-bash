package quality

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validV2(t *testing.T) Baseline {
	t.Helper()
	data, err := os.ReadFile("../../testdata/basic/quality-baseline.json")
	if err != nil {
		t.Fatal(err)
	}
	var b Baseline
	if err = json.Unmarshal(data, &b); err != nil {
		t.Fatal(err)
	}
	b.Schema = 2
	b.Method = comparableMethod(b.Method)
	return b
}

func TestBaselineV2ValidationAndUpgrade(t *testing.T) {
	b := validV2(t)
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, "numerator") || !strings.Contains(text, "coveredLines") || !strings.Contains(text, "killed") {
		t.Fatal(text)
	}
	for _, bad := range []string{
		strings.Replace(text, `"schema":2`, `"schema":2,"schema":2`, 1),
		strings.Replace(text, `"coveredLines":3`, `"coveredLines":6`, 1),
		strings.Replace(text, `"coveredLines":3`, `"coveredLines":-1`, 1),
		strings.Replace(text, `"coveredLines":3`, `"coveredLines":1.5`, 1),
		strings.Replace(text, `"totalLines":5`, `"totalLines":0`, 1),
		strings.Replace(strings.Replace(text, `"killed":2`, `"killed":0`, 1), `"survived":1`, `"survived":0`, 1),
	} {
		var got Baseline
		if json.Unmarshal([]byte(bad), &got) == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	for _, change := range []string{`"survived":18446744073709551615`, `"survived":null`, `"unknown":1`} {
		bad := strings.Replace(text, `"survived":1`, change, 1)
		var got Baseline
		if json.Unmarshal([]byte(bad), &got) == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	path := filepath.Join(t.TempDir(), "baseline.json")
	old := b
	old.Schema = 1
	old.Method.Parameters = map[string]string{"parser": b.Method.Parameters["parser"], "jobs": "2", "timeout": "5m0s"}
	if err := WriteJSON(path, old); err != nil {
		t.Fatal(err)
	}
	if err := UpgradeBaseline(path); err != nil {
		t.Fatal(err)
	}
	var upgraded Baseline
	if err := ReadJSON(path, &upgraded); err != nil {
		t.Fatal(err)
	}
	if upgraded.Schema != 2 || upgraded.Coverage != old.Coverage || upgraded.Mutation != old.Mutation {
		t.Fatal(upgraded)
	}
	if err := old.Check(upgraded); err != nil {
		t.Fatal(err)
	}
	lowered := upgraded
	lowered.Mutation = Ratio{1, 3}
	if old.Check(lowered) == nil {
		t.Fatal("format upgrade bypassed target threshold")
	}
	before, _ := os.ReadFile(path)
	if UpgradeBaseline(path) == nil {
		t.Fatal("repeated upgrade accepted")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("failed upgrade modified baseline")
	}
}

func TestUpgradeRejectsSymlinksBeforeWriting(t *testing.T) {
	for _, suffix := range []string{"", ".lock", ".migration.json"} {
		t.Run("baseline"+suffix, func(t *testing.T) {
			directory := t.TempDir()
			path := filepath.Join(directory, "baseline.json")
			external := filepath.Join(t.TempDir(), "private.json")
			old := validV2(t)
			old.Schema = 1
			if err := WriteJSON(external, old); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(external)
			if err != nil {
				t.Fatal(err)
			}
			if suffix != "" {
				if err := WriteJSON(path, old); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Symlink(external, path+suffix); err != nil {
				t.Fatal(err)
			}
			if err := UpgradeBaseline(path); err == nil || !strings.Contains(err.Error(), "symlink") {
				t.Fatalf("expected symlink rejection, got %v", err)
			}
			for _, artifact := range []string{".lock", ".migration.json"} {
				if artifact != suffix {
					if _, err := os.Lstat(path + artifact); !os.IsNotExist(err) {
						t.Fatalf("unexpected artifact %s: %v", artifact, err)
					}
				}
			}
			after, err := os.ReadFile(external)
			if err != nil || string(after) != string(before) {
				t.Fatal("external baseline changed", err)
			}
		})
	}
}
