// Package actions implements rook-triage phase 5's pre-write checks on
// proposed triage actions: label-set membership, the caps, the approver floor
// on an item's reviewer set, the issues-only label rule and the still-open
// recheck. It also owns the bounds themselves, and the run-wide budget
// phases 0 and 4 read (ApproverBudget).
//
// These are set and count operations, so they are decided here rather than by
// an agent re-deriving them per item. A wrong write lands on someone else's
// issue and cannot be taken back.
//
// Nothing here reaches GitHub. The caller supplies the live label list, the
// item snapshot and the kb's approver roster (`gh label list --json name`,
// `gh issue list --json number,state,labels`, ~/.cache/rook-triage/kb.json),
// so the gate judges what the maintainer can see and a fetch failure can never
// make it fail open. Parsing is exported for the same
// reason: any producer of that JSON — a file, a pipe, a test fixture — feeds
// the same checks.
//
// What it does NOT decide: whether a human answered the item since assessment.
// That needs judgment and stays with the orchestrator.
package actions

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/links"
	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/mdreport"
	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/rtanalyze"
)

// These thresholds are mirrors; the reference each cites owns the number and
// the reason for it. A constant cannot be a pointer and prose cannot enforce —
// so when the two diverge it is this file that decides what actually posts,
// silently. Change one, change the other.
const (
	MaxLabels    = 5 // references/label-map.md, "Rules"
	MaxMentions  = 3 // references/routing.md, Selection step 4
	MinReviewers = 3 // references/routing.md, Selection step 4
	MaxReviewers = 5 // references/routing.md, Selection step 4
	MinApprovers = 2 // references/routing.md, Selection step 4
)

// ApproverBudget is how many PRs a run can route before the approver floor
// exhausts the per-person cap: each PR spends MinApprovers approver slots and
// each approver has mdreport.PerPersonCap of them. Phase 0 sizes a sweep's
// scope against it and phase 4 reconciles against it, so both read the number
// from here rather than recomputing it.
//
// A roster too small to fill the floor budgets no PRs at all, rather than the
// share of one the division would otherwise yield.
func ApproverBudget(approvers int) int {
	if approvers < MinApprovers {
		return 0
	}
	return mdreport.PerPersonCap * approvers / MinApprovers
}

// LoadKBApprovers reads the kb at path for ParseKBApprovers. An empty path
// returns nil, which is how a run with no kb skips the checks that need one; a
// kb that lists no approvers is an error instead, since an empty set answers
// "is this login an approver?" with no for everyone and would fail every
// reviewer set in the run.
func LoadKBApprovers(path string) (map[string]bool, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("--kb: %w", err)
	}
	approvers, err := ParseKBApprovers(data)
	if err != nil {
		return nil, fmt.Errorf("--kb: %w", err)
	}
	if len(approvers) == 0 {
		return nil, fmt.Errorf("--kb: %s lists no roster.approvers", path)
	}
	return approvers, nil
}

// ParseKBApprovers reads roster.approvers out of a routing kb.json as the
// lowercased membership set the tier checks ask their question against.
func ParseKBApprovers(data []byte) (map[string]bool, error) {
	var kb struct {
		Roster rtanalyze.Roster `json:"roster"`
	}
	if err := json.Unmarshal(data, &kb); err != nil {
		return nil, err
	}
	return kb.Roster.ApproverSet(), nil
}

var knownActions = map[string]bool{
	"label": true, "comment": true, "close": true, "convert": true, "reviewers": true,
}

// Name is one label, reviewer or mention. gh emits these as {"name": x} and
// hand-written drafts use bare strings; both have to validate.
type Name string

func (n *Name) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*n = Name(s)
		return nil
	}
	var obj struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return fmt.Errorf("expected a string or an object with a name, got %s", b)
	}
	*n = Name(obj.Name)
	return nil
}

type Params struct {
	Labels    []Name `json:"labels"`
	Reviewers []Name `json:"reviewers"`
	Mentions  []Name `json:"mentions"`
}

type Action struct {
	Number *json.Number `json:"number"`
	Type   string       `json:"type"`
	Action string       `json:"action"`
	Params Params       `json:"params"`
}

type Item struct {
	Number *json.Number `json:"number"`
	Type   string       `json:"type"`
	State  *string      `json:"state"`
	Labels []Name       `json:"labels"`
}

