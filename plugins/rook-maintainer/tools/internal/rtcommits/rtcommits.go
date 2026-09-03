// Package rtcommits mines per-area commit authorship for the rook-triage kb
// refresh — the "git log per area path-set (24 months, recency-weighted author
// counts)" signal of skills/rook-triage/references/kb-refresh.md.
//
// It is the commit-side sibling of rtfetch/rtanalyze and shares their decisions
// by CALLING them, not by restating them: the 25-area taxonomy is
// rtanalyze.AreasForPaths, the recency weighting is rtanalyze.AgeDays on the
// same 1.0/0.5/0.25 boundaries, the window is rtfetch.WindowCutoff, and the bot
// rule is rtanalyze.IsBot. The one deliberate local delta is botIdentity, which
// narrows that rule for the login-less identities only git log produces. It
// fills the `commits` and `last_active` columns of kb-refresh.md's `maintainers`
// schema; `tier` comes from CODE-OWNERS and `reviews` from rt-analyze, and
// nothing here invents either.
//
// Merge commits are excluded: git lists no changed paths for one, its author is
// whoever pressed merge rather than whoever wrote the code, and the work it
// carries is already counted in the commits it merges. rtanalyze's self-review
// exclusion has no analogue — on this signal the author IS the signal.
//
// Git identities are not GitHub logins. Identities are unioned across a shared
// name or a shared email, but `login` is filled only from an
// <id>+<login>@users.noreply.github.com address; otherwise it is null and the
// caller maps `name`/`emails` onto the roster. Guessing a login from a display
// name would silently break the join against CODE-OWNERS, so the gap is
// reported instead — identities_without_login in the provenance block.
//
// Areas with no matching commit in the window are absent, not empty: the
// taxonomy list itself lives in rtanalyze, and deriving the keys from the data
// keeps a newly added area from silently missing here. Whether the mine ran at
// all is a provenance question, and the provenance block answers it.
//
// Nothing here touches the network, and nothing writes to the mined checkout.
// Changing the weighting, the exclusions or the schema changes what the kb
// refresh records; kb-refresh.md is that change's spec.
package rtcommits

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/rtanalyze"
	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/rtfetch"
)

const (
	summaryTop = 3

	// weightsNote travels in the provenance so a kb entry says on its face
	// which decay produced it.
	weightsNote = "1.0 <=182d, 0.5 <=365d, 0.25 older (by author date, relative to now)"
)

// noreply matches the only address that carries a GitHub login.
var noreply = regexp.MustCompile(`^(?:[0-9]+\+)?([^@]+)@users\.noreply\.github\.com$`)

// Commit is one record of a GitLogCommand dump: the mailmap-resolved author,
// the author date, and every path the commit touched (both sides of a rename).
type Commit struct {
	SHA   string
	Name  string
	Email string
	When  time.Time
	Paths []string
}

// Source is where the commits came from, recorded verbatim in the provenance.
type Source struct {
	Mode string // "repo" or "log"
	Path string
	Head string
}

type Options struct {
	Now    time.Time
	Months float64
	Source Source
}

// Result is the JSON document plus the human summary the CLI prints without
// --json.
type Result struct {
	Doc     Doc
	Summary []string
}

type Doc struct {
	Provenance Provenance      `json:"provenance"`
	Areas      map[string]Area `json:"areas"`
	Identities []Identity      `json:"identities"`
}

// Provenance is what lets a consumer tell a real empty area from a failed
// mine: the window, what was scanned, what survived each exclusion, and the
// exact command the numbers came from.
type Provenance struct {
	Tool              string  `json:"tool"`
	Source            string  `json:"source"`
	Path              string  `json:"path"`
	Head              string  `json:"head,omitempty"`
	GitLog            string  `json:"git_log"`
	Now               string  `json:"now"`
	Months            float64 `json:"months"`
	WindowDays        int     `json:"window_days"`
	Cutoff            string  `json:"cutoff"`
	RecencyWeights    string  `json:"recency_weights"`
	CommitsScanned    int     `json:"commits_scanned"`
	CommitsInWindow   int     `json:"commits_in_window"`
	CommitsCounted    int     `json:"commits_counted"`
	CommitsBot        int     `json:"commits_bot_excluded"`
	CommitsUnmatched  int     `json:"commits_unmatched"`
	OldestCommit      *string `json:"oldest_commit"`
	NewestCommit      *string `json:"newest_commit"`
	Areas             int     `json:"areas"`
	Identities        int     `json:"identities"`
	IdentitiesNoLogin int     `json:"identities_without_login"`
}

