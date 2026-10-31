package quality

import (
	"strings"
	"testing"
)

func TestCoverageScope(t *testing.T) {
	xml := `<coverage><sources><source>/project</source></sources><packages><package><classes>
 <class filename="sample.sh"><lines><line number="1" hits="2"/><line number="3" hits="0"/></lines></class>
 <class filename="sample.sh"><lines><line number="1" hits="0"/></lines></class>
 <class filename="helper.sh"><lines><line number="1" hits="9"/></lines></class>
 </classes></package></packages></coverage>`
	report, err := ReadCoverage(strings.NewReader(xml), "/project", []string{"sample.sh"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Score != (Ratio{1, 2}) || len(report.Files) != 1 || report.Files[0].Uncovered[0] != 3 {
		t.Fatalf("%+v", report)
	}
	if _, err = ReadCoverage(strings.NewReader(xml), "/project", []string{"missing.sh"}); err == nil {
		t.Fatal("missing trace accepted")
	}
	for _, bad := range []string{"", `<coverage/>`, strings.ReplaceAll(xml, `hits="2"`, `hits="0"`)} {
		if _, err := ReadCoverage(strings.NewReader(bad), "/project", []string{"sample.sh"}); err == nil {
			t.Fatal("empty tracing accepted")
		}
	}
}
