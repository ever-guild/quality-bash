package quality

import "testing"

func TestOutputCannotHideSelectedInputs(t *testing.T) {
	s := Selection{Scripts: []string{"src/a.sh"}, Tests: []string{"a.bats"}}
	for _, tc := range []struct{ reports, baseline string }{{"/project", "baseline.json"}, {"src", "baseline.json"}, {"reports", "src/a.sh"}, {"reports", "reports/baseline.json"}} {
		if ValidateOutputPaths("/project", tc.reports, tc.baseline, s) == nil {
			t.Fatalf("accepted %+v", tc)
		}
	}
	if err := ValidateOutputPaths("/project", "reports", "baseline.json", s); err != nil {
		t.Fatal(err)
	}
}