type Area struct {
	Commits int      `json:"commits"`
	Authors []Author `json:"authors"`
}

// Author is one area's row of the maintainers schema. Commits is the raw count,
// WeightedCommits the recency-decayed one the ranking uses.
type Author struct {
	Login           *string `json:"login"`
	Name            string  `json:"name"`
	Commits         int     `json:"commits"`
	WeightedCommits float64 `json:"weighted_commits"`
	LastActive      string  `json:"last_active"`
}

// Identity is the deduplicated author roster, repo-wide within the window — the
// worklist for mapping the ones without a login onto CODE-OWNERS.
type Identity struct {
	Login      *string  `json:"login"`
	Name       string   `json:"name"`
	Emails     []string `json:"emails"`
	Commits    int      `json:"commits"`
	LastActive string   `json:"last_active"`
}

// Mine buckets commits into areas and accrues recency-weighted authorship.
//
// An empty commits slice is an error, not an empty document: every caller
// reaches this with a repo or a dump it believes has history, so nothing to
// scan means the input was wrong, and reporting that as a clean zero is how a
// broken refresh gets written into the kb.
//
// A non-positive --months is rejected for the same reason, before the shared
// window arithmetic gets it: rtfetch.WindowCutoff would resolve it to a cutoff
// in the FUTURE, and a mine of an unreachable window looks exactly like a repo
// with no history.
func Mine(commits []Commit, opts Options) (*Result, error) {
	if len(commits) == 0 {
		return nil, fmt.Errorf("no commits parsed from %s: a mine that scans nothing is a failed mine, not an empty one",
			opts.Source.Path)
	}
	if math.IsNaN(opts.Months) || opts.Months <= 0 {
		return nil, fmt.Errorf("--months=%v: the window must be positive", opts.Months)
	}
	cutoff, days, err := rtfetch.WindowCutoff(opts.Now, opts.Months)
	if err != nil {
		return nil, err
	}

	var window []*Commit
	for i := range commits {
		if !commits[i].When.Before(cutoff) {
			window = append(window, &commits[i])
		}
	}

	people := groupIdentities(window)
	areas := map[string]*areaAcc{}
	var oldest, newest time.Time
	prov := Provenance{
		Tool:            "rt-commits",
		Source:          opts.Source.Mode,
		Path:            opts.Source.Path,
		Head:            opts.Source.Head,
		GitLog:          GitLogCommand(),
		Now:             isoformat(opts.Now),
		Months:          opts.Months,
		WindowDays:      days,
		Cutoff:          isoformat(cutoff),
		RecencyWeights:  weightsNote,
		CommitsScanned:  len(commits),
		CommitsInWindow: len(window),
	}

	for _, c := range window {
		if oldest.IsZero() || c.When.Before(oldest) {
			oldest = c.When
		}
		newest = laterOf(newest, c.When)

		p := people.of(c)
		if p.bot {
			prov.CommitsBot++
			continue
		}
		p.commits++
		p.last = laterOf(p.last, c.When)

		hits := rtanalyze.AreasForPaths(c.Paths)
		if len(hits) == 0 {
			prov.CommitsUnmatched++
			continue
		}
		prov.CommitsCounted++

		w := weight(opts.Now, c.When)
		for _, a := range hits {
			acc := areas[a]
			if acc == nil {
				acc = &areaAcc{byPerson: map[*person]*authorStat{}}
				areas[a] = acc
			}
			acc.commits++
			acc.add(p, w, c.When)
		}
	}

	prov.OldestCommit, prov.NewestCommit = stamp(oldest), stamp(newest)
	doc := Doc{Provenance: prov, Areas: map[string]Area{}, Identities: people.roster()}
	for name, acc := range areas {
		doc.Areas[name] = Area{Commits: acc.commits, Authors: acc.authors()}
	}
	doc.Provenance.Areas = len(doc.Areas)
	doc.Provenance.Identities = len(doc.Identities)
	for _, id := range doc.Identities {
		if id.Login == nil {
			doc.Provenance.IdentitiesNoLogin++
		}
	}
	return &Result{Doc: doc, Summary: summarize(doc)}, nil
}

// Render writes the document the way the kb refresh consumes it: HTML escaping
// off, since paths and display names are not HTML.
func Render(doc Doc) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// weight is rtanalyze's recency decay, on the commit's author date.
func weight(now, when time.Time) float64 {
	switch age := rtanalyze.AgeDays(now, when); {
	case age <= 182:
		return 1.0
	case age <= 365:
		return 0.5
	}
	return 0.25
}

