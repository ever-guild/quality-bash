package qualitybash

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstaller(t *testing.T) {
	for _, dependency := range []string{"bash", "jq", "tar", "sha256sum", "install"} {
		if _, err := exec.LookPath(dependency); err != nil {
			t.Skipf("installer test needs %s", dependency)
		}
	}
	script, err := filepath.Abs("install.sh")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, arch, failStep, bad    string
		paginate, success, installed bool
	}{
		{name: "latest published alpha", success: true, installed: true},
		{name: "arm64", arch: "aarch64", success: true, installed: true},
		{name: "publication across pages", paginate: true, success: true, installed: true},
		{name: "checksum mismatch", bad: "checksum"},
		{name: "missing digest", bad: "digest"},
		{name: "missing archive", bad: "asset"},
		{name: "no published release", bad: "empty"},
		{name: "invalid API JSON", bad: "json"},
		{name: "download failure", bad: "download"},
		{name: "tools failure stops doctor", failStep: "tools install", installed: true},
		{name: "doctor failure", failStep: "doctor", installed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			shim, destination := filepath.Join(root, "shims"), filepath.Join(root, "bin with spaces")
			for _, p := range []string{shim, destination} {
				if err := os.Mkdir(p, 0700); err != nil {
					t.Fatal(err)
				}
			}
			write := func(p, content string) {
				t.Helper()
				if err := os.WriteFile(p, []byte(content), 0700); err != nil {
					t.Fatal(err)
				}
			}
			binary := filepath.Join(destination, "quality-bash")
			write(binary, "previous CLI")
			payload := []byte("#!/bin/bash\nset -eu\nprintf '%s\\n' \"$*\" >> \"$CALL_LOG\"\n[[ \"$*\" != \"${FAIL_STEP:-}\" ]] || exit 23\n")
			archive := filepath.Join(root, "cli.tar.gz")
			file, err := os.Create(archive)
			if err != nil {
				t.Fatal(err)
			}
			gz := gzip.NewWriter(file)
			tw := tar.NewWriter(gz)
			if err := tw.WriteHeader(&tar.Header{Name: "quality-bash", Mode: 0755, Size: int64(len(payload))}); err != nil {
				t.Fatal(err)
			}
			if _, err := tw.Write(payload); err != nil {
				t.Fatal(err)
			}
			for _, closer := range []interface{ Close() error }{tw, gz, file} {
				if err := closer.Close(); err != nil {
					t.Fatal(err)
				}
			}
			data, err := os.ReadFile(archive)
			if err != nil {
				t.Fatal(err)
			}
			digest := fmt.Sprintf("sha256:%x", sha256.Sum256(data))
			if tc.bad == "checksum" {
				digest = "sha256:" + strings.Repeat("0", 64)
			}
			if tc.bad == "digest" {
				digest = ""
			}
			arch := "amd64"
			machine := "x86_64"
			if tc.arch != "" {
				arch = "arm64"
				machine = tc.arch
			}
			tag := "v1.0.0-alpha.99"
			asset := map[string]any{"name": "quality-bash-" + tag + "-linux-" + arch + ".tar.gz", "digest": digest, "state": "uploaded"}
			latest := map[string]any{"id": 2, "tag_name": tag, "draft": false, "prerelease": true, "published_at": "2026-11-08T00:00:00Z", "assets": []any{asset}}
			if tc.bad == "asset" {
				latest["assets"] = []any{}
			}
			older := map[string]any{"id": 3, "tag_name": "v9.0.0", "draft": false, "published_at": "2026-11-01T00:00:00Z"}
			draft := map[string]any{"id": 4, "tag_name": "v10.0.0", "draft": true, "published_at": "2026-12-01T00:00:00Z"}
			page := []any{older, latest, draft}
			if tc.bad == "empty" {
				page = []any{}
			}
			saveJSON := func(name string, v any) {
				t.Helper()
				data, err := json.Marshal(v)
				if err != nil {
					t.Fatal(err)
				}
				write(filepath.Join(root, name), string(data))
			}
			if tc.paginate {
				page = make([]any, 100)
				for i := range page {
					page[i] = older
				}
				saveJSON("2.json", []any{latest})
			}
			saveJSON("1.json", page)
			if tc.bad == "json" {
				write(filepath.Join(root, "1.json"), "not JSON")
			}
			write(filepath.Join(shim, "uname"), "#!/bin/bash\nif [[ $1 == -s ]]; then echo Linux; else echo \"$TEST_MACHINE\"; fi\n")
			write(filepath.Join(shim, "curl"), `#!/bin/bash
set -eu
url= output=
while [[ $# -gt 0 ]]; do
 case "$1" in
  https://*) url=$1 ;;
  -o) shift; output=$1 ;;
 esac
 shift
done
case "$url" in
 https://api.github.com/repos/ever-guild/quality-bash/releases\?per_page=100\&page=*)
  cp -- "$FIXTURE_ROOT/${url##*page=}.json" "$output" ;;
 https://github.com/ever-guild/quality-bash/releases/download/v1.0.0-alpha.99/quality-bash-v1.0.0-alpha.99-linux-*.tar.gz)
  [[ ${FAIL_DOWNLOAD:-} != yes ]] || exit 22
  cp -- "$FIXTURE_ROOT/cli.tar.gz" "$output" ;;
 *) echo "Unexpected URL: $url" >&2; exit 2 ;;
esac
`)
			t.Setenv("PATH", shim+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("XDG_BIN_HOME", destination)
			t.Setenv("FIXTURE_ROOT", root)
			t.Setenv("TEST_MACHINE", machine)
			t.Setenv("CALL_LOG", filepath.Join(root, "calls"))
			t.Setenv("FAIL_STEP", tc.failStep)
			t.Setenv("FAIL_DOWNLOAD", "")
			if tc.bad == "download" {
				t.Setenv("FAIL_DOWNLOAD", "yes")
			}
			output, err := exec.Command("bash", script).CombinedOutput()
			if (err == nil) != tc.success {
				t.Fatalf("success=%v, err=%v\n%s", tc.success, err, output)
			}
			calls, readErr := os.ReadFile(filepath.Join(root, "calls"))
			if readErr != nil && !os.IsNotExist(readErr) {
				t.Fatal(readErr)
			}
			wantCalls := ""
			if tc.installed {
				wantCalls = "tools install\n"
				if tc.failStep != "tools install" {
					wantCalls += "doctor\n"
				}
			}
			if string(calls) != wantCalls {
				t.Fatalf("calls %q, want %q", calls, wantCalls)
			}
			installed, err := os.ReadFile(binary)
			if err != nil {
				t.Fatal(err)
			}
			if !tc.installed && string(installed) != "previous CLI" {
				t.Fatal("failed download/verification replaced the previous CLI")
			}
			if strings.Contains(string(output), "Installation verified.") != tc.success {
				t.Fatalf("incorrect success message: %s", output)
			}
		})
	}
}
