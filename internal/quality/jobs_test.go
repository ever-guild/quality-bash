package quality

import (
	"runtime"
	"testing"
)

func TestParseJobs(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  int
	}{{"auto", AutoJobs}, {"1", 1}, {"4", 4}, {"16", 16}} {
		got, err := ParseJobs(tc.input)
		if err != nil || got != tc.want {
			t.Fatalf("%q: %d %v", tc.input, got, err)
		}
	}
	for _, input := range []string{"", "0", "-1", "AUTO", "1.5", "999999999999999999999999"} {
		if _, err := ParseJobs(input); err == nil {
			t.Fatalf("accepted %q", input)
		}
	}
}
func TestMutationWorkerBudget(t *testing.T) {
	previous := runtime.GOMAXPROCS(2)
	defer runtime.GOMAXPROCS(previous)
	if got := mutationWorkers(AutoJobs, 100); got != 2 {
		t.Fatal(got)
	}
	runtime.GOMAXPROCS(4)
	if got := mutationWorkers(AutoJobs, 100); got != 4 {
		t.Fatal(got)
	}
	for _, tc := range []struct{ jobs, mutants, want int }{
		{AutoJobs, 3, 3}, {1, 100, 1}, {16, 100, 16}, {16, 3, 3}, {int(^uint(0) >> 1), 3, 3},
	} {
		if got := mutationWorkers(tc.jobs, tc.mutants); got != tc.want {
			t.Fatalf("%+v: got %d", tc, got)
		}
	}
}
