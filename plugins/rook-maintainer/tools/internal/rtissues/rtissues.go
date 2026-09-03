// Package rtissues mines issue participation per area for the rook-triage kb
// refresh — the "Issue participation per area label (who answers what)" signal
// of skills/rook-triage/references/kb-refresh.md, which is this package's spec.
//
// What it decides: an issue reaches an area through the label → area table of
// skills/rook-triage/references/label-map.md (parsed by internal/actions, which
// owns that table's grammar), and a login is credited once per issue per area
// for having commented on it inside the window. The issue's own author is not
// answering it and a bot is not a person, so neither is credited; a login the
// internal/mentions grammar rejects is dropped rather than written through, the
// way rt-commits refuses to invent one. The provenance counts what the mine
// could not place — unlabelled issues and rejected logins — so a thin area can
// be told from a mine that lost its input; the comments it leaves uncredited —
// the author's, a bot's, and those outside the window — are not counted.
//
// The {data, flags} contract, its flag vocabulary and the rule that nothing
// here resolves a flag are internal/rtanalyze's; this is the issue-side
// sibling and shares them by calling it, and it scopes a truncation the way
// rtanalyze does — to an issue some area counted.
//
// The export it reads carries every issue's title and body. No struct here
// binds either, so no issue text can reach the output, and nothing here
// touches the network.
package rtissues

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/mentions"
	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/rtanalyze"
	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/rtfetch"
)

// CommentPage is the number of comments gh exports per issue. The export
// carries no page info, so a list of exactly this length is the only evidence
// that there may be more.
const CommentPage = 100

// topParticipants is how deep into an area's ranking an unknown identity is
// worth a question: the head is what a routing decision would read.
const topParticipants = 3

// Comment is a comment's author and when it was written; its text is not read.
type Comment struct {
	Login string
	At    time.Time
}

// Issue is one element of the gh export, reduced to what the mine counts.
type Issue struct {
	Number    int
	Author    string
	Labels    []string
	CreatedAt time.Time
	Comments  []Comment
}

type Options struct {
	Now    time.Time
	Months int
	// Areas maps a label to its areas; see actions.ParseLabelAreas.
	Areas map[string][]string
	// Roster is lowercased (rtanalyze.Lowered) and nil when the caller gave
	// none, which suppresses identity-unknown rather than flagging everyone.
	Roster map[string]bool
	// Top caps each area's ranking in the document; the head is what a
	// routing decision reads, and zero keeps the whole ranking.
	Top     int
	OutPath string
}

// Result is the miner contract document plus the one-line run summary and the
// flags, which the caller renders through FlagBrief.
type Result struct {
	Doc     rtanalyze.Obj
	Summary string
	Flags   []rtanalyze.Flag
}

type wireActor struct {
	Login string `json:"login"`
}

func (a *wireActor) login() string {
	if a == nil {
		return ""
	}
	return a.Login
}

type wireIssue struct {
	Number    int        `json:"number"`
	Author    *wireActor `json:"author"`
	CreatedAt string     `json:"createdAt"`
	Labels    []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Comments []struct {
		Author    *wireActor `json:"author"`
		CreatedAt string     `json:"createdAt"`
	} `json:"comments"`
}

// ExportCommand is the command line whose output Load reads: the wire structs
// above bind exactly its --json fields, and rt-issues prints it, so keep it
// copy-pasteable.
const ExportCommand = "gh issue list -R rook/rook --state all --limit 2000 " +
	"--json number,author,labels,createdAt,comments > issues.json"

