package quality

import (
	"reflect"
	"strings"
	"testing"
)

func TestASTMutations(t *testing.T) {
	source := `# exit 0 and a + b
url="https://example.org/a+b"
awk 'BEGIN { exit 0 }'
if [[ "$x" -eq 2 ]]; then
 (( x = x + 1 ))
 exit 0
fi
`
	mutations, err := GenerateMutations("sample.sh", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(mutations) != 3 {
		t.Fatalf("unexpected mutations: %+v", mutations)
	}
	again, _ := GenerateMutations("sample.sh", source)
	if !reflect.DeepEqual(mutations, again) {
		t.Fatal("unstable mutations")
	}
	for _, m := range mutations {
		changed, err := m.Apply(source)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(changed, "awk 'BEGIN { exit 0 }'") {
			t.Fatal("embedded language mutated")
		}
	}
	if _, err := GenerateMutations("bad.sh", "if then"); err == nil {
		t.Fatal("invalid Bash accepted")
	}
}
