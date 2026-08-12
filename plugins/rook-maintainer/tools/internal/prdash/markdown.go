package prdash

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/mdreport"
)

var (
	prHeader = []string{"#", "kind", "summary", "CI", "actions", "disposition",
		"issue #", "assignees", "reviewers", "labels"}
	skipHeader = []string{"#", "class", "author", "summary"}
)

// RenderMarkdown writes the per-item tables and the reviewer ledger of
// report.md, in the order reporting.md fixes: assessed rows by number, then
// the skipped section at the bottom with the WIP signal-rows — assessed, but
// skip-class — ahead of the draft/bot rows.
//
// It renders the SAME Page the dashboard does, so a cell can only differ
// between the two artifacts by differing here, not by being derived twice.
// Synthesis (disposition evidence, cross-cutting notes, repo hygiene) is not
// machine-derivable and stays with the model that writes the notes half.
func RenderMarkdown(w io.Writer, p Page) error {
	if err := mdreport.Section(w, 2, "Assessed PRs (%d)", len(p.Assessed)); err != nil {
		return err
	}
	if err := prTable(w, p.Assessed); err != nil {
		return err
	}

	if err := mdreport.Section(w, 2, "Skipped (%d)",
		len(p.WIP)+len(p.Skipped)); err != nil {
		return err
	}
	if err := mdreport.Section(w, 3, "WIP signal-rows (%d)", len(p.WIP)); err != nil {
		return err
	}
	if err := prTable(w, p.WIP); err != nil {
		return err
	}
	if err := mdreport.Section(w, 3, "Draft, bot and do-not-merge (%d)",
		len(p.Skipped)); err != nil {
		return err
	}
	if err := skipTable(w, p.Skipped); err != nil {
		return err
	}

	entries, swaps := ledger(p)
	return mdreport.Ledger{
		Heading: "Reviewer ledger",
		Column:  "reviewer",
		Empty:   "No reviewers proposed in this sweep.",
		Entries: entries,
		Swaps:   swaps,
	}.Write(w)
}

func prTable(w io.Writer, rows []Row) error {
	if len(rows) == 0 {
		return mdreport.Para(w, "_None._")
	}
	t := mdreport.NewTable(w, prHeader...)
	for _, r := range rows {
		summary := mdreport.Escape(r.Title)
		if r.Missing {
			summary = mdreport.NoSnapshot
		}
		t.Row(
			mdreport.PullLink(strconv.Itoa(r.Number)),
			mdreport.Escape(r.Kind),
			summary,
			r.Chip.Short,
			prose(Linkify(r.Next)),
			prose(r.Disposition),
			mdreport.Links(r.Refs, mdreport.IssueLink),
			mdLogins(r.Assignees),
			mdReviewers(r.Reviewers),
			mdLabels(r.Labels),
		)
	}
	return t.Err()
}

func skipTable(w io.Writer, rows []SkipRow) error {
	if len(rows) == 0 {
		return mdreport.Para(w, "_None._")
	}
	t := mdreport.NewTable(w, skipHeader...)
	for _, r := range rows {
		t.Row(
			mdreport.PullLink(strconv.Itoa(r.Number)),
			mdreport.Escape(r.Class),
			mdreport.Escape(r.Author),
			prose(r.Title),
		)
	}
	return t.Err()
}

// prose renders linkified text: plain runs escaped, references as links.
func prose(segs []Seg) string {
	var b strings.Builder
	for _, s := range segs {
		if s.Ref != "" {
			b.WriteString(mdreport.IssueLink(s.Ref))
			continue
		}
		b.WriteString(mdreport.Escape(s.Plain))
	}
	return b.String()
}

func mdLogins(users []User) string {
	out := make([]string, 0, len(users))
	for _, u := range users {
		out = append(out, mdreport.Escape(u.Name))
	}
	return strings.Join(out, ", ")
}

