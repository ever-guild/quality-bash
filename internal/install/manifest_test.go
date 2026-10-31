package install

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestManifestRequiresBothArchitectures(t *testing.T) {
	m := Manifest{Schema: 1}
	for _, arch := range []string{"amd64", "arm64"} {
		for _, name := range []string{"bash", "bats", "shellcheck", "kcov"} {
			m.Packages = append(m.Packages, Package{Name: name, Version: "1", OS: "linux", Arch: arch, URL: "https://example.invalid/tool.tar.gz", SHA256: strings.Repeat("ab", 32), Executables: []string{"bin/" + name}})
		}
	}
	encode := func(m Manifest) []byte {
		data, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	if _, err := ParseManifest(encode(m)); err != nil {
		t.Fatal(err)
	}
	missing := m
	missing.Packages = m.Packages[:7]
	if _, err := ParseManifest(encode(missing)); err == nil {
		t.Fatal("missing platform accepted")
	}
	duplicate := m
	duplicate.Packages = append(append([]Package{}, m.Packages...), m.Packages[0])
	if _, err := ParseManifest(encode(duplicate)); err == nil {
		t.Fatal("duplicate accepted")
	}
	m.Packages[0].SHA256 = "bad"
	if _, err := ParseManifest(encode(m)); err == nil {
		t.Fatal("bad checksum accepted")
	}
}
