package install

import (
	"os/exec"
	"runtime"
	"testing"
)

func TestSharedLibraryClosure(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("Linux amd64 system library fixture")
	}
	binary, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash required")
	}
	libraries, err := SharedLibraries(binary, []string{"/lib/x86_64-linux-gnu", "/usr/lib/x86_64-linux-gnu", "/lib64"})
	if err != nil {
		t.Fatal(err)
	}
	if len(libraries) == 0 {
		t.Fatal("expected dynamic dependencies")
	}
	if _, err = SharedLibraries(binary, []string{t.TempDir()}); err == nil {
		t.Fatal("missing dependencies ignored")
	}
}
