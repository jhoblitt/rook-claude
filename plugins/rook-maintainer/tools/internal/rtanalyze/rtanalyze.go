// Package rtanalyze is the deterministic analysis layer of the rook-triage kb
// refresh (skills/rook-triage/references/routing.md).
//
// It consumes the rt_fetch output (rt_prs.jsonl + rt_fetch_state.json), buckets
// merged PRs into the kb v3 area taxonomy and emits the two-tier miner contract
// — {"data": {...}, "flags": [...]} — so the orchestrator resolves trivial flags
// and sends only survivors to a resolver agent.
//
// Area taxonomy = kb v3 (25 areas; rebucketed 2026-07-23: +build/design/
// discover, core broadened). The classes the "Deliberately unbucketed"
// paragraph of skills/rook-triage/references/label-map.md names match no
// area — they surface as bucket-ambiguity flags to confirm the gap is still
// intentional, not silently dropped.
//
// Data: per-area top reviewers (recency-weighted: 1.0 <=6mo, 0.5 <=12mo, 0.25
// older; bots and self-reviews excluded; counted per review event) + 5 most
// recent items, plus authors_last_merged (YYYY-MM per author).
// Flags: bucket-ambiguity (zero-match groups, >=6-area overmatch split
// apis-driven vs cross-cutting) · truncation (scoped to counted PRs) ·
// spec-boundary (fetch errors, unclean stop reason) · identity-unknown (top
// reviewers outside the CODE-OWNERS roster) · coverage-gap (area has PRs but
// zero reviews).
//
// Nothing here touches the network.
package rtanalyze

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var allAreas = []string{
	"object", "object-multisite", "object-cosi", "object-bucket-claims",
	"ceph-mon", "ceph-osd", "ceph-mgr", "ceph-dashboard", "filesystem",
	"ceph-nfs", "csi", "block", "helm", "docs", "design", "ci", "test",
	"crd", "networking", "nvmeof", "ceph-external", "discover",
	"monitoring", "build", "core",
}

var bots = []string{"mergify", "dependabot", "github-actions"}

var repoMeta = map[string]bool{
	".mergify.yml": true, "PendingReleaseNotes.md": true, "ROADMAP.md": true,
	"ADOPTERS.md": true, "CODE-OWNERS": true, "README.md": true,
	"OWNERS.md": true, "SECURITY.md": true, "LICENSE": true,
	".gitignore": true, ".github/PULL_REQUEST_TEMPLATE.md": true,
	"mkdocs.yml": true, "CONTRIBUTING.md": true, "AGENTS.md": true,
}

// codegen is docs-sync.md's generated set minus the paths the table already
// places — deploy/examples/crds.yaml, unbucketed by its prefix, and the
// generated Go under pkg/apis/** and pkg/client/**, crd by their row. What
// is left sits under a bucketed prefix, where the rule would read a
// regeneration as the helm or docs change a human made, so a path added
// there is added here.
var codegen = map[string]bool{
	"deploy/charts/rook-ceph/templates/resources.yaml": true,
	"Documentation/CRDs/specification.md":              true,
	"Documentation/Helm-Charts/operator-chart.md":      true,
	"Documentation/Helm-Charts/ceph-cluster-chart.md":  true,
}

// summaryAreas are echoed to stderr as a smoke test of the run.
var summaryAreas = []string{"object", "csi", "core", "build"}

// PR is one record of rt_prs.jsonl.
type PR struct {
	Number   int    `json:"number"`
	Title    string `json:"title"`
	MergedAt string `json:"mergedAt"`
	Author   *struct {
		Login string `json:"login"`
	} `json:"author"`
	Files struct {
		Nodes []*struct {
			Path string `json:"path"`
		} `json:"nodes"`
	} `json:"files"`
	Reviews struct {
		Nodes []*struct {
			Author *struct {
				Login string `json:"login"`
			} `json:"author"`
		} `json:"nodes"`
	} `json:"reviews"`
}

// Truncation is one rt_fetch_state.json truncation entry.
type Truncation struct {
	Number   int     `json:"number"`
	Kind     string  `json:"kind"`
	MergedAt *string `json:"mergedAt"`
}

// State is rt_fetch_state.json, the fetch layer's provenance record.
type State struct {
	PagesFetched   *json.Number `json:"pages_fetched"`
	Counted        *json.Number `json:"counted"`
	OldestMergedAt *string      `json:"oldest_mergedat"`
	StopReason     *string      `json:"stop_reason"`
	Errors         []string     `json:"errors"`
	Truncations    []Truncation `json:"truncations"`
}

