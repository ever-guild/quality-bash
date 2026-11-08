package install

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
)

type Manifest struct {
	Schema   int       `json:"schema"`
	Packages []Package `json:"packages"`
}

func ParseManifest(data []byte) (Manifest, error) {
	var manifest Manifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return manifest, err
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return manifest, fmt.Errorf("trailing manifest data")
	}
	if manifest.Schema != 1 {
		return manifest, fmt.Errorf("unsupported tool manifest schema")
	}
	seen := map[string]bool{}
	for _, p := range manifest.Packages {
		if p.Name == "bash" {
			return manifest, fmt.Errorf("Bash must be provided by the system, not installed from a tool manifest")
		}
		if err := p.Validate(); err != nil {
			return manifest, err
		}
		digest, err := hex.DecodeString(p.SHA256)
		if err != nil || len(digest) != 32 {
			return manifest, fmt.Errorf("invalid package checksum")
		}
		location, err := url.Parse(p.URL)
		if err != nil || location.Scheme != "https" || location.Host == "" || location.User != nil {
			return manifest, fmt.Errorf("invalid package URL")
		}
		seenDependencies := map[string]bool{}
		if len(p.Dependencies) > 64 {
			return manifest, fmt.Errorf("too many dependencies")
		}
		for _, a := range p.Dependencies {
			digest, err := hex.DecodeString(a.SHA256)
			if err != nil || len(digest) != 32 {
				return manifest, fmt.Errorf("invalid dependency checksum")
			}
			u, err := url.Parse(a.URL)
			if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
				return manifest, fmt.Errorf("invalid dependency URL")
			}
			if a.Name == "" || a.Version == "" || seenDependencies[a.Name] {
				return manifest, fmt.Errorf("invalid or duplicate dependency")
			}
			seenDependencies[a.Name] = true
		}
		key := p.Name + "/" + p.OS + "/" + p.Arch
		if seen[key] {
			return manifest, fmt.Errorf("duplicate tool platform: %s", key)
		}
		seen[key] = true
	}
	for _, arch := range []string{"amd64", "arm64"} {
		for _, name := range []string{"bats", "shellcheck", "kcov"} {
			if !seen[name+"/linux/"+arch] {
				return manifest, fmt.Errorf("missing package %s for %s", name, arch)
			}
		}
	}
	return manifest, nil
}