// Payload is a parsed proposed-actions file. A payload that is not a JSON list,
// or an element that is not an object, is a validation problem rather than a
// read error: those come from the drafting agent, and the report is where they
// get fixed.
type Payload struct {
	IsList  bool
	Entries []*Action // a nil entry is an element that was not an object
}

// Parse decodes a proposed-actions payload.
func Parse(data []byte) (Payload, error) {
	var probe any
	if err := json.Unmarshal(data, &probe); err != nil {
		return Payload{}, err
	}
	if _, ok := probe.([]any); !ok {
		return Payload{}, nil
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return Payload{}, err
	}
	out := Payload{IsList: true, Entries: make([]*Action, 0, len(raw))}
	for i, r := range raw {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(r, &obj); err != nil || obj == nil {
			out.Entries = append(out.Entries, nil)
			continue
		}
		var a Action
		if err := json.Unmarshal(r, &a); err != nil {
			return Payload{}, fmt.Errorf("actions[%d]: %w", i, err)
		}
		out.Entries = append(out.Entries, &a)
	}
	return out, nil
}

// ParseLabels decodes `gh label list --json name` output into label names.
func ParseLabels(data []byte) ([]string, error) {
	var raw []Name
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return names(raw), nil
}

// ParseItems decodes the live per-item snapshot. A nil result means no snapshot
// was supplied, which skips the open/PR checks rather than passing them.
func ParseItems(data []byte) ([]Item, error) {
	var items []Item
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// Validate returns human-readable problems; empty means every action is safe to
// execute. A nil items snapshot skips the open/PR checks rather than passing
// them, and a nil approver set skips the approver floor the same way.
func Validate(payload Payload, live []string, items []Item, approvers map[string]bool) []string {
	if !payload.IsList {
		return []string{"actions payload: expected a list"}
	}

	liveSet := make(map[string]bool, len(live))
	for _, l := range live {
		liveSet[l] = true
	}
	requested := reviewersByItem(payload.Entries)
	checked := make(map[string]bool, len(requested))
	byNumber := make(map[string]*Item, len(items))
	for i := range items {
		if items[i].Number != nil {
			byNumber[numKey(*items[i].Number)] = &items[i]
		}
	}

	var problems []string
	for idx, a := range payload.Entries {
		tag := fmt.Sprintf("actions[%d]", idx)
		if a == nil {
			problems = append(problems, tag+": not an object")
			continue
		}
		if a.Number == nil {
			problems = append(problems, tag+": missing `number`")
			continue
		}

		where := fmt.Sprintf("%s #%s", tag, a.Number.String())
		kind := strings.ToLower(a.Type)
		action := strings.ToLower(a.Action)
		if !knownActions[action] {
			problems = append(problems,
				fmt.Sprintf("%s: unknown action %s", where, reprString(action)))
			continue
		}

		item := byNumber[numKey(*a.Number)]
		if items != nil {
			if item == nil {
				problems = append(problems, where+": no live state supplied for this item")
				continue
			}
			if upper(item.State) != "OPEN" {
				problems = append(problems, fmt.Sprintf(
					"%s: item is %s, not OPEN — re-assess before writing", where, repr(item.State)))
				continue
			}
			if kind == "" {
				kind = strings.ToLower(item.Type)
			}
		}

		switch action {
		case "label":
			proposed := names(a.Params.Labels)
			if kind == "pr" {
				problems = append(problems,
					where+": label action on a PR — triage labels issues only")
				continue
			}
			if len(proposed) == 0 {
				problems = append(problems, where+": label action with no labels")
				continue
			}
			var invented []string
			for _, l := range proposed {
				if !liveSet[l] {
					invented = append(invented, l)
				}
			}
			if len(invented) > 0 {
				slices.Sort(invented)
				problems = append(problems, fmt.Sprintf("%s: label(s) not in the live list: %s",
					where, strings.Join(clean(invented), ", ")))
			}
			var current []string
			if item != nil {
				current = names(item.Labels)
			}
			total := union(current, proposed)
			if len(total) > MaxLabels {
				problems = append(problems, fmt.Sprintf(
					"%s: %d labels after apply exceeds the cap of %d (%s)",
					where, len(total), MaxLabels, strings.Join(clean(total), ", ")))
			}

		case "reviewers":
			key := numKey(*a.Number)
			if checked[key] {
				continue
			}
			checked[key] = true
			set := requested[key]
			n := len(set)
			if n < MinReviewers || n > MaxReviewers {
				problems = append(problems, fmt.Sprintf("%s: %d reviewers is outside %d–%d",
					where, n, MinReviewers, MaxReviewers))
			} else if approvers != nil {
				// A set of the wrong size is redrawn whole, and the tiers of the
				// names in it are then a different set's question.
				held := 0
				for _, login := range set {
					if approvers[login] {
						held++
					}
				}
				if held < MinApprovers {
					problems = append(problems, fmt.Sprintf(
						"%s: %d of %d reviewers hold an approver tier, want %d",
						where, held, n, MinApprovers))
				}
			}

		case "comment":
			if n := len(names(a.Params.Mentions)); n > MaxMentions {
				problems = append(problems, fmt.Sprintf("%s: %d mentions exceeds the cap of %d",
					where, n, MaxMentions))
			}
		}
	}
	return problems
}

// reviewersByItem unions the names of every reviewers action per item: two
// actions on one PR are one request to GitHub, so the bounds are the union's,
// as the label cap counts the item rather than the action. Logins are folded to
// lower case, which is how GitHub compares them.
func reviewersByItem(entries []*Action) map[string][]string {
	out := map[string][]string{}
	seen := map[string]map[string]bool{}
	for _, a := range entries {
		if a == nil || a.Number == nil || strings.ToLower(a.Action) != "reviewers" {
			continue
		}
		key := numKey(*a.Number)
		if seen[key] == nil {
			seen[key] = map[string]bool{}
		}
		for _, name := range names(a.Params.Reviewers) {
			login := strings.ToLower(name)
			if seen[key][login] {
				continue
			}
			seen[key][login] = true
			out[key] = append(out[key], login)
		}
	}
	return out
}

// clean bounds the label names a problem echoes. A proposed label is whatever
// the model wrote and a live one whatever its creator named it; unbounded and
// unstripped, either can forge a line of the report the caller fences.
func clean(in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = links.Sanitize(s)
	}
	return out
}