// Flag is one entry of the miner contract's flags array.
type Flag struct {
	Type     string
	Item     string
	Evidence string
	Question string
}

type Options struct {
	OutPath string
	Top     int
	Now     time.Time
	// Roster must be lowercased; see Lowered.
	Roster map[string]bool
}

// Result is the miner contract document plus the stderr run summary.
type Result struct {
	Doc     Obj
	Summary []string
}

func hasAnyPrefix(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// baseName is os.path.basename, not path.Base: the latter returns "." for an
// empty path and strips trailing slashes, which would change what the build
// rules below match.
func baseName(p string) string {
	return p[strings.LastIndex(p, "/")+1:]
}

// AreasFor maps one repo-relative path to the v3 area taxonomy. It is the
// deterministic layer rook-triage's references/label-map.md table specifies,
// and rook-triage's phase 0 stamps its answer into snapshot.json via
// sweep-prefetch; changing a rule here changes both, and the table is that
// change's spec.
func AreasFor(p string) map[string]bool {
	out := map[string]bool{}
	if codegen[p] {
		return out
	}
	base := baseName(p)
	goManifest := base == "go.mod" || base == "go.sum"
	if strings.Contains(strings.ToLower(p), "cosi") {
		out["object-cosi"] = true
	}
	if hasAnyPrefix(p, "pkg/operator/ceph/object/", "pkg/daemon/ceph/rgw/") {
		if !strings.Contains(p, "/cosi/") {
			out["object"] = true
		}
		if containsAny(p, "multisite", "zone", "zonegroup", "realm") {
			out["object-multisite"] = true
		}
		if strings.Contains(p, "/bucket/") {
			out["object-bucket-claims"] = true
		}
	}
	if strings.HasPrefix(p, "pkg/operator/ceph/cluster/mon/") {
		out["ceph-mon"] = true
	}
	if hasAnyPrefix(p, "pkg/operator/ceph/cluster/osd/", "pkg/daemon/ceph/osd/") {
		out["ceph-osd"] = true
	}
	if strings.HasPrefix(p, "pkg/operator/ceph/cluster/mgr/") {
		out["ceph-mgr"] = true
		if strings.Contains(p, "dashboard") {
			out["ceph-dashboard"] = true
		}
	}
	if strings.HasPrefix(p, "pkg/operator/ceph/file/") {
		out["filesystem"] = true
	}
	if strings.HasPrefix(p, "pkg/operator/ceph/nfs/") {
		out["ceph-nfs"] = true
	}
	if strings.HasPrefix(p, "pkg/operator/ceph/csi/") {
		out["csi"] = true
	}
	if strings.HasPrefix(p, "pkg/operator/ceph/pool/") || strings.Contains(p, "rbdmirror") ||
		(strings.Contains(p, "/rbd") &&
			!hasAnyPrefix(p, "pkg/operator/ceph/csi/", "deploy/examples/csi/")) {
		out["block"] = true
	}
	if strings.HasPrefix(p, "deploy/charts/") {
		out["helm"] = true
	}
	if strings.HasPrefix(p, "Documentation/") {
		out["docs"] = true
	}
	if strings.HasPrefix(p, "design/") {
		out["design"] = true
	}
	if hasAnyPrefix(p, ".github/workflows/", "tests/scripts/") {
		out["ci"] = true
	} else if strings.HasPrefix(p, "tests/") {
		out["test"] = true
	}
	if hasAnyPrefix(p, "pkg/apis/", "pkg/client/") && !goManifest {
		out["crd"] = true
	}
	if strings.Contains(p, "multus") || strings.HasPrefix(p, "pkg/operator/ceph/controller/network") {
		out["networking"] = true
	}
	if strings.Contains(p, "nvmeof") {
		out["nvmeof"] = true
	}
	// pkg/client/ is client-go codegen, whose externalversions package has
	// nothing to do with an external Ceph cluster.
	if strings.Contains(p, "external") && !strings.HasPrefix(p, "pkg/client/") {
		out["ceph-external"] = true
	}
	if hasAnyPrefix(p, "pkg/operator/discover/", "pkg/daemon/discover/") {
		out["discover"] = true
	}
	if strings.HasPrefix(p, "pkg/operator/ceph/reporting/") || strings.Contains(p, "exporter") ||
		strings.Contains(p, "monitoring") {
		out["monitoring"] = true
	}
	if goManifest || p == "go.work" || p == "go.work.sum" ||
		hasAnyPrefix(p, "build/", "images/") ||
		base == "Makefile" || strings.HasSuffix(base, ".mk") ||
		hasAnyPrefix(base, ".golangci", ".commitlintrc", ".codespell") {
		out["build"] = true
	}
	if len(out) == 0 {
		if hasAnyPrefix(p, "pkg/operator/", "pkg/daemon/ceph/", "pkg/util/",
			"pkg/clusterd/", "cmd/", "pkg/") {
			out["core"] = true
		}
	}
	return out
}

// IsBot is the kb refresh's bot rule, and takes a GitHub LOGIN: the bare "bot"
// suffix is unambiguous on a login but would swallow a human display name, so a
// caller holding something else (rtcommits, mining git identities) must decide
// for itself what it may hand over.
func IsBot(login string) bool {
	ll := strings.ToLower(login)
	return hasAnyPrefix(ll, bots...) || strings.Contains(ll, "copilot") ||
		strings.HasSuffix(ll, "bot") || strings.HasSuffix(ll, "[bot]")
}

// AreasForPaths classifies a changed-path set, returning the areas in
// allAreas order so a stamped snapshot and a re-run agree byte for byte.
func AreasForPaths(paths []string) []string {
	hit := map[string]bool{}
	for _, p := range paths {
		for a := range AreasFor(p) {
			hit[a] = true
		}
	}
	out := make([]string, 0, len(hit))
	for _, a := range allAreas {
		if hit[a] {
			out = append(out, a)
		}
	}
	return out
}

var codeOwnersKey = regexp.MustCompile(`^(approvers|reviewers):`)

// ParseCodeOwners mines the flat CODE-OWNERS roster: every "- login" under an
// approvers:/reviewers: key, until a non-comment non-list line closes it.
func ParseCodeOwners(r io.Reader) (map[string]bool, error) {
	roster := map[string]bool{}
	inKey := false
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		s := strings.TrimSpace(sc.Text())
		if codeOwnersKey.MatchString(s) {
			inKey = true
			continue
		}
		if inKey && strings.HasPrefix(s, "- ") {
			roster[strings.TrimSpace(s[2:])] = true
		} else if s != "" && !strings.HasPrefix(s, "#") && !strings.HasPrefix(s, "-") {
			inKey = false
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return roster, nil
}

// ParseRoster reads the --roster form: comma-separated logins.
func ParseRoster(s string) map[string]bool {
	roster := map[string]bool{}
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			roster[p] = true
		}
	}
	return roster
}