// mdReviewers keeps the dashboard's reviewer ordering — pending requests, then
// the latest review per person, then triage's proposals — and spells out in
// words what the dashboard says with an icon and a tooltip.
func mdReviewers(rows []Reviewer) string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		name := mdreport.Escape(r.User.Name)
		switch {
		case r.Class == classProposed && r.Note != "":
			out = append(out, fmt.Sprintf("%s (propose: %s)",
				mdreport.Bold(r.User.Name), mdreport.Escape(r.Note)))
		case r.Class == classProposed:
			out = append(out, mdreport.Bold(r.User.Name)+" (propose)")
		default:
			out = append(out, fmt.Sprintf("%s (%s)", name, mdreport.Escape(r.Title)))
		}
	}
	return strings.Join(out, ", ")
}

// mdLabels shows what is on the PR. Triage proposes no PR labels (rook
// convention), so unlike the issues table there is nothing to mark.
func mdLabels(labels []string) string {
	out := make([]string, 0, len(labels))
	for _, l := range labels {
		out = append(out, mdreport.Escape(l))
	}
	return strings.Join(out, ", ")
}

// ProposedReviewers lists every reviewer triage proposed, one entry per PR in
// row order, WIP rows included: the cap bounds what one person receives across
// the sweep, not what one table says.
//
// It is the single extraction both this sweep's own ledger and the run-scoped
// ledger (internal/runledger, which sums a `both` run's two dirs against the
// cap) count, so the two cannot disagree about what this corpus proposed.
func (p Page) ProposedReviewers() []mdreport.Proposal {
	var out []mdreport.Proposal
	for _, r := range slices.Concat(p.Assessed, p.WIP) {
		var logins []string
		for _, rv := range r.Reviewers {
			if rv.Class == classProposed {
				logins = append(logins, rv.User.Name)
			}
		}
		out = append(out, mdreport.Proposal{
			Number: strconv.Itoa(r.Number),
			Logins: logins,
		})
	}
	return out
}

func ledger(p Page) ([]mdreport.Entry, []mdreport.Swap) {
	var swaps []mdreport.Swap
	for _, r := range slices.Concat(p.Assessed, p.WIP) {
		if r.CapNote != "" {
			swaps = append(swaps, mdreport.Swap{
				Link: mdreport.PullLink(strconv.Itoa(r.Number)),
				Note: prose(Linkify(r.CapNote)),
			})
		}
	}
	slices.SortStableFunc(swaps, func(a, b mdreport.Swap) int {
		return strings.Compare(a.Link, b.Link)
	})
	return mdreport.Counts(mdreport.Group(p.ProposedReviewers())), swaps
}

// Missing lists the assessed numbers the snapshot has no entry for, in row
// order. Their summary, labels, assignees, reviews and CI counts are all
// unavailable, so a caller must say so rather than let the table imply "none".
func (p Page) Missing() []string {
	var out []string
	for _, r := range slices.Concat(p.Assessed, p.WIP) {
		if r.Missing {
			out = append(out, strconv.Itoa(r.Number))
		}
	}
	return out
}

// GenerateMarkdown writes <dir>/report-tables.md, the machine-derivable half
// of report.md. It refuses a sweep dir with no batch file rather than publish
// an empty table: "no assessments" and "no input" look identical in the
// output and only one of them is a finished sweep.
func GenerateMarkdown(dir string, log io.Writer) error {
	s, err := Load(dir)
	if err != nil {
		return err
	}
	if len(s.Batches) == 0 {
		return fmt.Errorf("%s: no batch-*.json to report on", s.Dir)
	}
	page := s.Page()
	var buf bytes.Buffer
	if err := RenderMarkdown(&buf, page); err != nil {
		return err
	}
	out := filepath.Join(s.Dir, mdreport.ReportFile)
	if err := os.WriteFile(out, buf.Bytes(), 0o644); err != nil {
		return err
	}
	if missing := page.Missing(); len(missing) > 0 {
		if _, err := fmt.Fprintf(log, "warning: %d assessed PR(s) absent from "+
			"snapshot.json, marked in the summary column: %s\n",
			len(missing), strings.Join(missing, ", ")); err != nil {
			return err
		}
	}
	entries, swaps := ledger(page)
	_, err = fmt.Fprintf(log, "%s/%s: %d assessed, %d WIP, %d skipped; "+
		"%d reviewer(s) proposed, %d cap-swapped set(s)\n",
		filepath.Base(s.Dir), mdreport.ReportFile, len(page.Assessed), len(page.WIP),
		len(page.Skipped), len(entries), len(swaps))
	return err
}
