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
	return validateTAP(reader, exitCode, false)
}

// validateTAP permits only the pinned Bats abort trailer after a failed test.
func validateTAP(reader io.Reader, exitCode int, abort bool) (bool, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 4<<20)
	planPattern := regexp.MustCompile(`^1\.\.(\d+)(?:\s+#.*)?$`)
	resultPattern := regexp.MustCompile(`^(not ok|ok)\s+(\d+)(?:\s|$)`)
	planned := -1
	count := 0
	failed := false
	lastFailed := false
	stopped := false
	for scanner.Scan() {
		line := scanner.Text()
		if stopped && strings.TrimSpace(line) != "" {
			return false, fmt.Errorf("output after abort trailer")
		}
		if strings.HasPrefix(line, "# bats warning: Executed ") {
			expected := fmt.Sprintf("# bats warning: Executed %d instead of expected %d tests", count, planned)
			if !abort || exitCode != 1 || !lastFailed || count >= planned || line != expected {
				return false, fmt.Errorf("invalid abort trailer")
			}
			stopped = true
			continue
		}
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
			if planned < 1 {
				return false, fmt.Errorf("missing TAP plan")
			}
			number, _ := strconv.Atoi(match[2])
			count++
			if number != count {
				return false, fmt.Errorf("nonsequential TAP test number")
			}
			lastFailed = match[1] == "not ok"
			if strings.Contains(line, " # timeout ") {
				return false, fmt.Errorf("Bats test timed out")
			}
			if lastFailed {
				failed = true
			}
			continue
		}
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "#") {
			return false, fmt.Errorf("malformed TAP line")
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	if planned <= 0 || (planned != count && !stopped) {
		return false, fmt.Errorf("incomplete or empty TAP plan: planned=%d completed=%d", planned, count)
	}
	if exitCode != 0 && exitCode != 1 {
		return false, fmt.Errorf("unexpected Bats exit code")
	}
	if (exitCode == 0 && failed) || (exitCode != 0 && !failed) {
		return false, fmt.Errorf("Bats exit code disagrees with TAP results")
	}
	return failed, nil
}
