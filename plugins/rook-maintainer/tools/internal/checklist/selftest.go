package checklist

import (
	"fmt"
	"strings"
)

// fixtureTemplate is a snapshot of rook/rook's PR template, kept here only so
// the gate can check itself without a checkout. It is NEVER the template an
// audit runs against — that always comes from the caller's --template ref, so
// a drifting upstream template makes this fixture stale, not the tool wrong.
var fixtureTemplate = strings.Join([]string{
	"<!-- Thank you for contributing to Rook! -->",
	"",
	"<!-- STEPS TO FOLLOW:",
	"  1. Add a description of the changes (frequently the same as the commit description)",
	"  2. Enter the issue number next to \"Resolves #\" below (if there is no tracking issue resolved, **remove that section**)",
	"  3. Review our Contributing documentation at https://rook.io/docs/rook/latest/Contributing/development-flow/",
	"  4. Follow the steps in the checklist below, starting with the **Commit Message Formatting**.",
	"-->",
	"",
	"<!-- Uncomment this section with the issue number if an issue is being resolved",
	"**Issue resolved by this Pull Request:**",
	"Resolves #",
	"--->",
	"",
	"**Checklist:**",
	"",
	"- [ ] **Commit Message Formatting**: Commit titles and messages follow guidelines in the [developer guide](https://rook.io/docs/rook/latest/Contributing/development-flow/#commit-structure).",
	"- [ ] Reviewed the developer guide on [Submitting a Pull Request](https://rook.io/docs/rook/latest/Contributing/development-flow/#submitting-a-pull-request)",
	"- [ ] Reviewed [AI guidelines](https://rook.io/docs/rook/latest/Contributing/ai-guidelines), if AI assisted with the PR.",
	"- [ ] [Pending release notes](https://github.com/rook/rook/blob/master/PendingReleaseNotes.md) updated with breaking and/or notable changes for the next minor release.",
	"  - Overwriting Ceph's configurations should be marked as breaking changes.",
	"- [ ] Documentation has been updated, if necessary (under the `Documentation` folder).",
	"- [ ] Unit tests have been added, if necessary (`_test.go` files under the `cmd` and `pkg` folders).",
	"- [ ] Integration tests have been added, if necessary (in the `tests/integration` folder).",
	"",
	"",
}, "\n")

// render writes an item back out the way the fixture spells it, so the
// mutations below are built from what the parser read rather than from a
// second copy of the template's lines.
func render(it Item) string {
	out := "- "
	if it.Depth > 0 {
		out = "  - "
	}
	switch it.State {
	case StateChecked:
		out += "[x] "
	case StateUnchecked:
		out += "[ ] "
	}
	return out + it.Text
}

// SelfTest exercises the audit against the fixture and returns what disagreed;
// empty means the gate behaves. It ships inside the binary because the gate is
// invoked from a skill, where the test suite is not available and a silently
// wrong gate reads as "the checklist is fine" — the one verdict a rewritten
// checklist must never produce.
func SelfTest() []string {
	tmpl, err := Template(fixtureTemplate)
	if err != nil {
		return []string{fmt.Sprintf("fixture template: %v", err)}
	}
	nested, last := -1, len(tmpl)-1
	for i, it := range tmpl {
		if it.State == StateNone {
			nested = i
			break
		}
	}
	if nested < 0 || last < 1 {
		return []string{fmt.Sprintf("fixture template parsed to %d item(s) and no sub-bullet", len(tmpl))}
	}

	var fails []string
	audit := func(name, body string, want Verdict) Report {
		rep := Audit(tmpl, body)
		if rep.Verdict != want {
			fails = append(fails, fmt.Sprintf("%s: verdict = %s, want %s", name, rep.Verdict, want))
		}
		return rep
	}
	status := func(name string, rep Report, i int, want Status) {
		if i >= len(rep.Lines) {
			fails = append(fails, fmt.Sprintf("%s: report has %d line(s), want more than %d",
				name, len(rep.Lines), i))
			return
		}
		if rep.Lines[i].Status != want {
			fails = append(fails, fmt.Sprintf("%s: line %d is %s, want %s",
				name, i, rep.Lines[i].Status, want))
		}
	}
	problem := func(name string, rep Report, want string) {
		for _, p := range rep.Problems {
			if strings.Contains(p, want) {
				return
			}
		}
		fails = append(fails, fmt.Sprintf("%s: problems = %q, want one mentioning %q",
			name, rep.Problems, want))
	}

	if rep := audit("verbatim", fixtureTemplate, VerdictConforming); rep.Count(StatusOK) != len(tmpl) {
		fails = append(fails, fmt.Sprintf("verbatim: %d of %d line(s) matched",
			rep.Count(StatusOK), len(tmpl)))
	}
	audit("every box ticked", strings.ReplaceAll(fixtureTemplate, "- [ ]", "- [x]"), VerdictConforming)

	name := "reworded item"
	rep := audit(name, strings.Replace(fixtureTemplate, render(tmpl[last]),
		"- [x] Tests were added where it made sense.", 1), VerdictNonConforming)
	status(name, rep, last, StatusAltered)

	name = "dropped item"
	rep = audit(name, strings.Replace(fixtureTemplate, render(tmpl[last])+"\n", "", 1),
		VerdictNonConforming)
	status(name, rep, last, StatusMissing)

	name = "added item"
	rep = audit(name, strings.Replace(fixtureTemplate, render(tmpl[last]),
		render(tmpl[last])+"\n- [x] Ran the linter.", 1), VerdictNonConforming)
	status(name, rep, last+1, StatusExtra)

	name = "dropped sub-bullet"
	rep = audit(name, strings.Replace(fixtureTemplate, render(tmpl[nested])+"\n", "", 1),
		VerdictNonConforming)
	status(name, rep, nested, StatusMissing)

	audit("no checklist", "Fixes a nil dereference in the mon health checker.\n", VerdictNoChecklist)

	name = "checklist twice"
	rep = audit(name, fixtureTemplate+"\nAnd once more:\n\n"+fixtureTemplate, VerdictNonConforming)
	problem(name, rep, "twice")

	name = "fenced checklist"
	rep = audit(name, "```\n"+fixtureTemplate+"```\n", VerdictNoChecklist)
	problem(name, rep, "code fence")

	return fails
}
