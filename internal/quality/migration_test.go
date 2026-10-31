package quality

import (
	"path/filepath"
	"testing"
)

func TestExplicitMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "baseline.json")
	old := Baseline{Schema: 1, Files: []string{"a.sh"}, Coverage: Ratio{4, 5}, Mutation: Ratio{3, 5}}
	if err := WriteJSON(path, old); err != nil {
		t.Fatal(err)
	}
	m := Measurement{Fingerprint: "fresh", Qualified: true, Baseline: old}
	if MigrateBaseline(path, m, "fresh", "lower compatible threshold") == nil {
		t.Fatal("compatible migration accepted")
	}
	m.Baseline.Method.Coverage = "new method"
	if MigrateBaseline(path, m, "fresh", "") == nil {
		t.Fatal("unexplained migration accepted")
	}
	if err := MigrateBaseline(path, m, "fresh", "collector changed line counting"); err != nil {
		t.Fatal(err)
	}
	var record Migration
	if err := ReadJSON(path+".migration.json", &record); err != nil {
		t.Fatal(err)
	}
	if record.Previous.Method.Coverage != "" || record.Replacement.Method.Coverage != "new method" {
		t.Fatal(record)
	}
}
