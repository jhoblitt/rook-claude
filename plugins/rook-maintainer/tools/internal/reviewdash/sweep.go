// Package reviewdash renders the rook-code-review sweep dashboard from the
// canonical sweep-dir inputs (contract: sweep.md phase 3).
//
// It reads snapshot.json (live metadata from the prefetch: titles, authors,
// CI rollup), pr-*/findings.json (verified findings with assigned IDs, the
// recomputed verdict, the backport assessment) and sweep.json (skip rows).
// Verdicts and finding counts always come from findings.json and never from
// agent prose; CI cells are always passing/total from the snapshot.
//
// pr-<N>/findings.json is either the record object below or a bare findings
// array; fields beyond `findings` are optional and render as an em dash when
// absent:
//
//	{"pr": 18123, "verdict": "ACCEPT|REQUEST_CHANGES|REJECT",
//	 "bug": "REAL|FABRICATED|N/A", "rationale": "one paragraph",
//	 "backport": {"eligible": true, "label": "...", "reason": "..."},
//	 "needs_proposal_review": {"flag": false, "paths": []},
//	 "takeover_candidate": {"flag": false, "reason": ""},
//	 "findings": [{"id": "B1", "severity": "blocker", "domain": "bug",
//	               "path": "pkg/...", "line": 42, "summary": "...",
//	               "confidence": 85, "status": "pending|approved|posted|dropped"}],
//	 "clean": ["areas audited and found correct"]}
package reviewdash

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const (
	defaultRepo   = "rook/rook"
	dashboardFile = "dashboard.html"
)

// value holds one JSON value as the dashboard needs it: the text to render,
// whether it counts as set (an empty string, 0 and false do not), and its
// integer value when it has one. findings.json is agent-written, so a field
// that arrives as a number where a string was expected has to render rather
// than fail the whole dashboard.
type value struct {
	str    string
	truthy bool
	i      int64
	isInt  bool
}

func (v *value) UnmarshalJSON(b []byte) error {
	s := string(bytes.TrimSpace(b))
	if s == "" || s == "null" {
		*v = value{}
		return nil
	}
	switch s[0] {
	case '"':
		var str string
		if err := json.Unmarshal(b, &str); err != nil {
			return err
		}
		*v = value{str: str, truthy: str != ""}
	case '[', '{':
		var compact bytes.Buffer
		if err := json.Compact(&compact, b); err != nil {
			return err
		}
		c := compact.String()
		*v = value{str: c, truthy: c != "[]" && c != "{}"}
	case 't':
		*v = value{str: "True", truthy: true}
	case 'f':
		*v = value{str: "False"}
	default:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("unexpected JSON value %q", s)
		}
		i, ierr := strconv.ParseInt(s, 10, 64)
		*v = value{str: s, truthy: f != 0, i: i, isInt: ierr == nil}
	}
	return nil
}

func (v *value) text() string {
	if v == nil {
		return ""
	}
	return v.str
}

// textOr substitutes fallback for an absent field only.
func (v *value) textOr(fallback string) string {
	if v == nil {
		return fallback
	}
	return v.str
}

// orElse substitutes fallback for a field that is absent or empty.
func (v *value) orElse(fallback string) string {
	if v == nil || v.str == "" {
		return fallback
	}
	return v.str
}

// number is the whole part of a numeric value, 0 for anything else.
func (v *value) number() int {
	if v == nil {
		return 0
	}
	if v.isInt {
		return int(v.i)
	}
	f, err := strconv.ParseFloat(v.str, 64)
	if err != nil {
		return 0
	}
	return int(f)
}

type finding struct {
	ID         *value `json:"id"`
	Severity   *value `json:"severity"`
	Domain     value  `json:"domain"`
	Path       value  `json:"path"`
	Line       value  `json:"line"`
	Summary    value  `json:"summary"`
	Confidence value  `json:"confidence"`
	Status     *value `json:"status"`
}

func (f finding) live() bool {
	return f.Status.text() != "dropped"
}

type backport struct {
	Eligible value `json:"eligible"`
	Label    value `json:"label"`
	Reason   value `json:"reason"`
}

type proposalReview struct {
	Flag  value   `json:"flag"`
	Paths []value `json:"paths"`
}

type takeover struct {
	Flag   value `json:"flag"`
	Reason value `json:"reason"`
}