// Lowered folds a roster for case-insensitive membership tests.
func Lowered(roster map[string]bool) map[string]bool {
	out := make(map[string]bool, len(roster))
	for k := range roster {
		out[strings.ToLower(k)] = true
	}
	return out
}

// LoadState reads rt_fetch_state.json.
func LoadState(path string) (*State, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var st State
	if err := json.Unmarshal(b, &st); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &st, nil
}

// LoadPRs reads rt_prs.jsonl, deduplicating by PR number: the last record for a
// number wins, at the position where that number first appeared.
func LoadPRs(path string) ([]*PR, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var order []int
	byNumber := map[int]*PR{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var pr PR
		if err := json.Unmarshal([]byte(line), &pr); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if _, seen := byNumber[pr.Number]; !seen {
			order = append(order, pr.Number)
		}
		byNumber[pr.Number] = &pr
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	prs := make([]*PR, 0, len(order))
	for _, n := range order {
		prs = append(prs, byNumber[n])
	}
	return prs, nil
}

// ParseISO parses the ISO-8601 forms datetime.fromisoformat accepts here, after
// the same "Z" -> "+00:00" rewrite rt_analyze.py applies. A naive timestamp is
// rejected: Python cannot subtract it from the aware mergedAt either.
//
// The result is truncated to microseconds because datetime is microsecond-
// precision and fromisoformat drops any further digits: keeping Go's nanoseconds
// would let a 7th fractional digit in --now shift ageDays by one, which moves a
// PR across a recency-weight boundary and reorders the reviewer ranking.
func ParseISO(s string) (time.Time, error) {
	norm := strings.ReplaceAll(s, "Z", "+00:00")
	for _, layout := range []string{
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02T15:04Z07:00",
		"2006-01-02 15:04Z07:00",
	} {
		if t, err := time.Parse(layout, norm); err == nil {
			return t.Truncate(time.Microsecond), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid isoformat timestamp: %q", s)
}

// AgeDays is (now - merged).days: whole days, floored toward minus infinity the
// way timedelta normalization does. It is the input to the recency weighting
// here and in rtcommits, which weights commits on the same boundaries.
func AgeDays(now, merged time.Time) int {
	const secPerDay = 86400
	sec := now.Unix() - merged.Unix()
	days, rem := sec/secPerDay, sec%secPerDay
	if rem < 0 {
		days--
		rem += secPerDay
	}
	if rem*int64(time.Second)+int64(now.Nanosecond())-int64(merged.Nanosecond()) < 0 {
		days--
	}
	return int(days)
}

type reviewerStat struct {
	login    string
	raw      int
	weighted float64
}

type recentItem struct {
	mergedAt string
	number   int
	title    string
}

type areaState struct {
	count    int
	recent   []recentItem
	revOrder []*reviewerStat
	revIndex map[string]*reviewerStat
}

type zeroEntry struct {
	number int
	paths  []string
}

type overEntry struct {
	number int
	paths  []string
}

type unknownIdentity struct {
	login string
	areas map[string]bool
	raw   int
}

// headLimit is Python's list[:n] bound, negative n included.
func headLimit(n, length int) int {
	if n < 0 {
		n += length
	}
	if n < 0 {
		return 0
	}
	if n > length {
		return length
	}
	return n
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedNumbers(set map[int]bool) []int {
	out := make([]int, 0, len(set))
	for n := range set {
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}

type tally struct {
	areas       map[string]*areaState
	authorsLast map[string]string
	zeroMatch   []zeroEntry
	overmatch   []overEntry
}

// tallyPRs walks the PRs once, bucketing each by changed path and accruing
// recency-weighted review credit: 1.0 within 6 months of now, 0.5 within 12,
// 0.25 beyond. Bots and self-reviews never count, and credit accrues per review
// event rather than per reviewer.
func tallyPRs(prs []*PR, now time.Time) (*tally, error) {
	t := &tally{
		areas:       make(map[string]*areaState, len(allAreas)),
		authorsLast: map[string]string{},
	}
	for _, a := range allAreas {
		t.areas[a] = &areaState{revIndex: map[string]*reviewerStat{}}
	}

	for _, pr := range prs {
		merged, err := ParseISO(pr.MergedAt)
		if err != nil {
			return nil, fmt.Errorf("PR #%d mergedAt: %w", pr.Number, err)
		}
		w := 0.25
		switch age := AgeDays(now, merged); {
		case age <= 182:
			w = 1.0
		case age <= 365:
			w = 0.5
		}

		author := ""
		if pr.Author != nil {
			author = pr.Author.Login
		}
		if author != "" {
			ym := pr.MergedAt
			if len(ym) > 7 {
				ym = ym[:7]
			}
			if prev, ok := t.authorsLast[author]; !ok || ym > prev {
				t.authorsLast[author] = ym
			}
		}

		var paths []string
		prAreas := map[string]bool{}
		for _, n := range pr.Files.Nodes {
			if n == nil {
				continue
			}
			paths = append(paths, n.Path)
			for a := range AreasFor(n.Path) {
				prAreas[a] = true
			}
		}
		if len(prAreas) == 0 {
			t.zeroMatch = append(t.zeroMatch, zeroEntry{pr.Number, paths})
			continue
		}
		if len(prAreas) >= 6 {
			t.overmatch = append(t.overmatch, overEntry{pr.Number, paths})
		}

		var events []string
		for _, r := range pr.Reviews.Nodes {
			if r == nil || r.Author == nil {
				continue
			}
			login := r.Author.Login
			if login == "" || login == author || IsBot(login) {
				continue
			}
			events = append(events, login)
		}

		for _, a := range allAreas {
			if !prAreas[a] {
				continue
			}
			as := t.areas[a]
			as.count++
			as.recent = append(as.recent, recentItem{pr.MergedAt, pr.Number, pr.Title})
			for _, login := range events {
				stat, ok := as.revIndex[login]
				if !ok {
					stat = &reviewerStat{login: login}
					as.revIndex[login] = stat
					as.revOrder = append(as.revOrder, stat)
				}
				stat.raw++
				stat.weighted += w
			}
		}
	}
	return t, nil
}

var zeroGroups = []struct {
	label string
	pred  func([]string) bool
}{
	{"deploy/examples generic manifests (deliberately unbucketed)", func(paths []string) bool {
		return hasPathPrefix(paths, "deploy/examples/")
	}},
	{"repo meta files (deliberately unbucketed)", func(paths []string) bool {
		for _, p := range paths {
			if repoMeta[p] || strings.HasPrefix(p, ".docs/") {
				return true
			}
		}
		return false
	}},
	{"generated artifacts (deliberately unbucketed)", func(paths []string) bool {
		for _, p := range paths {
			if codegen[p] {
				return true
			}
		}
		return false
	}},
}

func (t *tally) bucketFlags() []Flag {
	var flags []Flag
	grouped := map[int]bool{}
	for _, g := range zeroGroups {
		matched := map[int]bool{}
		for _, z := range t.zeroMatch {
			if g.pred(z.paths) {
				matched[z.number] = true
			}
		}
		nums := sortedNumbers(matched)
		if len(nums) == 0 {
			continue
		}
		for _, n := range nums {
			grouped[n] = true
		}
		flags = append(flags, Flag{
			Type:     "bucket-ambiguity",
			Item:     fmt.Sprintf("%d PRs: no area rule matches — %s", len(nums), g.label),
			Evidence: "PR numbers: " + pyReprInts(nums),
			Question: "kb v3 leaves this class unbucketed on purpose — confirm the gap is still intentional, or name the area these should count toward.",
		})
	}

	leftoverSet := map[int]bool{}
	for _, z := range t.zeroMatch {
		if !grouped[z.number] {
			leftoverSet[z.number] = true
		}
	}
	if leftover := sortedNumbers(leftoverSet); len(leftover) > 0 {
		flags = append(flags, Flag{
			Type:     "bucket-ambiguity",
			Item:     fmt.Sprintf("%d PRs: no area rule matches — ungrouped/misc", len(leftover)),
			Evidence: fmt.Sprintf("PR numbers: %s; sample paths: %s", pyReprInts(leftover), t.samplePaths(leftoverSet)),
			Question: "Not in any deliberate unbucketed class — what area (if any) should each count toward, or is a taxonomy/classifier fix needed?",
		})
	}

	apis, cross := map[int]bool{}, map[int]bool{}
	for _, o := range t.overmatch {
		if hasPathPrefix(o.paths, "pkg/apis/") {
			apis[o.number] = true
		}
	}
	for _, o := range t.overmatch {
		if !apis[o.number] {
			cross[o.number] = true
		}
	}
	if nums := sortedNumbers(apis); len(nums) > 0 {
		flags = append(flags, Flag{
			Type:     "bucket-ambiguity",
			Item:     fmt.Sprintf("%d PRs match >=6 areas — pkg/apis/** type changes fanning across operator code, hand-written docs and tests", len(nums)),
			Evidence: "PR numbers: " + pyReprInts(nums),
			Question: "Likely the legitimate blast radius of an API change, not a classifier bug — confirm these should still count toward each touched area's reviewer stats.",
		})
	}
	if nums := sortedNumbers(cross); len(nums) > 0 {
		flags = append(flags, Flag{
			Type:     "bucket-ambiguity",
			Item:     fmt.Sprintf("%d PRs match >=6 areas — cross-cutting sweep, no pkg/apis change", len(nums)),
			Evidence: "PR numbers: " + pyReprInts(nums),
			Question: "Confirm genuine cross-cutting refactors (shared helpers/test framework) rather than the classifier over-matching unrelated paths.",
		})
	}
	return flags
}

// samplePaths shows up to 8 paths per unbucketed PR, capped at 1500 characters
// so one pathological PR cannot bury the rest of the evidence.
func (t *tally) samplePaths(want map[int]bool) string {
	sample := Obj{}
	for _, z := range t.zeroMatch {
		if !want[z.number] {
			continue
		}
		head := z.paths
		if len(head) > 8 {
			head = head[:8]
		}
		sample = append(sample, Member{Key: strconv.Itoa(z.number), Val: toAnySlice(head)})
	}
	encoded := MarshalCompact(sample)
	if len(encoded) > 1500 {
		encoded = encoded[:1500]
	}
	return encoded
}

// provenanceFlags turns the fetch layer's own record of its limits into flags.
// Truncations are scoped to PRs that survived into the counted set: a warning
// about a PR nothing counted is noise.
func provenanceFlags(st *State, counted map[int]bool) []Flag {
	var flags []Flag
	for _, tr := range st.Truncations {
		if !counted[tr.Number] {
			continue
		}
		mergedAt := "None"
		if tr.MergedAt != nil {
			mergedAt = *tr.MergedAt
		}
		flags = append(flags, Flag{
			Type:     "truncation",
			Item:     fmt.Sprintf("PR #%d", tr.Number),
			Evidence: fmt.Sprintf("%s pageInfo.hasNextPage=true (mergedAt=%s)", tr.Kind, mergedAt),
			Question: fmt.Sprintf("PR has more %s than fetched (100 files / 30 reviews cap) — counts for it may be incomplete.", tr.Kind),
		})
	}
	for _, e := range st.Errors {
		flags = append(flags, Flag{
			Type:     "spec-boundary",
			Item:     "fetch pipeline",
			Evidence: e,
			Question: "Did this error drop or duplicate any PRs in the counted set?",
		})
	}
	stop := ""
	if st.StopReason != nil {
		stop = *st.StopReason
	}
	if !hasAnyPrefix(stop, "reached", "full page entirely", "no more pages") {
		flags = append(flags, Flag{
			Type: "spec-boundary",
			Item: "stop condition",
			Evidence: fmt.Sprintf("pagination stopped due to: %s (pages_fetched=%s)",
				pyReprString(stop), pyStrNumber(st.PagesFetched)),
			Question: "Neither the cap nor a clean window/history boundary — is the dataset still valid for the stated window?",
		})
	}
	return flags
}

type areaReport struct {
	obj     Obj
	flags   []Flag
	unknown []*unknownIdentity
	topLine map[string]string
}

func (t *tally) report(opts Options) areaReport {
	rep := areaReport{obj: Obj{}, topLine: map[string]string{}}
	index := map[string]*unknownIdentity{}

	for _, a := range allAreas {
		as := t.areas[a]
		top := rankReviewers(as.revOrder, opts.Top)
		if as.count > 0 && len(top) == 0 {
			rep.flags = append(rep.flags, Flag{
				Type:     "coverage-gap",
				Item:     a,
				Evidence: fmt.Sprintf("%d PR(s) bucketed, 0 non-bot/non-self reviews recorded", as.count),
				Question: "Genuinely under-reviewed area, or is its path rule missing real hits?",
			})
		}

		reviewers := make([]any, 0, len(top))
		var line []string
		for i, rev := range top {
			if !opts.Roster[strings.ToLower(rev.login)] {
				acc, ok := index[rev.login]
				if !ok {
					acc = &unknownIdentity{login: rev.login, areas: map[string]bool{}}
					index[rev.login] = acc
					rep.unknown = append(rep.unknown, acc)
				}
				acc.areas[a] = true
				acc.raw += rev.raw
			}
			weighted := round2(rev.weighted)
			reviewers = append(reviewers, Obj{
				{Key: "login", Val: rev.login},
				{Key: "weighted_reviews", Val: weighted},
				{Key: "raw", Val: rev.raw},
			})
			if i < 3 {
				line = append(line, fmt.Sprintf("%s(%s/%d)", rev.login, pyFloat(weighted), rev.raw))
			}
		}
		rep.topLine[a] = strings.Join(line, ", ")

		recent := topRecent(as.recent)
		items := make([]any, 0, len(recent))
		for _, it := range recent {
			items = append(items, Obj{
				{Key: "number", Val: it.number},
				{Key: "title", Val: it.title},
			})
		}

		rep.obj = append(rep.obj, Member{Key: a, Val: Obj{
			{Key: "reviewers", Val: reviewers},
			{Key: "recent_items", Val: items},
		}})
	}
	return rep
}

// rankReviewers orders an area's reviewers by weight, then raw count, then
// lowercased login — and where even that ties, by the order the reviewer was
// first seen, which is what Python's stable sort fell back on.
//
// Known, accepted divergence: ToLower is simple case mapping where str.lower()
// is full, so U+0130 folds to "i" here but to "i" plus a combining dot in
// Python, which can break a tie the other way. GitHub logins are ASCII
// alphanumeric plus hyphen, so no real login reaches it.
func rankReviewers(revs []*reviewerStat, top int) []*reviewerStat {
	out := make([]*reviewerStat, len(revs))
	copy(out, revs)
	sort.SliceStable(out, func(i, j int) bool {
		x, y := out[i], out[j]
		if x.weighted != y.weighted {
			return x.weighted > y.weighted
		}
		if x.raw != y.raw {
			return x.raw > y.raw
		}
		return strings.ToLower(x.login) < strings.ToLower(y.login)
	})
	return out[:headLimit(top, len(out))]
}

func topRecent(items []recentItem) []recentItem {
	out := make([]recentItem, len(items))
	copy(out, items)
	sort.SliceStable(out, func(i, j int) bool {
		x, y := out[i], out[j]
		if x.mergedAt != y.mergedAt {
			return x.mergedAt > y.mergedAt
		}
		if x.number != y.number {
			return x.number > y.number
		}
		return x.title > y.title
	})
	if len(out) > 5 {
		out = out[:5]
	}
	return out
}

// identityFlags ranks unknown reviewers by review volume, keeping first-seen
// order for equal volumes.
func identityFlags(unknown []*unknownIdentity) []Flag {
	ranked := make([]*unknownIdentity, len(unknown))
	copy(ranked, unknown)
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].raw > ranked[j].raw })

	flags := make([]Flag, 0, len(ranked))
	for _, acc := range ranked {
		flags = append(flags, Flag{
			Type: "identity-unknown",
			Item: acc.login,
			Evidence: fmt.Sprintf("raw_reviews_total=%d across areas=%s",
				acc.raw, pyReprStrings(sortedKeys(acc.areas))),
			Question: "Not in the CODE-OWNERS roster and not an obvious bot — who is this / a legitimate community reviewer?",
		})
	}
	return flags
}

// Analyze buckets prs into the area taxonomy and builds the miner contract.
func Analyze(prs []*PR, st *State, opts Options) (*Result, error) {
	t, err := tallyPRs(prs, opts.Now)
	if err != nil {
		return nil, err
	}
	counted := make(map[int]bool, len(prs))
	for _, pr := range prs {
		counted[pr.Number] = true
	}

	rep := t.report(opts)
	flags := t.bucketFlags()
	flags = append(flags, provenanceFlags(st, counted)...)
	flags = append(flags, rep.flags...)
	flags = append(flags, identityFlags(rep.unknown)...)

	flagsArr := make([]any, 0, len(flags))
	for _, f := range flags {
		flagsArr = append(flagsArr, Obj{
			{Key: "type", Val: f.Type},
			{Key: "item", Val: f.Item},
			{Key: "evidence", Val: f.Evidence},
			{Key: "question", Val: f.Question},
		})
	}

	authors := Obj{}
	for _, login := range sortedKeys(t.authorsLast) {
		authors = append(authors, Member{Key: login, Val: t.authorsLast[login]})
	}

	doc := Obj{
		{Key: "data", Val: Obj{
			{Key: "generated_from", Val: fmt.Sprintf("%s merged PRs back to %s",
				pyStrNumber(st.Counted), oldestDay(st))},
			{Key: "areas", Val: rep.obj},
			{Key: "authors_last_merged", Val: authors},
		}},
		{Key: "flags", Val: flagsArr},
	}

	summary := []string{fmt.Sprintf("PRs=%d zero_match=%d overmatch=%d flags=%s -> %s",
		len(prs), len(t.zeroMatch), len(t.overmatch), pyReprCounts(flagCounts(flags)), opts.OutPath)}
	for _, a := range summaryAreas {
		summary = append(summary, fmt.Sprintf("  %s: %s", a, rep.topLine[a]))
	}
	return &Result{Doc: doc, Summary: summary}, nil
}

func oldestDay(st *State) string {
	oldest := ""
	if st.OldestMergedAt != nil {
		oldest = *st.OldestMergedAt
	}
	if day := strings.SplitN(oldest, "T", 2)[0]; day != "" {
		return day
	}
	return "unknown"
}

func flagCounts(flags []Flag) []countedType {
	var counts []countedType
	at := map[string]int{}
	for _, f := range flags {
		if i, ok := at[f.Type]; ok {
			counts[i].n++
			continue
		}
		at[f.Type] = len(counts)
		counts = append(counts, countedType{name: f.Type, n: 1})
	}
	return counts
}

func hasPathPrefix(paths []string, prefix string) bool {
	for _, p := range paths {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

func toAnySlice(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}
