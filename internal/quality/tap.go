package quality

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// ValidateTAP requires a complete nonempty test plan. A bare nonzero process
// exit is not evidence that the tests detected a mutation.
func ValidateTAP(reader io.Reader, exitCode int) (bool, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 4<<20)
	planPattern := regexp.MustCompile(`^1\.\.(\d+)(?:\s+#.*)?$`)
	resultPattern := regexp.MustCompile(`^(not ok|ok)\s+(\d+)(?:\s|$)`)
	planned := -1
	count := 0
	failed := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Bail out!") {
			return false, fmt.Errorf("Bats bailed out")
		}
		if match := planPattern.FindStringSubmatch(line); match != nil {
			if planned != -1 {
				return false, fmt.Errorf("duplicate TAP plan")
			}
			planned, _ = strconv.Atoi(match[1])
			continue
		}
		if match := resultPattern.FindStringSubmatch(line); match != nil {
			number, _ := strconv.Atoi(match[2])
			count++
			if number != count {
				return false, fmt.Errorf("nonsequential TAP test number")
			}
			if match[1] == "not ok" {
				failed = true
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	if planned <= 0 || planned != count {
		return false, fmt.Errorf("incomplete or empty TAP plan: planned=%d completed=%d", planned, count)
	}
	if (exitCode == 0 && failed) || (exitCode != 0 && !failed) {
		return false, fmt.Errorf("Bats exit code disagrees with TAP results")
	}
	return failed, nil
}
