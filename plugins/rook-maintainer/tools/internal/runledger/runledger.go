// Package runledger builds the RUN-scoped routing ledger of a rook-triage run:
// per person, how many ITEMS the whole run proposes routing to them, against
// mdreport.PerPersonCap.
//
// The cap of skills/rook-triage/references/routing.md is per person per RUN —
// 3 items across every corpus the run touches — and `both`, the default mode,
// allocates one sweep dir per corpus (SKILL.md "State"). Each corpus's own
// ledger therefore counts only half of what the cap bounds: a person proposed
// on two PRs and two issues sits under the cap in both dirs and breaches the
// run's. Nothing downstream re-checks the cap — validate-actions deliberately
// does not — so this fragment is the only place that breach becomes visible.
// Caller: rook-triage phase 4's cross-dir reconciliation, via gen-run-ledger.
//
// Counting is mdreport's, not this package's: Counts charges a login once per
// item, and the same login on an issue and a PR is two charges. What is added
// here is the sum ACROSS dirs and the provenance beside it — which items in
// which dir a person's charges came from — because a total without that names
// a problem no maintainer can act on.
package runledger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/issuesdash"
	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/mdreport"
	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/prdash"
)

// LedgerFile is written into EVERY dir of the run, with identical content: the
// cap is run-scoped, so both corpora's report.md must carry the same combined
// view. A maintainer reading only the issues report has to see a breach the PR
// dir contributed to, and a per-dir copy is also what keeps each sweep dir a
// self-contained artifact.
const LedgerFile = "run-ledger.md"

// Corpus is which of a run's two item pools a sweep dir holds.
type Corpus string

const (
	PRs    Corpus = "prs"
	Issues Corpus = "issues"
)

// order is the canonical corpus order of the fragment. Rendering follows it
// rather than the argument order so the same run yields the same bytes however
// the dirs were passed.
var order = []Corpus{PRs, Issues}

// column heads the corpus's provenance column, in the vocabulary of the
// per-corpus ledgers ("reviewer" for PRs, "mentioned" for issues).
func (c Corpus) column() string {
	if c == PRs {
		return "PRs (reviewer)"
	}
	return "issues (mentioned)"
}

func (c Corpus) link() func(string) string {
	if c == PRs {
		return mdreport.PullLink
	}
	return mdreport.IssueLink
}

// Sweep is one sweep dir's contribution: every item it assessed, with the
// logins it proposes on that item.
type Sweep struct {
	Dir       string
	Corpus    Corpus
	Proposals []mdreport.Proposal
}

// Run is a triage run's sweep dirs, at most one per corpus.
type Run struct {
	Sweeps []Sweep
}

// Row is one person's line of the ledger: the run-wide count charged to them,
// and the items each corpus proposed them on so a maintainer can see what to
// drop.
type Row struct {
	Login string
	Count int
	Items map[Corpus][]string
}

// Load reads a run's sweep dirs — one for an issues-only or prs-only run, two
// for the default `both` — and extracts what each proposes.
//
// Every failure here is loud, because the alternative is a total that reads
// fine and is wrong: a dir with no batch file has produced no assessments yet
// and would sum to a clean ledger, and two dirs of the SAME corpus would
// double one corpus's proposals into a breach nobody committed (or, for a dir
// passed twice, into one that is pure arithmetic).
func Load(dirs ...string) (*Run, error) {
	if len(dirs) == 0 {
		return nil, fmt.Errorf("no sweep dir given: the ledger spans a run's %d dir(s)", len(order))
	}
	if len(dirs) > len(order) {
		return nil, fmt.Errorf("%d sweep dirs given: a run has at most one per corpus (%s)",
			len(dirs), strings.Join(names(order), ", "))
	}

	run := &Run{}
	seen := map[Corpus]string{}
	for _, dir := range dirs {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return nil, err
		}
		for _, s := range run.Sweeps {
			if s.Dir == abs {
				return nil, fmt.Errorf("%s given twice: summing a dir against itself "+
					"doubles every proposal in it", abs)
			}
		}
		s, err := loadSweep(abs)
		if err != nil {
			return nil, err
		}
		if other, dup := seen[s.Corpus]; dup {
			return nil, fmt.Errorf("%s and %s are both %s sweeps (snapshot.json \"kind\"): "+
				"a run has one dir per corpus, and two of one corpus are two runs, not one",
				other, abs, s.Corpus)
		}
		seen[s.Corpus] = abs
		run.Sweeps = append(run.Sweeps, *s)
	}
	slices.SortStableFunc(run.Sweeps, func(a, b Sweep) int {
		return slices.Index(order, a.Corpus) - slices.Index(order, b.Corpus)
	})
	return run, nil
}

