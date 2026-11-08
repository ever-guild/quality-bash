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

type CoverageMethod struct {
	Algorithm string `json:"algorithm"`
}
type Parser struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
type MutationMethodSettings struct {
	Algorithm string   `json:"algorithm"`
	Operators []string `json:"operators"`
	Parser    Parser   `json:"parser"`
}
type Method struct {
	Coverage CoverageMethod         `json:"coverage"`
	Mutation MutationMethodSettings `json:"mutation"`
	Tools    map[string]string      `json:"tools"`
}
type Baseline struct {
	Files    []string `json:"files"`
	Method   Method   `json:"method"`
	Coverage Ratio    `json:"coverage"`
	Mutation Ratio    `json:"mutation"`
}

func (b Baseline) Check(current Baseline) error {
	if !reflect.DeepEqual(b.Files, current.Files) || !reflect.DeepEqual(b.Method, current.Method) {
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
	coverageOK := current.Coverage.scoreHundredths() >= b.Coverage.scoreHundredths()
	mutationOK := current.Mutation.scoreHundredths() >= b.Mutation.scoreHundredths()
	if !coverageOK || !mutationOK {
		return fmt.Errorf("quality regression: coverage passed=%t, mutation passed=%t", coverageOK, mutationOK)
	}
	return nil
}

// scoreHundredths rounds a validated ratio to hundredths of a percent, half up.
// Integer arithmetic avoids floating-point drift and uint64 multiplication overflow.
func (r Ratio) scoreHundredths() uint64 {
	scaled := new(big.Int).Mul(new(big.Int).SetUint64(r.Numerator), big.NewInt(10000))
	denominator := new(big.Int).SetUint64(r.Denominator)
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(scaled, denominator, remainder)
	if remainder.Lsh(remainder, 1).Cmp(denominator) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	return quotient.Uint64()
}
