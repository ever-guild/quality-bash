package quality

import (
	"fmt"
	"runtime"
	"strconv"
)

// AutoJobs selects the Go runtime's current container-aware parallelism.
const AutoJobs = 0

func ParseJobs(value string) (int, error) {
	if value == "auto" {
		return AutoJobs, nil
	}
	jobs, err := strconv.Atoi(value)
	if err != nil || jobs < 1 {
		return 0, fmt.Errorf("jobs must be auto or a positive integer")
	}
	return jobs, nil
}

func mutationWorkers(jobs, mutants int) int {
	if jobs == AutoJobs {
		jobs = runtime.GOMAXPROCS(0)
	}
	return min(jobs, mutants)
}
