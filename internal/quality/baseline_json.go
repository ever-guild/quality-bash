package quality

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const BaselineSchemaURL = "https://raw.githubusercontent.com/ever-guild/quality-bash/v1.0.0-alpha.2/internal/quality/schema/baseline-v2.schema.json"

//go:embed schema/baseline-v2.schema.json
var BaselineSchema []byte

//go:embed schema/baseline-v1.schema.json
var legacyBaselineSchema []byte

var compiledLegacySchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	return compileBaselineSchema("urn:quality-bash:baseline:v1", legacyBaselineSchema)
})

var compiledBaselineSchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	return compileBaselineSchema(BaselineSchemaURL, BaselineSchema)
})

func compileBaselineSchema(url string, data []byte) (*jsonschema.Schema, error) {
	var doc any
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := d.Decode(&doc); err != nil {
		return nil, err
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource(url, doc); err != nil {
		return nil, err
	}
	return c.Compile(url)
}

type baselineV2 struct {
	SchemaURL string   `json:"$schema,omitempty"`
	Schema    int      `json:"schema"`
	Files     []string `json:"files"`
	Method    Method   `json:"method"`
	Coverage  struct {
		Covered uint64 `json:"coveredLines"`
		Total   uint64 `json:"totalLines"`
	} `json:"coverage"`
	Mutation struct {
		Killed   uint64 `json:"killed"`
		Survived uint64 `json:"survived"`
	} `json:"mutation"`
}

type baselineLegacy Baseline

func comparableMethod(m Method) Method {
	parameters := make(map[string]string)
	for k, v := range m.Parameters {
		if k != "jobs" && k != "timeout" {
			parameters[k] = v
		}
	}
	if len(parameters) == 0 {
		parameters = nil
	}
	m.Parameters = parameters
	return m
}

func (b Baseline) MarshalJSON() ([]byte, error) {
	if b.Schema != 2 {
		return json.Marshal(baselineLegacy(b))
	}
	if err := b.Coverage.Validate(); err != nil {
		return nil, err
	}
	if err := b.Mutation.Validate(); err != nil {
		return nil, err
	}
	v := baselineV2{SchemaURL: BaselineSchemaURL, Schema: 2, Files: b.Files, Method: comparableMethod(b.Method)}
	v.Coverage.Covered, v.Coverage.Total = b.Coverage.Numerator, b.Coverage.Denominator
	v.Mutation.Killed, v.Mutation.Survived = b.Mutation.Numerator, b.Mutation.Denominator-b.Mutation.Numerator
	return json.Marshal(v)
}

func (b *Baseline) UnmarshalJSON(data []byte) error {
	if err := rejectDuplicateKeys(data); err != nil {
		return err
	}
	var header struct {
		Schema int `json:"schema"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return err
	}
	if header.Schema == 1 || header.Schema == 0 {
		if header.Schema == 1 {
			var doc any
			d := json.NewDecoder(bytes.NewReader(data))
			d.UseNumber()
			if err := d.Decode(&doc); err != nil {
				return err
			}
			schema, err := compiledLegacySchema()
			if err != nil {
				return err
			}
			if err := schema.Validate(doc); err != nil {
				return fmt.Errorf("legacy baseline schema: %w", err)
			}
		}
		var old baselineLegacy
		d := json.NewDecoder(bytes.NewReader(data))
		d.DisallowUnknownFields()
		if err := d.Decode(&old); err != nil {
			return err
		}
		*b = Baseline(old)
		return nil
	}
	if header.Schema != 2 {
		return fmt.Errorf("unsupported baseline schema %d", header.Schema)
	}
	var doc any
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := d.Decode(&doc); err != nil {
		return err
	}
	schema, err := compiledBaselineSchema()
	if err != nil {
		return err
	}
	if err := schema.Validate(doc); err != nil {
		return fmt.Errorf("baseline schema: %w", err)
	}
	var v baselineV2
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	if v.Mutation.Survived > math.MaxUint64-v.Mutation.Killed {
		return fmt.Errorf("mutation count overflow")
	}
	result := Baseline{Schema: 2, Files: v.Files, Method: v.Method, Coverage: Ratio{v.Coverage.Covered, v.Coverage.Total}, Mutation: Ratio{v.Mutation.Killed, v.Mutation.Killed + v.Mutation.Survived}}
	if err := result.Coverage.Validate(); err != nil {
		return err
	}
	if err := result.Mutation.Validate(); err != nil {
		return err
	}
	*b = result
	return nil
}

// The standard decoder accepts repeated keys; a policy document must not.
func rejectDuplicateKeys(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var value func(int) error
	value = func(depth int) error {
		if depth > 100 {
			return fmt.Errorf("JSON nesting exceeds 100 levels")
		}
		token, err := d.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok {
					return fmt.Errorf("invalid object key")
				}
				if seen[name] {
					return fmt.Errorf("duplicate JSON key %q", name)
				}
				seen[name] = true
				if err := value(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for d.More() {
				if err := value(depth + 1); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("unexpected JSON delimiter")
		}
		_, err = d.Token()
		return err
	}
	if err := value(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("trailing JSON data")
	}
	return nil
}
