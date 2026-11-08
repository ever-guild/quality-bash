package qualitybash

import _ "embed"

const Version = "v1.0.0-alpha.5"

// The schema version changes only when the baseline format changes.
const BaselineSchemaURL = "https://raw.githubusercontent.com/ever-guild/quality-bash/v1.0.0-alpha.3/baseline.schema.json"

//go:embed baseline.schema.json
var BaselineSchema []byte