// Load reads the JSON array ExportCommand writes.
//
// A timestamp that does not parse fails the run: every count and the
// provenance window rest on them, and a mine that silently treated an
// unreadable date as out of window would answer with a plausible thin area.
func Load(path string) ([]Issue, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var wire []wireIssue
	if err := json.Unmarshal(raw, &wire); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	issues := make([]Issue, 0, len(wire))
	for _, w := range wire {
		at, err := rtanalyze.ParseISO(w.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("%s: issue #%d createdAt: %w", path, w.Number, err)
		}
		issue := Issue{Number: w.Number, Author: w.Author.login(), CreatedAt: at}
		for _, l := range w.Labels {
			issue.Labels = append(issue.Labels, l.Name)
		}
		for i, c := range w.Comments {
			cat, err := rtanalyze.ParseISO(c.CreatedAt)
			if err != nil {
				return nil, fmt.Errorf("%s: issue #%d comment %d createdAt: %w",
					path, w.Number, i+1, err)
			}
			issue.Comments = append(issue.Comments, Comment{Login: c.Author.login(), At: cat})
		}
		issues = append(issues, issue)
	}
	return issues, nil
}

type tally struct {
	counts     map[string]map[string]int
	skipped    map[string]bool
	truncated  []int
	unlabelled int
	oldest     time.Time
}

// Mine counts participation and builds the miner contract document.
func Mine(issues []Issue, opts Options) (*Result, error) {
	if opts.Months <= 0 {
		return nil, fmt.Errorf("--months must be positive, got %d", opts.Months)
	}
	if len(opts.Areas) == 0 {
		return nil, fmt.Errorf("the label map yielded no label → area pairs")
	}
	cutoff, _, err := rtfetch.WindowCutoff(opts.Now, float64(opts.Months))
	if err != nil {
		return nil, err
	}

	t := &tally{counts: map[string]map[string]int{}, skipped: map[string]bool{}}
	for _, issue := range issues {
		if t.oldest.IsZero() || issue.CreatedAt.Before(t.oldest) {
			t.oldest = issue.CreatedAt
		}
		areas := areasOf(issue.Labels, opts.Areas)
		if len(areas) == 0 {
			t.unlabelled++
			continue
		}
		participants := t.participants(issue, cutoff)
		if len(participants) == 0 {
			continue
		}
		if len(issue.Comments) == CommentPage {
			t.truncated = append(t.truncated, issue.Number)
		}
		for _, login := range participants {
			for _, area := range areas {
				in, ok := t.counts[area]
				if !ok {
					in = map[string]int{}
					t.counts[area] = in
				}
				in[login]++
			}
		}
	}

	ranked := map[string][]string{}
	areasObj := rtanalyze.Obj{}
	for _, area := range sortedKeys(t.counts) {
		logins := rank(t.counts[area])
		if opts.Top > 0 && len(logins) > opts.Top {
			logins = logins[:opts.Top]
		}
		ranked[area] = logins
		in := rtanalyze.Obj{}
		for _, login := range logins {
			in = append(in, rtanalyze.Member{Key: login, Val: t.counts[area][login]})
		}
		areasObj = append(areasObj, rtanalyze.Member{Key: area, Val: in})
	}

	flags := t.truncationFlags()
	flags = append(flags, identityFlags(ranked, t.counts, opts.Roster)...)

	doc := rtanalyze.Obj{
		{Key: "data", Val: rtanalyze.Obj{
			{Key: "areas", Val: areasObj},
			{Key: "provenance", Val: rtanalyze.Obj{
				{Key: "issues", Val: len(issues)},
				{Key: "oldest_createdat", Val: iso(t.oldest)},
				{Key: "unlabelled", Val: t.unlabelled},
				{Key: "window_months", Val: opts.Months},
				{Key: "now", Val: iso(opts.Now)},
				{Key: "cutoff", Val: iso(cutoff)},
				{Key: "skipped_logins", Val: len(t.skipped)},
			}},
		}},
		{Key: "flags", Val: rtanalyze.FlagArray(flags)},
	}

	oldest := iso(t.oldest)
	if oldest == "" {
		oldest = "unknown"
	}
	summary := fmt.Sprintf("issues=%d oldest=%s window=%dmo flags=%d -> %s",
		len(issues), oldest, opts.Months, len(flags), opts.OutPath)
	return &Result{Doc: doc, Summary: summary, Flags: flags}, nil
}

