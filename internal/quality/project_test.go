package quality

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoveryAndFreshness(t *testing.T) {
	root := t.TempDir()
	write := func(name, data string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("some script.sh", "echo yes")
	if _, err := Discover(root, nil, nil); err == nil {
		t.Fatal("missing tests accepted")
	}
	write("some script.bats", "@test example { true; }")
	selection, err := Discover(root, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(selection.Scripts) != 1 || selection.Tests[0] != "some script.bats" {
		t.Fatal(selection)
	}
	excluded := []string{"reports", "quality-baseline.json"}
	first, err := Fingerprint(root, excluded, selection)
	if err != nil {
		t.Fatal(err)
	}
	write("quality-baseline.json", "ignored")
	same, err := Fingerprint(root, excluded, selection)
	if err != nil || first != same {
		t.Fatal("baseline invalidated measurement", err)
	}
	write("helper.txt", "changed helper")
	changed, err := Fingerprint(root, excluded, selection)
	if err != nil || changed == first {
		t.Fatal("helper omitted from fingerprint", err)
	}
	if _, err := Discover(root, []string{"../outside.sh"}, nil); err == nil {
		t.Fatal("escape accepted")
	}
}

func TestUnmatchedTestsRequireExplicitSelection(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a.sh", "a.bats", "integration.bats"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("# fixture\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Discover(root, nil, nil); err == nil {
		t.Fatal("unmatched test silently ignored")
	}
	s, err := Discover(root, []string{"a.sh"}, []string{"a.bats", "integration.bats"})
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Tests) != 2 {
		t.Fatal(s)
	}
}

func TestExternalReportsDoNotInvalidateFingerprint(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "source.sh"), []byte("true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a, err := Fingerprint(root, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Fingerprint(root, []string{outside}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatal("external exclusion changed fingerprint")
	}
}
