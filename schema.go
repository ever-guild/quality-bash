package qualitybash

import _ "embed"

const Version = "v1.0.0-alpha.3"
const BaselineSchemaURL = "https://raw.githubusercontent.com/ever-guild/quality-bash/" + Version + "/baseline.schema.json"

//go:embed baseline.schema.json
var BaselineSchema []byte
