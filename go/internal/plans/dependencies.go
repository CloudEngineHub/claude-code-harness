package plans

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var taskIDRefRE = regexp.MustCompile(`\d+(?:\.\d+)*`)

// DependencyViolation reports a done task whose dependency is not closed.
type DependencyViolation struct {
	TaskID       string
	DependencyID string
	Status       string
	DepStatus    string
}

// CheckDoneDependencies verifies that done tasks only depend on closed tasks.
// Closed dependency states are cc:done/cc:完了 and cc:withdrawn.
func CheckDoneDependencies(tasks []Task) []DependencyViolation {
	byID := make(map[string]Task, len(tasks))
	for _, task := range tasks {
		byID[task.TaskID] = task
	}

	var violations []DependencyViolation
	for _, task := range tasks {
		if !task.Tags.Done {
			continue
		}
		for _, depID := range dependencyIDs(task.Depends) {
			dep, ok := byID[depID]
			if !ok {
				continue
			}
			if isClosedDependency(dep) {
				continue
			}
			violations = append(violations, DependencyViolation{
				TaskID:       task.TaskID,
				DependencyID: depID,
				Status:       task.Status,
				DepStatus:    dep.Status,
			})
		}
	}

	sort.SliceStable(violations, func(i, j int) bool {
		if violations[i].TaskID == violations[j].TaskID {
			return violations[i].DependencyID < violations[j].DependencyID
		}
		return violations[i].TaskID < violations[j].TaskID
	})
	return violations
}

func FormatDependencyViolations(violations []DependencyViolation) string {
	var lines []string
	for _, v := range violations {
		lines = append(lines, fmt.Sprintf("task %s is done but depends on %s (%s)", v.TaskID, v.DependencyID, v.DepStatus))
	}
	return strings.Join(lines, "\n")
}

func dependencyIDs(depends string) []string {
	depends = strings.TrimSpace(depends)
	if depends == "" || depends == "-" || strings.EqualFold(depends, "none") {
		return nil
	}

	seen := map[string]bool{}
	var ids []string
	for _, id := range taskIDRefRE.FindAllString(depends, -1) {
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

func isClosedDependency(task Task) bool {
	if task.Tags.Done {
		return true
	}
	status := strings.ToLower(task.Status)
	return strings.Contains(status, "cc:withdrawn")
}
