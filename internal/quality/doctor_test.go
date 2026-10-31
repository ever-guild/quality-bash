package quality

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestDoctorRetainsBuildSuffix(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash required")
	}
	output, err := exec.Command(bash, "--version").Output()
	if err != nil {
		t.Fatal(err)
	}
	versions, _ := Doctor(context.Background(), time.Second, nil)
	for _, tool := range versions {
		if tool.Name == "bash" {
			if tool.Version == "" || !strings.Contains(string(output), "version "+tool.Version+" ") {
				t.Fatalf("version token truncated: %q", tool.Version)
			}
			return
		}
	}
	t.Fatal("bash absent from doctor")
}
