package install

import (
	"os/exec"
	"runtime"
	"testing"
)

func TestSharedLibraryClosure(t *testing.T) {
	triplet := map[string]string{"amd64": "x86_64-linux-gnu", "arm64": "aarch64-linux-gnu"}[runtime.GOARCH]
	if runtime.GOOS != "linux" || triplet == "" {
		t.Skip("Linux amd64/arm64 system library fixture")
	}
	binary, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash required")
	}
	libraries, err := SharedLibraries(binary, []string{"/lib/" + triplet, "/usr/lib/" + triplet, "/lib64", "/lib", "/usr/lib"})
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
