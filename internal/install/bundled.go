package install

import _ "embed"

// BundledManifest is populated from verified native CI artifacts before release.
// An empty file denotes a development build without a qualified tool set.
//
//go:embed tools-manifest.json
var BundledManifest []byte
