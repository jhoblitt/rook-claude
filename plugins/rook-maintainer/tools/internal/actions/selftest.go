package actions

import (
	"fmt"
	"strings"
)

// SelfTest exercises every check against fixtures and returns what disagreed;
// empty means the gate behaves. It ships inside the binary because the gate is
// invoked from a skill, where the test suite is not available and a silently
// wrong gate reads as "nothing to fix" — the one failure mode that lets a bad
// write through.
func SelfTest() []string {
	const labels = `[{"name": "bug"}, {"name": "feature"}, {"name": "needs-info"},
	                 {"name": "ceph-object"}, {"name": "core"}, {"name": "docs"}]`
	const snapshot = `[
	  {"number": 1, "type": "issue", "state": "OPEN", "labels": [{"name": "bug"}]},
	  {"number": 2, "type": "pr", "state": "OPEN", "labels": []},
	  {"number": 3, "type": "issue", "state": "CLOSED", "labels": []},
	  {"number": 4, "type": "issue", "state": "OPEN",
	   "labels": [{"name": "bug"}, {"name": "core"}, {"name": "docs"}]}
	]`
	const accepted = `[
	  {"number": 1, "action": "label", "params": {"labels": ["needs-info"]}},
	  {"number": 2, "action": "reviewers", "params": {"reviewers": ["a", "b"]}},
	  {"number": 1, "action": "comment", "params": {"mentions": ["a", "b", "c"]}}
	]`
	rejected := []struct{ action, want string }{
		{`{"number": 1, "action": "label", "params": {"labels": ["invented"]}}`,
			"not in the live list"},
		{`{"number": 2, "action": "label", "params": {"labels": ["bug"]}}`,
			"triage labels issues only"},
		{`{"number": 3, "action": "label", "params": {"labels": ["bug"]}}`,
			"not OPEN"},
		{`{"number": 4, "action": "label",
		   "params": {"labels": ["feature", "ceph-object", "needs-info"]}}`,
			"exceeds the cap of 5"},
		{`{"number": 1, "action": "reviewers", "params": {"reviewers": []}}`,
			"outside 1–5"},
		{`{"number": 1, "action": "comment",
		   "params": {"mentions": ["a", "b", "c", "d"]}}`,
			"exceeds the cap of 3"},
		{`{"number": 9, "action": "label", "params": {"labels": ["bug"]}}`,
			"no live state supplied"},
		{`{"number": 1, "action": "frobnicate", "params": {}}`,
			"unknown action"},
	}

	var fails []string
	live, err := ParseLabels([]byte(labels))
	if err != nil {
		return []string{fmt.Sprintf("fixture labels: %v", err)}
	}
	items, err := ParseItems([]byte(snapshot))
	if err != nil {
		return []string{fmt.Sprintf("fixture items: %v", err)}
	}

	if got, err := validated(accepted, live, items); err != nil {
		fails = append(fails, err.Error())
	} else if len(got) > 0 {
		fails = append(fails, fmt.Sprintf("safe actions were rejected: %v", got))
	}

	for _, tc := range rejected {
		got, err := validated("["+tc.action+"]", live, items)
		if err != nil {
			fails = append(fails, err.Error())
			continue
		}
		if len(got) != 1 || !strings.Contains(got[0], tc.want) {
			fails = append(fails, fmt.Sprintf("%s: want one problem containing %q, got %v",
				compact(tc.action), tc.want, got))
		}
	}

	// The label-map diff is the same gate one level up: the map may not name a
	// label the repo does not have.
	const labelMap = "| Paths touched | Area | Issue label |\n|---|---|---|\n" +
		"| `pkg/**` | `core` | `bug` |\n| `x/**` | `y` | `invented` |\n"
	if mapped, err := ParseLabelMap([]byte(labelMap)); err != nil {
		fails = append(fails, fmt.Sprintf("fixture label map: %v", err))
	} else {
		missing, unmapped := DiffLabels(mapped, live)
		if len(missing) != 1 || missing[0] != "invented" {
			fails = append(fails, fmt.Sprintf("label-map diff missed the absent label: %v", missing))
		}
		if len(unmapped) != len(live)-1 {
			fails = append(fails, fmt.Sprintf("label-map diff miscounted the unmapped labels: %v", unmapped))
		}
	}

	// Without live state the open/PR checks are skipped, not silently passed.
	const noState = `[{"number": 1, "action": "label", "params": {"labels": ["bug"]}}]`
	if got, err := validated(noState, live, nil); err != nil {
		fails = append(fails, err.Error())
	} else if len(got) > 0 {
		fails = append(fails, fmt.Sprintf("no-snapshot run reported %v", got))
	}
	return fails
}

func validated(payload string, live []string, items []Item) ([]string, error) {
	parsed, err := Parse([]byte(payload))
	if err != nil {
		return nil, fmt.Errorf("fixture %s: %v", compact(payload), err)
	}
	if !parsed.IsList {
		return nil, fmt.Errorf("fixture %s: not a list", compact(payload))
	}
	return Validate(parsed, live, items), nil
}

func compact(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