func names(cs []Corpus) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, string(c))
	}
	return out
}

// loadSweep reads one dir through its corpus's own loader, so the run ledger
// counts exactly what that corpus's report counts.
func loadSweep(dir string) (*Sweep, error) {
	batches, err := filepath.Glob(filepath.Join(dir, "batch-*.json"))
	if err != nil {
		return nil, err
	}
	if len(batches) == 0 {
		return nil, fmt.Errorf("%s: no batch-*.json to report on", dir)
	}
	corpus, err := corpusOf(dir)
	if err != nil {
		return nil, err
	}
	if corpus == PRs {
		s, err := prdash.Load(dir)
		if err != nil {
			return nil, err
		}
		return &Sweep{Dir: dir, Corpus: PRs, Proposals: s.Page().ProposedReviewers()}, nil
	}
	s, err := issuesdash.Load(dir)
	if err != nil {
		return nil, err
	}
	return &Sweep{Dir: dir, Corpus: Issues, Proposals: s.Page().ProposedMentions()}, nil
}

// corpusOf reads the corpus off snapshot.json's top-level "kind".
//
// That field is written by sweep-prefetch rather than by a model, and the
// snapshot is a mandatory input of both dashboard loaders, so it is the one
// corpus signal a sweep dir cannot be missing. The dir NAME carries the same
// kind by convention (SKILL.md "State"), but a name is renameable and copied,
// and mistaking one corpus for the other here produces a ledger that reads
// plausibly while counting the wrong proposals.
func corpusOf(dir string) (Corpus, error) {
	path := filepath.Join(dir, "snapshot.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var snap struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(b, &snap); err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}
	switch c := Corpus(snap.Kind); c {
	case PRs, Issues:
		return c, nil
	}
	return "", fmt.Errorf("%s: \"kind\" is %q, want one of %s", path, snap.Kind,
		strings.Join(names(order), "/"))
}

// Rows tallies the run. The counts are mdreport.Counts over every dir's
// proposals, grouped per item; the provenance is gathered beside them under
// the same case-folded key.
//
// Those two derivations are independent, so they are checked against each
// other: a row whose count and whose item list disagree is a bug in this
// package, and reporting either number would be reporting a cap total nobody
// can verify.
func (r *Run) Rows() ([]Row, error) {
	var proposals []mdreport.Proposal
	items := map[string]map[Corpus][]string{}
	for _, s := range r.Sweeps {
		for _, p := range s.Proposals {
			charged := map[string]bool{}
			for _, login := range p.Logins {
				key := mdreport.Fold(login)
				if charged[key] {
					continue
				}
				charged[key] = true
				if items[key] == nil {
					items[key] = map[Corpus][]string{}
				}
				items[key][s.Corpus] = append(items[key][s.Corpus], p.Number)
			}
		}
		proposals = append(proposals, s.Proposals...)
	}

	entries := mdreport.Counts(mdreport.Group(proposals))
	rows := make([]Row, 0, len(entries))
	for _, e := range entries {
		row := Row{Login: e.Login, Count: e.Count, Items: items[mdreport.Fold(e.Login)]}
		listed := 0
		for _, nums := range row.Items {
			listed += len(nums)
		}
		if listed != e.Count {
			return nil, fmt.Errorf("%s: counted on %d item(s) but listed on %d",
				e.Login, e.Count, listed)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// over lists the rows that breach the cap, heaviest first (Rows' order).
func over(rows []Row) []Row {
	var out []Row
	for _, row := range rows {
		if row.Count > mdreport.PerPersonCap {
			out = append(out, row)
		}
	}
	return out
}

// Render writes the fragment: one table over the whole run, in the shape and
// status vocabulary of the per-corpus ledgers, plus a column per corpus naming
// the items each charge came from. Cap-swapped sets are NOT repeated here —
// they are a per-item note, and each corpus's report-tables.md already carries
// its own.
func (r *Run) Render(w io.Writer) error {
	rows, err := r.Rows()
	if err != nil {
		return err
	}
	if err := mdreport.Section(w, 2, "Run routing ledger (per-person per-run cap: %d)",
		mdreport.PerPersonCap); err != nil {
		return err
	}
	if err := mdreport.Para(w, "_%s_", r.scope()); err != nil {
		return err
	}
	if len(rows) == 0 {
		return mdreport.Para(w, "_No routing proposed in this run._")
	}

	header := []string{"person", "proposed", "cap", "status"}
	for _, s := range r.Sweeps {
		header = append(header, s.Corpus.column())
	}
	t := mdreport.NewTable(w, header...)
	for _, row := range rows {
		cells := []string{
			mdreport.Escape(row.Login),
			fmt.Sprint(row.Count),
			fmt.Sprint(mdreport.PerPersonCap),
			mdreport.Status(row.Count),
		}
		for _, s := range r.Sweeps {
			cells = append(cells, mdreport.Links(row.Items[s.Corpus], s.Corpus.link()))
		}
		t.Row(cells...)
	}
	return t.Err()
}

// scope says which dirs the totals span and why they can exceed what either
// corpus ledger in the same report shows.
func (r *Run) scope() string {
	spans := make([]string, 0, len(r.Sweeps))
	for _, s := range r.Sweeps {
		spans = append(spans,
			fmt.Sprintf("%s (%s)", mdreport.Escape(filepath.Base(s.Dir)), s.Corpus))
	}
	if len(spans) == 1 {
		return fmt.Sprintf("Spans %s, this run's only sweep dir.", spans[0])
	}
	return fmt.Sprintf("Spans %s. The cap counts items per person per RUN, so a "+
		"person under it in both per-corpus ledgers can still breach it here.",
		strings.Join(spans, " and "))
}

// Generate writes LedgerFile into every dir of the run and reports the totals
// on log, naming anyone over the cap. Rendering completes before any file is
// touched: a failure must leave the previous ledger in place rather than
// truncate it.
//
// Being over cap is not a tool failure — the fragment has to reach the report
// either way, and the breach is the maintainer's call at phase 4 — so it is
// reported loudly and exits 0. Only a broken input exits non-zero.
func Generate(dirs []string, log io.Writer) error {
	run, err := Load(dirs...)
	if err != nil {
		return err
	}
	rows, err := run.Rows()
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := run.Render(&buf); err != nil {
		return err
	}

	var written []string
	for _, s := range run.Sweeps {
		out := filepath.Join(s.Dir, LedgerFile)
		if err := os.WriteFile(out, buf.Bytes(), 0o644); err != nil {
			return err
		}
		written = append(written, filepath.Join(filepath.Base(s.Dir), LedgerFile))
	}

	spans := make([]string, 0, len(run.Sweeps))
	for _, s := range run.Sweeps {
		spans = append(spans, fmt.Sprintf("%s (%s, %d item(s))",
			filepath.Base(s.Dir), s.Corpus, len(s.Proposals)))
	}
	if _, err := fmt.Fprintf(log, "%s: %d person/people proposed across the run\n",
		strings.Join(spans, " + "), len(rows)); err != nil {
		return err
	}
	over := over(rows)
	if len(over) == 0 {
		if _, err := fmt.Fprintf(log, "nobody over the per-person per-run cap of %d\n",
			mdreport.PerPersonCap); err != nil {
			return err
		}
	} else {
		breaches := make([]string, 0, len(over))
		for _, row := range over {
			breaches = append(breaches, fmt.Sprintf("%s %d/%d",
				row.Login, row.Count, mdreport.PerPersonCap))
		}
		if _, err := fmt.Fprintf(log, "OVER CAP (%d): %s\n",
			len(over), strings.Join(breaches, ", ")); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(log, "wrote %s\n", strings.Join(written, ", "))
	return err
}
