package quality

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	qualitybash "github.com/ever-guild/quality-bash"
)

func validBaseline(t *testing.T) Baseline {
	t.Helper()
	data, err := os.ReadFile("../../testdata/basic/quality-baseline.json")
	if err != nil {
		t.Fatal(err)
	}
	var b Baseline
	if err = json.Unmarshal(data, &b); err != nil {
		t.Fatal(err)
	}
	return b
}

func TestBaselineSchema(t *testing.T) {
	b := validBaseline(t)
	b.Mutation = Ratio{2, 3}
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, `"schema":`) || strings.Contains(text, "numerator") || strings.Contains(text, "killed") || !strings.Contains(text, `"score":66.67`) || !strings.Contains(text, qualitybash.BaselineSchemaURL) {
		t.Fatal(text)
	}
	var restored Baseline
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if err := restored.Check(b); err != nil {
		t.Fatal("rounded baseline rejected original measurement:", err)
	}
	for name, bad := range map[string]string{
		"legacy version":    strings.Replace(text, `"files":`, `"schema":2,"files":`, 1),
		"missing schema":    strings.Replace(text, `"$schema":"`+qualitybash.BaselineSchemaURL+`",`, "", 1),
		"old schema":        strings.Replace(text, qualitybash.BaselineSchemaURL, "https://raw.githubusercontent.com/ever-guild/quality-bash/v1.0.0-alpha.2/internal/quality/schema/baseline-v2.schema.json", 1),
		"duplicate":         strings.Replace(text, `"score":66.67`, `"score":66.67,"score":66.67`, 1),
		"unknown":           strings.Replace(text, `"score":66.67`, `"score":66.67,"killed":2`, 1),
		"negative":          strings.Replace(text, `"score":66.67`, `"score":-1`, 1),
		"over 100":          strings.Replace(text, `"score":66.67`, `"score":100.01`, 1),
		"precision":         strings.Replace(text, `"score":66.67`, `"score":66.666`, 1),
		"string":            strings.Replace(text, `"score":66.67`, `"score":"66.67"`, 1),
		"null":              strings.Replace(text, `"score":66.67`, `"score":null`, 1),
		"coverage overflow": strings.Replace(text, `"linePercent":60.00`, `"linePercent":100.01`, 1),
		"coverage empty":    strings.Replace(text, `"linePercent":60.00`, `"linePercent":null`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			var got Baseline
			if json.Unmarshal([]byte(bad), &got) == nil {
				t.Fatalf("accepted %s", bad)
			}
		})
	}
	for _, score := range []string{"0", "100", "66.67", "6.667e1"} {
		var got Baseline
		if err := json.Unmarshal([]byte(strings.Replace(text, `"score":66.67`, `"score":`+score, 1)), &got); err != nil {
			t.Fatal(score, err)
		}
	}
}

func TestMutationScoreRounding(t *testing.T) {
	for _, tc := range []struct {
		ratio Ratio
		want  uint64
	}{
		{Ratio{0, 1}, 0}, {Ratio{1, 1}, 10000}, {Ratio{2, 3}, 6667},
		{Ratio{1, 3}, 3333}, {Ratio{1, 20000}, 1}, {Ratio{1, 20001}, 0},
		{Ratio{math.MaxUint64 - 1, math.MaxUint64}, 10000},
	} {
		if got := tc.ratio.scoreHundredths(); got != tc.want {
			t.Fatalf("%v: %d, want %d", tc.ratio, got, tc.want)
		}
	}
	b := validBaseline(t)
	for _, tc := range []struct {
		ratio Ratio
		pass  bool
	}{
		{Ratio{2, 3}, true}, {Ratio{66665, 100000}, true},
		{Ratio{66664, 100000}, false}, {Ratio{1, 2}, false},
	} {
		current := b
		current.Mutation = tc.ratio
		if (b.Check(current) == nil) != tc.pass {
			t.Fatalf("gate for %v: want pass=%v", tc.ratio, tc.pass)
		}
	}
	// The on-disk rounded score must remain stable through repeated updates.
	path := filepath.Join(t.TempDir(), "baseline.json")
	m := Measurement{Fingerprint: "fresh", Qualified: true, Baseline: b}
	m.Baseline.Mutation = Ratio{2, 3}
	m.CoverageCounts = m.Baseline.Coverage
	m.MutationCounts = m.Baseline.Mutation
	if err := UpdateBaseline(path, m, "fresh", true); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	for i := 0; i < 3; i++ {
		if err := UpdateBaseline(path, m, "fresh", false); err != nil {
			t.Fatal(err)
		}
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("score drifted during updates")
	}
}

func TestCoveragePercentGateAndRoundTrip(t *testing.T) {
	b := validBaseline(t)
	b.Coverage = Ratio{2, 3}
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"linePercent":66.67`) {
		t.Fatal(string(data))
	}
	var policy Baseline
	if err := json.Unmarshal(data, &policy); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		coverage Ratio
		pass     bool
	}{{Ratio{2, 3}, true}, {Ratio{66665, 100000}, true}, {Ratio{66664, 100000}, false}} {
		current := b
		current.Coverage = tc.coverage
		if (policy.Check(current) == nil) != tc.pass {
			t.Fatal(tc)
		}
	}
	for _, value := range []string{"-1", "100.01", "66.666", "null", `"60"`} {
		bad := strings.Replace(string(data), `"linePercent":66.67`, `"linePercent":`+value, 1)
		if json.Unmarshal([]byte(bad), &policy) == nil {
			t.Fatal("invalid coverage accepted", value)
		}
	}
}