// participants returns the logins credited for one issue, deduplicated: an
// area counts the issues a login answered, not the comments it left, so a long
// back-and-forth does not outweigh answering ten separate issues.
//
// The bot test runs before the grammar test, so a `[bot]` suffix — which the
// grammar rejects — is excluded as the bot it is rather than counted as a
// login the mine could not read.
func (t *tally) participants(issue Issue, cutoff time.Time) []string {
	seen := map[string]bool{}
	var logins []string
	for _, c := range issue.Comments {
		if c.At.Before(cutoff) || rtanalyze.IsBot(c.Login) || c.Login == issue.Author {
			continue
		}
		if !mentions.ValidLogin(c.Login) {
			t.skipped[c.Login] = true
			continue
		}
		if !seen[c.Login] {
			seen[c.Login] = true
			logins = append(logins, c.Login)
		}
	}
	return logins
}

func (t *tally) truncationFlags() []rtanalyze.Flag {
	numbers := append([]int(nil), t.truncated...)
	sort.Ints(numbers)
	flags := make([]rtanalyze.Flag, 0, len(numbers))
	for _, n := range numbers {
		flags = append(flags, rtanalyze.Flag{
			Type:     "truncation",
			Item:     fmt.Sprintf("issue #%d", n),
			Evidence: fmt.Sprintf("comments=%d, the whole first page (the export carries no pageInfo)", CommentPage),
			Question: "Issue may have more comments than were exported — participation counts for it may be incomplete.",
		})
	}
	return flags
}

// identityFlags asks about the head of each area's ranking, where an unknown
// login would actually be routed to, and asks once per login however many
// areas it tops.
func identityFlags(ranked map[string][]string, counts map[string]map[string]int, roster map[string]bool) []rtanalyze.Flag {
	if roster == nil {
		return nil
	}
	areas := map[string][]string{}
	totals := map[string]int{}
	var order []string
	for _, area := range sortedKeys(ranked) {
		for _, login := range ranked[area][:min(topParticipants, len(ranked[area]))] {
			if roster[strings.ToLower(login)] {
				continue
			}
			if _, ok := areas[login]; !ok {
				order = append(order, login)
			}
			areas[login] = append(areas[login], area)
			totals[login] += counts[area][login]
		}
	}
	sort.SliceStable(order, func(i, j int) bool { return totals[order[i]] > totals[order[j]] })

	flags := make([]rtanalyze.Flag, 0, len(order))
	for _, login := range order {
		flags = append(flags, rtanalyze.Flag{
			Type:     "identity-unknown",
			Item:     login,
			Evidence: fmt.Sprintf("issues_total=%d across areas: %s", totals[login], strings.Join(areas[login], ", ")),
			Question: "Answers issues among an area's top participants but is not in the roster and not an obvious bot — who is this / a legitimate community responder?",
		})
	}
	return flags
}

// FlagBrief renders the flags for the resolver agent, fenced — rtanalyze's
// brief with this mine's data named. rt-issues writes it to --brief's file.
func FlagBrief(flags []rtanalyze.Flag) string {
	return rtanalyze.FlagBriefFor(flags, "the exported issues — a login is contributor-authored")
}

func areasOf(labels []string, byLabel map[string][]string) []string {
	seen := map[string]bool{}
	var areas []string
	for _, label := range labels {
		for _, area := range byLabel[label] {
			if !seen[area] {
				seen[area] = true
				areas = append(areas, area)
			}
		}
	}
	sort.Strings(areas)
	return areas
}

// rank orders an area's participants by issues answered, then by login, so the
// head of the list is what routing would read and re-runs agree byte for byte.
func rank(counts map[string]int) []string {
	logins := make([]string, 0, len(counts))
	for login := range counts {
		logins = append(logins, login)
	}
	sort.Slice(logins, func(i, j int) bool {
		if counts[logins[i]] != counts[logins[j]] {
			return counts[logins[i]] > counts[logins[j]]
		}
		return strings.ToLower(logins[i]) < strings.ToLower(logins[j])
	})
	return logins
}

func iso(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