type record struct {
	PR                  *value          `json:"pr"`
	Verdict             *value          `json:"verdict"`
	Bug                 value           `json:"bug"`
	Rationale           value           `json:"rationale"`
	Backport            *backport       `json:"backport"`
	NeedsProposalReview *proposalReview `json:"needs_proposal_review"`
	TakeoverCandidate   *takeover       `json:"takeover_candidate"`
	Findings            []finding       `json:"findings"`
	Clean               []value         `json:"clean"`

	num   string
	order float64
}

func (r record) liveFindings() []finding {
	out := make([]finding, 0, len(r.Findings))
	for _, f := range r.Findings {
		if f.live() {
			out = append(out, f)
		}
	}
	return out
}

type ciRollup struct {
	Total   value   `json:"total"`
	Passing value   `json:"passing"`
	Failing value   `json:"failing"`
	Pending value   `json:"pending"`
	Failed  []value `json:"failed"`
}

type item struct {
	Title             value     `json:"title"`
	Author            value     `json:"author"`
	AuthorAssociation value     `json:"authorAssociation"`
	CI                *ciRollup `json:"ci"`
}

type skip struct {
	Number value `json:"number"`
	Reason value `json:"reason"`
}

type snapshotFile struct {
	Repo  *value          `json:"repo"`
	Items map[string]item `json:"items"`
}

type sweepFile struct {
	Skipped []skip `json:"skipped"`
}

// Sweep is one sweep directory, loaded and ordered, ready to render.
type Sweep struct {
	Dir   string
	Date  string
	Repo  string
	recs  []record
	items map[string]item
	skips []skip
}

// Load reads the three canonical inputs. A missing input is empty, not an
// error — phase 3 regenerates the dashboard repeatedly as a sweep progresses —
// but an unreadable or malformed one is: a dashboard that silently drops half
// the sweep reads as "nothing found".
func Load(dir string) (*Sweep, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	var snap snapshotFile
	if err := readJSON(filepath.Join(abs, "snapshot.json"), &snap); err != nil {
		return nil, err
	}
	var sweep sweepFile
	if err := readJSON(filepath.Join(abs, "sweep.json"), &sweep); err != nil {
		return nil, err
	}

	s := &Sweep{
		Dir:   abs,
		Date:  firstRunes(filepath.Base(abs), 10),
		Repo:  snap.Repo.textOr(defaultRepo),
		items: snap.Items,
		skips: sweep.Skipped,
	}

	paths, err := filepath.Glob(filepath.Join(abs, "pr-*", "findings.json"))
	if err != nil {
		return nil, err
	}
	slices.Sort(paths)
	for _, p := range paths {
		rec, err := loadRecord(p)
		if err != nil {
			return nil, err
		}
		s.recs = append(s.recs, rec)
	}
	slices.SortStableFunc(s.recs, func(a, b record) int {
		return cmp.Compare(a.order, b.order)
	})
	return s, nil
}

// Path is where Render's output belongs.
func (s *Sweep) Path() string {
	return filepath.Join(s.Dir, dashboardFile)
}

// Summary is the one-line phase-3 receipt: verdict tally and live finding
// count, both recomputed from the same records the page was rendered from.
func (s *Sweep) Summary() string {
	counts := map[string]int{}
	live := 0
	for _, r := range s.recs {
		counts[r.Verdict.textOr(emDash)]++
		live += len(r.liveFindings())
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	tally := make([]string, 0, len(keys))
	for _, k := range keys {
		tally = append(tally, fmt.Sprintf("%d %s", counts[k], k))
	}
	return fmt.Sprintf("%s: %d PRs %s %s; %d live findings -> %s",
		filepath.Base(s.Dir), len(s.recs), emDash, strings.Join(tally, ", "), live, s.Path())
}

func readJSON(path string, out any) error {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func loadRecord(path string) (record, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return record{}, err
	}
	var rec record
	if bytes.HasPrefix(bytes.TrimLeft(b, " \t\r\n"), []byte("[")) {
		err = json.Unmarshal(b, &rec.Findings)
	} else {
		err = json.Unmarshal(b, &rec)
	}
	if err != nil {
		return record{}, fmt.Errorf("%s: %w", path, err)
	}
	if rec.PR != nil {
		rec.num = rec.PR.str
		rec.order = float64(rec.PR.number())
		return rec, nil
	}
	name := strings.TrimPrefix(filepath.Base(filepath.Dir(path)), "pr-")
	n, err := strconv.Atoi(name)
	if err != nil {
		return record{}, fmt.Errorf("%s: no \"pr\" field and %q is not a PR directory", path, name)
	}
	rec.num, rec.order = strconv.Itoa(n), float64(n)
	return rec, nil
}

func firstRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}
