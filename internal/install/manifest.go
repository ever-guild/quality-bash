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
		key := p.Name + "/" + p.OS + "/" + p.Arch
		if seen[key] {
			return manifest, fmt.Errorf("duplicate tool platform: %s", key)
		}
		seen[key] = true
	}
	for _, arch := range []string{"amd64", "arm64"} {
		for _, name := range []string{"bash", "bats", "shellcheck", "kcov"} {
			if !seen[name+"/linux/"+arch] {
				return manifest, fmt.Errorf("missing package %s for %s", name, arch)
			}
		}
	}
	return manifest, nil
}