// botIdentity decides whether a git identity is a bot. rtanalyze.IsBot runs on
// the login a noreply address yields — its intended input, and the one every
// bot that commits through the GitHub API has. For an identity with no login
// only the unambiguous "[bot]" marker counts: IsBot's bare "bot" suffix is safe
// on logins but would swallow a human display name like "Talbot".
func botIdentity(login string, names, emails []string) bool {
	if login != "" {
		return rtanalyze.IsBot(login)
	}
	for _, n := range names {
		if strings.HasSuffix(strings.ToLower(n), "[bot]") {
			return true
		}
	}
	for _, e := range emails {
		if strings.HasSuffix(strings.ToLower(localPart(e)), "[bot]") {
			return true
		}
	}
	return false
}

func localPart(email string) string {
	if i := strings.LastIndex(email, "@"); i >= 0 {
		return email[:i]
	}
	return email
}

type person struct {
	names   map[string]int
	emails  map[string]string // lowercased -> as written
	login   string
	name    string
	bot     bool
	commits int
	last    time.Time
}

type identitySet struct {
	dsu     dsu
	byRoot  map[int]*person
	commits map[*Commit]*person
}

// groupIdentities unions the window's authors across a shared name or a shared
// email: "parth-gr" commits under two addresses and "Santosh"/"Santosh Pillai"
// under one, and both are one person for routing purposes. The transitive
// closure is deliberate — the alternative is one kb row per address.
func groupIdentities(window []*Commit) *identitySet {
	set := &identitySet{
		dsu:     dsu{id: map[string]int{}},
		byRoot:  map[int]*person{},
		commits: make(map[*Commit]*person, len(window)),
	}
	for _, c := range window {
		n, e := set.dsu.nodes(c.Name, strings.ToLower(c.Email))
		if n >= 0 && e >= 0 {
			set.dsu.union(n, e)
		}
	}
	for _, c := range window {
		n, e := set.dsu.nodes(c.Name, strings.ToLower(c.Email))
		root := n
		if root < 0 {
			root = e
		}
		root = set.dsu.find(root)
		p := set.byRoot[root]
		if p == nil {
			p = &person{names: map[string]int{}, emails: map[string]string{}}
			set.byRoot[root] = p
		}
		if c.Name != "" {
			p.names[c.Name]++
		}
		if c.Email != "" {
			if _, seen := p.emails[strings.ToLower(c.Email)]; !seen {
				p.emails[strings.ToLower(c.Email)] = c.Email
			}
		}
		set.commits[c] = p
	}
	for _, p := range set.byRoot {
		p.name = primaryName(p.names, p.emails)
		p.login = loginFrom(p.emails)
		p.bot = botIdentity(p.login, sortedKeys(p.names), sortedValues(p.emails))
	}
	return set
}

func (s *identitySet) of(c *Commit) *person { return s.commits[c] }

// roster is the non-bot identities, busiest first.
func (s *identitySet) roster() []Identity {
	people := make([]*person, 0, len(s.byRoot))
	for _, p := range s.byRoot {
		if !p.bot && p.commits > 0 {
			people = append(people, p)
		}
	}
	sort.Slice(people, func(i, j int) bool {
		x, y := people[i], people[j]
		if x.commits != y.commits {
			return x.commits > y.commits
		}
		return x.name < y.name
	})
	out := make([]Identity, 0, len(people))
	for _, p := range people {
		out = append(out, Identity{
			Login:      login(p.login),
			Name:       p.name,
			Emails:     sortedValues(p.emails),
			Commits:    p.commits,
			LastActive: yearMonth(p.last),
		})
	}
	return out
}

// primaryName picks the identity's display name: the most-used spelling, ties
// broken lexicographically. Two components can never share a name — a shared
// name is what unions them — so the pick is unique across the roster and can be
// used as its key.
func primaryName(names map[string]int, emails map[string]string) string {
	best, bestN := "", -1
	for _, n := range sortedKeys(names) {
		if names[n] > bestN {
			best, bestN = n, names[n]
		}
	}
	if best == "" && len(emails) > 0 {
		return sortedValues(emails)[0]
	}
	return best
}

func loginFrom(emails map[string]string) string {
	for _, e := range sortedValues(emails) {
		if m := noreply.FindStringSubmatch(strings.ToLower(e)); m != nil {
			return m[1]
		}
	}
	return ""
}

