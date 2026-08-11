package mentions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type Options struct {
	SweepDir    string
	Numbers     string
	NumbersFile string
	Repo        string
	Refetch     bool
	CachePath   string
	Lookup      LoginResolver
}

type issueMentions struct {
	keys []string
	byID map[string][]string
}

func (m *issueMentions) set(id string, logins []string) {
	if _, ok := m.byID[id]; !ok {
		m.keys = append(m.keys, id)
	}
	m.byID[id] = logins
}

// Run mines <sweep-dir>/threads.json into <sweep-dir>/issues-mentions.json,
// printing a per-issue diff against the previous version and a one-line
// summary.
func Run(ctx context.Context, opt Options, out io.Writer) error {
	threadsPath := filepath.Join(opt.SweepDir, "threads.json")
	outPath := filepath.Join(opt.SweepDir, "issues-mentions.json")

	raw, err := loadOrFetch(ctx, opt, threadsPath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(opt.CachePath), 0o755); err != nil {
		return err
	}
	cache, err := LoadCache(opt.CachePath)
	if err != nil {
		return err
	}

	mentions := issueMentions{byID: make(map[string][]string)}
	for _, k := range raw.Keys {
		thread := raw.Threads[k]
		if thread == nil {
			continue
		}
		var kept []string
		seen := make(map[string]bool)
		for _, tok := range Candidates(thread.Docs()) {
			login, err := Resolve(ctx, tok, cache, opt.Lookup)
			if err != nil {
				return err
			}
			key := strings.ToLower(login)
			if login == "" || seen[key] {
				continue
			}
			seen[key] = true
			kept = append(kept, login)
		}
		if len(kept) > 0 {
			mentions.set(strconv.Itoa(thread.Number), kept)
		}
	}
	if err := cache.Save(opt.CachePath); err != nil {
		return err
	}

	old, err := loadMentions(outPath)
	if err != nil {
		return err
	}
	if err := printDiff(out, old, mentions.byID); err != nil {
		return err
	}

	body, err := indentedObject(mentions.keys, func(k string) any { return mentions.byID[k] })
	if err != nil {
		return err
	}
	if err := os.WriteFile(outPath, body, 0o644); err != nil {
		return err
	}

	unique := make(map[string]bool)
	for _, logins := range mentions.byID {
		for _, l := range logins {
			unique[strings.ToLower(l)] = true
		}
	}
	_, err = fmt.Fprintf(out, "issues w/ mentions: %d/%d; unique logins: %d; "+
		"unresolvable tokens ever seen: %d\n",
		len(mentions.byID), len(raw.Keys), len(unique), cache.Unresolvable())
	return err
}

func loadOrFetch(ctx context.Context, opt Options, threadsPath string) (*ThreadSet, error) {
	if !opt.Refetch {
		if _, err := os.Stat(threadsPath); err == nil {
			return LoadThreads(threadsPath)
		}
	}
	numbers, err := issueNumbers(opt)
	if err != nil {
		return nil, err
	}
	raw, err := FetchThreads(ctx, opt.Repo, numbers)
	if err != nil {
		return nil, err
	}
	if err := SaveThreads(threadsPath, raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func issueNumbers(opt Options) ([]int, error) {
	var fields []string
	switch {
	case opt.Numbers != "":
		fields = strings.Split(opt.Numbers, ",")
	case opt.NumbersFile != "":
		b, err := os.ReadFile(opt.NumbersFile)
		if err != nil {
			return nil, err
		}
		fields = strings.Split(string(b), "\n")
	default:
		return nil, errors.New("threads.json missing: pass --numbers or --numbers-file to fetch")
	}
	var numbers []int
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		n, err := strconv.Atoi(f)
		if err != nil {
			return nil, fmt.Errorf("issue number %q: %w", f, err)
		}
		numbers = append(numbers, n)
	}
	slices.Sort(numbers)
	return slices.Compact(numbers), nil
}

func loadMentions(path string) (map[string][]string, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string][]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	var m map[string][]string
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return m, nil
}

func printDiff(out io.Writer, old, current map[string][]string) error {
	ids := make([]string, 0, len(old)+len(current))
	for id := range old {
		ids = append(ids, id)
	}
	for id := range current {
		if _, ok := old[id]; !ok {
			ids = append(ids, id)
		}
	}
	numbers := make(map[string]int, len(ids))
	for _, id := range ids {
		n, err := strconv.Atoi(id)
		if err != nil {
			return fmt.Errorf("issue key %q is not a number", id)
		}
		numbers[id] = n
	}
	slices.SortFunc(ids, func(a, b string) int { return numbers[a] - numbers[b] })

	for _, id := range ids {
		dropped := missing(old[id], current[id])
		added := missing(current[id], old[id])
		if len(dropped) == 0 && len(added) == 0 {
			continue
		}
		parts := make([]string, 0, len(dropped)+len(added))
		for _, d := range dropped {
			parts = append(parts, "-"+d)
		}
		for _, a := range added {
			parts = append(parts, "+"+a)
		}
		if _, err := fmt.Fprintf(out, "#%s: %s\n", id, strings.Join(parts, " ")); err != nil {
			return err
		}
	}
	return nil
}

// missing returns the sorted members of a that b does not carry.
func missing(a, b []string) []string {
	have := make(map[string]bool, len(b))
	for _, s := range b {
		have[s] = true
	}
	var out []string
	seen := make(map[string]bool, len(a))
	for _, s := range a {
		if !have[s] && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	slices.Sort(out)
	return out
}
