package quality

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBatsAbortLifecycle(t *testing.T) {
	if _, err := exec.LookPath("bats"); err != nil {
		t.Skip("bats required")
	}
	for _, mode := range []string{"test", "setup", "teardown", "survived"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			script := "#!/bin/bash\nexit 0\n"
			tests := `setup() { if [[ "$MODE" == setup ]]; then false; fi; }
teardown() { echo teardown >> "$TRACE"; if [[ "$MODE" == teardown ]]; then false; fi; }
@test "first" { if [[ "$MODE" == test ]]; then false; fi; }
@test "second" { echo second >> "$TRACE"; }
`
			trace := filepath.Join(t.TempDir(), "trace")
			t.Setenv("MODE", mode)
			t.Setenv("TRACE", trace)
			for name, data := range map[string]string{"a.sh": script, "a.bats": tests, "b.bats": `@test "other file" { echo other >> "$TRACE"; }`} {
				if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0700); err != nil {
					t.Fatal(err)
				}
			}
			s, err := Discover(root, []string{"a.sh"}, []string{"a.bats", "b.bats"})
			if err != nil {
				t.Fatal(err)
			}
			mutations, err := GenerateMutations("a.sh", script)
			if err != nil || len(mutations) != 1 {
				t.Fatalf("%v %v", mutations, err)
			}
			r := Runner{Root: root, Reports: filepath.Join(root, "reports"), Timeout: 5 * time.Second}
			got := r.runMutant(context.Background(), s, mutations[0], r.Reports)
			want := "detected"
			if mode == "survived" {
				want = "survived"
			}
			if got != want {
				t.Fatalf("got %s want %s", got, want)
			}
			data, err := os.ReadFile(trace)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), "teardown") {
				t.Fatal("teardown omitted")
			}
			if strings.Contains(string(data), "other") != (mode == "survived") {
				t.Fatalf("unexpected next file execution: %s", data)
			}
			if strings.Contains(string(data), "second") != (mode == "survived") {
				t.Fatalf("unexpected execution: %s", data)
			}
		})
	}
}

func TestMutationControlMustComplete(t *testing.T) {
	if _, err := exec.LookPath("bats"); err != nil {
		t.Skip("bats required")
	}
	root := t.TempDir()
	trace := filepath.Join(t.TempDir(), "trace")
	t.Setenv("TRACE", trace)
	for name, data := range map[string]string{"a.sh": "#!/bin/bash\nexit 0\n", "a.bats": `@test "fail" { false; }
@test "still runs" { echo control >> "$TRACE"; }
`} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0700); err != nil {
			t.Fatal(err)
		}
	}
	s, err := Discover(root, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	r := Runner{Root: root, Reports: filepath.Join(root, "reports"), Timeout: 5 * time.Second}
	report, err := r.Mutate(context.Background(), s, 1)
	if err == nil || len(report.Mutations) != 0 {
		t.Fatal("failed control allowed mutants")
	}
	if _, err := os.Stat(trace); err != nil {
		t.Fatal("control stopped early", err)
	}
}

func TestMutantProcessFailures(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
		cancel           bool
	}{
		{"truncated", "printf '1..2\\nnot ok 1 failed\\n'; exit 1", "infrastructure-error", false},
		{"runner", "exit 127", "infrastructure-error", false},
		{"timeout", "sleep 5", "timeout", false},
		{"cancelled", "sleep 5", "infrastructure-error", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			bin := t.TempDir()
			if err := os.WriteFile(filepath.Join(bin, "bats"), []byte("#!/bin/bash\n"+tc.body+"\n"), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
			source := "#!/bin/bash\nexit 0\n"
			if err := os.WriteFile(filepath.Join(root, "a.sh"), []byte(source), 0700); err != nil {
				t.Fatal(err)
			}
			mutations, err := GenerateMutations("a.sh", source)
			if err != nil {
				t.Fatal(err)
			}
			r := Runner{Root: root, Reports: filepath.Join(root, "reports"), Timeout: 100 * time.Millisecond}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancel {
				cancel()
			}
			if got := r.runMutant(ctx, Selection{Scripts: []string{"a.sh"}, Tests: []string{"a.bats"}}, mutations[0], r.Reports); got != tc.want {
				t.Fatalf("got %s want %s", got, tc.want)
			}
		})
	}
}