func login(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

type dsu struct {
	parent []int
	id     map[string]int
}

// nodes returns the node ids of a name and an email, -1 for an absent one.
func (d *dsu) nodes(name, email string) (int, int) {
	n, e := -1, -1
	if name != "" {
		n = d.node("n\x00" + name)
	}
	if email != "" {
		e = d.node("e\x00" + email)
	}
	return n, e
}

func (d *dsu) node(key string) int {
	if i, ok := d.id[key]; ok {
		return i
	}
	i := len(d.parent)
	d.parent = append(d.parent, i)
	d.id[key] = i
	return i
}

func (d *dsu) find(i int) int {
	for d.parent[i] != i {
		d.parent[i] = d.parent[d.parent[i]]
		i = d.parent[i]
	}
	return i
}

func (d *dsu) union(a, b int) {
	if ra, rb := d.find(a), d.find(b); ra != rb {
		d.parent[ra] = rb
	}
}

type authorStat struct {
	p        *person
	commits  int
	weighted float64
	last     time.Time
}

type areaAcc struct {
	commits  int
	byPerson map[*person]*authorStat
	order    []*authorStat
}

func (a *areaAcc) add(p *person, w float64, when time.Time) {
	st := a.byPerson[p]
	if st == nil {
		st = &authorStat{p: p}
		a.byPerson[p] = st
		a.order = append(a.order, st)
	}
	st.commits++
	st.weighted += w
	st.last = laterOf(st.last, when)
}

// authors ranks an area's contributors the way rtanalyze ranks reviewers:
// weight, then raw count, then lowercased name, then first seen.
func (a *areaAcc) authors() []Author {
	ranked := make([]*authorStat, len(a.order))
	copy(ranked, a.order)
	sort.SliceStable(ranked, func(i, j int) bool {
		x, y := ranked[i], ranked[j]
		if x.weighted != y.weighted {
			return x.weighted > y.weighted
		}
		if x.commits != y.commits {
			return x.commits > y.commits
		}
		return strings.ToLower(x.p.name) < strings.ToLower(y.p.name)
	})
	out := make([]Author, 0, len(ranked))
	for _, st := range ranked {
		out = append(out, Author{
			Login:           login(st.p.login),
			Name:            st.p.name,
			Commits:         st.commits,
			WeightedCommits: round2(st.weighted),
			LastActive:      yearMonth(st.last),
		})
	}
	return out
}

func summarize(doc Doc) []string {
	p := doc.Provenance
	head := ""
	if p.Head != "" {
		head = " @ " + shortSHA(p.Head)
	}
	out := []string{
		fmt.Sprintf("source: %s %s%s", p.Source, p.Path, head),
		fmt.Sprintf("window: %s..%s (%v months = %d days, now=%s)",
			day(p.Cutoff), day(p.Now), p.Months, p.WindowDays, p.Now),
		fmt.Sprintf("commits: scanned=%d in_window=%d counted=%d bot=%d unmatched=%d",
			p.CommitsScanned, p.CommitsInWindow, p.CommitsCounted, p.CommitsBot, p.CommitsUnmatched),
		fmt.Sprintf("identities: %d (%d without a GitHub login)", p.Identities, p.IdentitiesNoLogin),
	}
	for _, name := range sortedKeys(doc.Areas) {
		area := doc.Areas[name]
		var top []string
		for i, a := range area.Authors {
			if i == summaryTop {
				break
			}
			who := a.Name
			if a.Login != nil {
				who = *a.Login
			}
			top = append(top, fmt.Sprintf("%s(%s/%d)", who, trimFloat(a.WeightedCommits), a.Commits))
		}
		out = append(out, fmt.Sprintf("  %-20s %5d  %s", name, area.Commits, strings.Join(top, ", ")))
	}
	return out
}

func isoformat(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func yearMonth(t time.Time) string { return t.UTC().Format("2006-01") }

func day(iso string) string { return strings.SplitN(iso, "T", 2)[0] }

func shortSHA(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

// stamp is null for the zero time: an empty window has no oldest commit, and
// saying so beats printing the epoch.
func stamp(t time.Time) *string {
	if t.IsZero() {
		return nil
	}
	s := isoformat(t)
	return &s
}

func laterOf(cur, t time.Time) time.Time {
	if t.After(cur) {
		return t
	}
	return cur
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }

func trimFloat(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedValues(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for _, k := range sortedKeys(m) {
		out = append(out, m[k])
	}
	return out
}