func names(in []Name) []string {
	out := make([]string, 0, len(in))
	for _, n := range in {
		if n != "" {
			out = append(out, string(n))
		}
	}
	return out
}

func union(a, b []string) []string {
	set := make(map[string]struct{}, len(a)+len(b))
	for _, s := range a {
		set[s] = struct{}{}
	}
	for _, s := range b {
		set[s] = struct{}{}
	}
	return slices.Sorted(maps.Keys(set))
}

func upper(s *string) string {
	if s == nil {
		return ""
	}
	return strings.ToUpper(*s)
}

// numKey collapses 1 and 1.0 onto the same item; an action whose number does
// not match its snapshot entry is rejected as unknown, and a formatting
// difference is not a reason to reject.
func numKey(n json.Number) string {
	if i, err := n.Int64(); err == nil {
		return strconv.FormatInt(i, 10)
	}
	if f, err := n.Float64(); err == nil {
		return strconv.FormatFloat(f, 'g', -1, 64)
	}
	return n.String()
}

// repr renders a value as Python's %r did. The messages are the report's
// wording and are grepped for, so they stay byte-identical to the original.
func repr(s *string) string {
	if s == nil {
		return "None"
	}
	return reprString(*s)
}

func reprString(s string) string {
	quote := '\''
	if strings.ContainsRune(s, '\'') && !strings.ContainsRune(s, '"') {
		quote = '"'
	}
	var b strings.Builder
	b.WriteRune(quote)
	for _, r := range s {
		switch {
		case r == quote || r == '\\':
			b.WriteRune('\\')
			b.WriteRune(r)
		case r == '\n':
			b.WriteString("\\n")
		case r == '\r':
			b.WriteString("\\r")
		case r == '\t':
			b.WriteString("\\t")
		case !unicode.IsPrint(r):
			b.WriteString(escapeRune(r))
		default:
			b.WriteRune(r)
		}
	}
	b.WriteRune(quote)
	return b.String()
}

func escapeRune(r rune) string {
	switch {
	case r < 0x100:
		return fmt.Sprintf("\\x%02x", r)
	case r < 0x10000:
		return fmt.Sprintf("\\u%04x", r)
	}
	return fmt.Sprintf("\\U%08x", r)
}
