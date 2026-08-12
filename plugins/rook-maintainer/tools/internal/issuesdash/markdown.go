package issuesdash

import (
	"fmt"
	"io"
	"strings"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/mdreport"
)

var issueHeader = []string{"#", "kind", "summary", "actions", "disposition",
	"pr #", "assignees", "mentions", "labels"}

// RenderMarkdown writes the per-item table and the mention ledger of
// report.md. It renders the SAME Page the dashboard does, so a cell can only
// differ between the two artifacts by differing here, not by being derived
// twice. Synthesis — disposition evidence, cross-cutting notes, repo hygiene —
// is not machine-derivable and stays with the model that writes the notes half.
//
// There is no skipped section here: skip rows are a PR-side rule
// (draft/bot/do-not-merge), and an issues sweep assesses everything it fetched.
func RenderMarkdown(w io.Writer, p Page) error {
	if err := mdreport.Section(w, 2, "Assessed issues (%d)", len(p.Rows)); err != nil {
		return err
	}
	if len(p.Rows) == 0 {
		if err := mdreport.Para(w, "_None._"); err != nil {
			return err
		}
	} else if err := table(w, p.Rows); err != nil {
		return err
	}

	entries, swaps := ledger(p)
	return mdreport.Ledger{
		Heading: "Mention ledger",
		Column:  "mentioned",
		Empty:   "No @-mentions proposed in this sweep.",
		Entries: entries,
		Swaps:   swaps,
	}.Write(w)
}

func table(w io.Writer, rows []Row) error {
	t := mdreport.NewTable(w, issueHeader...)
	for _, r := range rows {
		summary := mdreport.Escape(r.Title)
		if r.Missing {
			summary = mdreport.NoSnapshot
		}
		t.Row(
			mdreport.IssueLink(r.Number),
			mdreport.Escape(r.Kind),
			summary,
			actions(r.Actions),
			prose(r.Disposition),
			mdreport.Links(r.Refs, mdreport.PullLink),
			logins(r.Assignees),
			mentions(r.Mentions),
			labels(r.Labels),
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
		b.WriteString(mdreport.Escape(s.Text))
	}
	return b.String()
}

// actions are sentences, so they are separated by a semicolon rather than the
// comma the identifier lists use, and linkified for the same reason the
// disposition is: a number a maintainer has to act on should be one click.
func actions(l List) string {
	out := make([]string, 0, len(l.Items))
	for _, a := range l.Items {
		out = append(out, prose(Linkify(a)))
	}
	return strings.Join(out, "; ")
}

func logins(names []string) string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		out = append(out, mdreport.Escape(displayName(n)))
	}
	return strings.Join(out, ", ")
}

// displayName matches the dashboard's one renaming: the review bot posts under
// an app login no one recognizes.
func displayName(login string) string {
	if login == copilotLogin {
		return "Copilot"
	}
	return login
}

// mentions keeps the dashboard's split: who the thread already mentioned,
// then what triage proposes to add, bolded so the two never blur.
func mentions(rows []MentionRow) string {
	var seen, proposed []string
	for _, m := range rows {
		switch {
		case m.More > 0:
			seen = append(seen, fmt.Sprintf("+%d more", m.More))
		case m.Proposed:
			proposed = append(proposed, mdreport.Bold(displayName(m.Login)))
		default:
			seen = append(seen, mdreport.Escape(displayName(m.Login)))
		}
	}
	return withProposals(strings.Join(seen, ", "), proposed)
}

// labels shows the CURRENT labels, with the additions triage proposes marked
// distinctly — the issues table is the only one that carries proposals.
func labels(c LabelsCell) string {
	current := make([]string, 0, len(c.Current.Items))
	for _, l := range c.Current.Items {
		current = append(current, mdreport.Escape(l))
	}
	proposed := make([]string, 0, len(c.Proposed.Items))
	for _, l := range c.Proposed.Items {
		proposed = append(proposed, mdreport.Bold(l))
	}
	return withProposals(strings.Join(current, ", "), proposed)
}

func withProposals(current string, proposed []string) string {
	if len(proposed) == 0 {
		return current
	}
	add := "propose: " + strings.Join(proposed, ", ")
	if current == "" {
		return add
	}
	return current + " · " + add
}

// ProposedMentions lists the @-mentions triage proposes, one entry per issue in
// row order. Mentions already in a thread are somebody else's ping and never
// count against the cap.
//
// It is the single extraction both this sweep's own ledger and the run-scoped
// ledger (internal/runledger, which sums a `both` run's two dirs against the
// cap) count, so the two cannot disagree about what this corpus proposed.
func (p Page) ProposedMentions() []mdreport.Proposal {
	out := make([]mdreport.Proposal, 0, len(p.Rows))
	for _, r := range p.Rows {
		var proposed []string
		for _, m := range r.Mentions {
			if m.Proposed {
				proposed = append(proposed, displayName(m.Login))
			}
		}
		out = append(out, mdreport.Proposal{Number: r.Number, Logins: proposed})
	}
	return out
}

func ledger(p Page) ([]mdreport.Entry, []mdreport.Swap) {
	var swaps []mdreport.Swap
	for _, r := range p.Rows {
		if r.CapNote != "" {
			swaps = append(swaps, mdreport.Swap{
				Link: mdreport.IssueLink(r.Number),
				Note: prose(Linkify(r.CapNote)),
			})
		}
	}
	return mdreport.Counts(mdreport.Group(p.ProposedMentions())), swaps
}

// Missing lists the numbers the snapshot has no entry for, in row order. Their
// summary, labels and assignees are unavailable, so a caller must say so
// rather than let the table imply "none".
func (p Page) Missing() []string {
	var out []string
	for _, r := range p.Rows {
		if r.Missing {
			out = append(out, r.Number)
		}
	}
	return out
}
