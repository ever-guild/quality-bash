package quality

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"sync"

	qualitybash "github.com/ever-guild/quality-bash"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

var compiledBaselineSchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	var doc any
	d := json.NewDecoder(bytes.NewReader(qualitybash.BaselineSchema))
	d.UseNumber()
	if err := d.Decode(&doc); err != nil {
		return nil, err
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource(qualitybash.BaselineSchemaURL, doc); err != nil {
		return nil, err
	}
	return c.Compile(qualitybash.BaselineSchemaURL)
})

type baselineDocument struct {
	SchemaURL string   `json:"$schema"`
	Files     []string `json:"files"`
	Method    Method   `json:"method"`
	Coverage  struct {
		LinePercent json.Number `json:"linePercent"`
	} `json:"coverage"`
	Mutation struct {
		Score json.Number `json:"score"`
	} `json:"mutation"`
}

func (b Baseline) MarshalJSON() ([]byte, error) {
	if err := b.Coverage.Validate(); err != nil {
		return nil, err
	}
	if err := b.Mutation.Validate(); err != nil {
		return nil, err
	}
	v := baselineDocument{SchemaURL: qualitybash.BaselineSchemaURL, Files: b.Files, Method: b.Method}
	coverage := b.Coverage.scoreHundredths()
	v.Coverage.LinePercent = json.Number(fmt.Sprintf("%d.%02d", coverage/100, coverage%100))
	score := b.Mutation.scoreHundredths()
	v.Mutation.Score = json.Number(fmt.Sprintf("%d.%02d", score/100, score%100))
	return json.Marshal(v)
}

func (b *Baseline) UnmarshalJSON(data []byte) error {
	if err := rejectDuplicateKeys(data); err != nil {
		return err
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
	var v baselineDocument
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	mutation, err := parsePercent(v.Mutation.Score)
	if err != nil {
		return err
	}
	coverage, err := parsePercent(v.Coverage.LinePercent)
	if err != nil {
		return err
	}
	result := Baseline{Files: v.Files, Method: v.Method, Coverage: coverage, Mutation: mutation}
	if err := result.Coverage.Validate(); err != nil {
		return err
	}
	if err := result.Mutation.Validate(); err != nil {
		return err
	}
	*b = result
	return nil
}

func parsePercent(value json.Number) (Ratio, error) {
	score, ok := new(big.Rat).SetString(string(value))
	if !ok {
		return Ratio{}, fmt.Errorf("invalid percentage")
	}
	score.Mul(score, big.NewRat(100, 1))
	if !score.IsInt() || !score.Num().IsUint64() || score.Num().Uint64() > 10000 {
		return Ratio{}, fmt.Errorf("percentage must be 0..100 with at most two decimal places")
	}
	return Ratio{score.Num().Uint64(), 10000}, nil
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
