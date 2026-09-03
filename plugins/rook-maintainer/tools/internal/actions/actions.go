// Package actions implements rook-triage phase 5's pre-write checks on
// proposed triage actions: label-set membership, the caps, the issues-only
// label rule and the still-open recheck.
//
// These are set and count operations, so they are decided here rather than by
// an agent re-deriving them per item. A wrong write lands on someone else's
// issue and cannot be taken back.
//
// Nothing here reaches GitHub. The caller supplies the live label list and the
// item snapshot (`gh label list --json name`, `gh issue list --json
// number,state,labels`), so the gate judges what the maintainer can see and a
// fetch failure can never make it fail open. Parsing is exported for the same
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
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/links"
)

// These thresholds belong to the skill's prose; this file only enforces them.
// Each cites the reference that owns it, because a constant cannot be a pointer
// and prose cannot enforce — so when the two diverge it is this file that
// decides what actually posts, silently. Change one, change the other.
const (
	MaxLabels    = 5 // references/label-map.md, "Rules"
	MaxMentions  = 3 // references/routing.md, Selection bounds
	MinReviewers = 1 // references/routing.md, Selection bounds
	MaxReviewers = 5 // references/routing.md, Selection bounds
)

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
// them.
func Validate(payload Payload, live []string, items []Item) []string {
	if !payload.IsList {
		return []string{"actions payload: expected a list"}
	}

	liveSet := make(map[string]bool, len(live))
	for _, l := range live {
		liveSet[l] = true
	}
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
			n := len(names(a.Params.Reviewers))
			if n < MinReviewers || n > MaxReviewers {
				problems = append(problems, fmt.Sprintf("%s: %d reviewers is outside %d–%d",
					where, n, MinReviewers, MaxReviewers))
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
