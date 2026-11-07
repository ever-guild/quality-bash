package quality

import (
	"fmt"
	"math/big"
	"reflect"
)

// Ratio retains exact counts; percentages are presentation only.
type Ratio struct {
	Numerator   uint64 `json:"numerator"`
	Denominator uint64 `json:"denominator"`
}

func (r Ratio) Validate() error {
	if r.Denominator == 0 || r.Numerator > r.Denominator {
		return fmt.Errorf("invalid quality ratio %d/%d", r.Numerator, r.Denominator)
	}
	return nil
}
func (r Ratio) AtLeast(b Ratio) bool {
	if r.Validate() != nil || b.Validate() != nil {
		return false
	}
	left := new(big.Int).Mul(new(big.Int).SetUint64(r.Numerator), new(big.Int).SetUint64(b.Denominator))
	right := new(big.Int).Mul(new(big.Int).SetUint64(b.Numerator), new(big.Int).SetUint64(r.Denominator))
	return left.Cmp(right) >= 0
}

type Method struct {
	Coverage   string            `json:"coverage"`
	Tools      map[string]string `json:"tools"`
	Mutation   string            `json:"mutation"`
	Operators  []string          `json:"operators"`
	Parameters map[string]string `json:"parameters"`
}
type Baseline struct {
	Schema   int      `json:"schema"`
	Files    []string `json:"files"`
	Method   Method   `json:"method"`
	Coverage Ratio    `json:"coverage"`
	Mutation Ratio    `json:"mutation"`
}

func (b Baseline) Check(current Baseline) error {
	if (b.Schema != 1 && b.Schema != 2) || (current.Schema != 1 && current.Schema != 2) {
		return fmt.Errorf("unsupported baseline schema")
	}
	if !reflect.DeepEqual(b.Files, current.Files) || !reflect.DeepEqual(comparableMethod(b.Method), comparableMethod(current.Method)) {
		return fmt.Errorf("incompatible scope or methodology: explicit baseline migration required")
	}
	if err := b.Coverage.Validate(); err != nil {
		return err
	}
	if err := b.Mutation.Validate(); err != nil {
		return err
	}
	if err := current.Coverage.Validate(); err != nil {
		return err
	}
	if err := current.Mutation.Validate(); err != nil {
		return err
	}
	coverageOK := current.Coverage.AtLeast(b.Coverage)
	mutationOK := current.Mutation.AtLeast(b.Mutation)
	if !coverageOK || !mutationOK {
		return fmt.Errorf("quality regression: coverage passed=%t, mutation passed=%t", coverageOK, mutationOK)
	}
	return nil
}
