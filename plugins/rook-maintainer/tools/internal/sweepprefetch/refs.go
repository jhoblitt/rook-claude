package sweepprefetch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
)

const refsChunk = 75

type RefsResult struct {
	Path string
	Refs int
	New  int
}

func (r RefsResult) String() string {
	return fmt.Sprintf("%d refs, %d newly classified -> %s", r.Refs, r.New, r.Path)
}

type batchItem struct {
	Xlinks []json.RawMessage `json:"xlinks"`
	Dups   []json.RawMessage `json:"dups"`
}

// ClassifyRefs resolves the xlink/dup numbers the triagers recorded in
// batch-*.json to Issue vs PullRequest and merges them into refs-types.json.
// A reference's type is never guessed from context: issueOrPullRequest is the
// only thing that knows, and a dashboard that links an issue as a PR is wrong
// in a way nobody notices.
func (c *Client) ClassifyRefs(ctx context.Context, sweepDir string) (RefsResult, error) {
	res := RefsResult{Path: filepath.Join(sweepDir, "refs-types.json")}
	numbers, err := refNumbers(sweepDir)
	if err != nil {
		return res, err
	}
	res.Refs = len(numbers)

	types, err := loadRefTypes(res.Path)
	if err != nil {
		return res, err
	}
	var todo []int
	for _, n := range numbers {
		if !types.Has(strconv.Itoa(n)) {
			todo = append(todo, n)
		}
	}
	res.New = len(todo)

	for start := 0; start < len(todo); start += refsChunk {
		batch := todo[start:min(start+refsChunk, len(todo))]
		var aliases strings.Builder
		for i, n := range batch {
			if i > 0 {
				aliases.WriteString(" ")
			}
			fmt.Fprintf(&aliases, "n%d: issueOrPullRequest(number: %d) { __typename }", n, n)
		}
		q := fmt.Sprintf("query { repository(owner: %s, name: %s) { %s } }",
			quote(c.Owner), quote(c.Name), aliases.String())
		var wrap repoWrapper
		if err := c.query(ctx, q, &wrap); err != nil {
			return res, err
		}
		for _, n := range batch {
			raw, ok := wrap.Repository[fmt.Sprintf("n%d", n)]
			if !ok || isNull(raw) {
				continue
			}
			var node struct {
				Typename string `json:"__typename"`
			}
			if err := json.Unmarshal(raw, &node); err != nil {
				return res, fmt.Errorf("decoding ref %d: %w", n, err)
			}
			types.Set(strconv.Itoa(n), node.Typename)
		}
	}

	data, err := Encode(types)
	if err != nil {
		return res, err
	}
	if err := os.WriteFile(res.Path, data, 0o644); err != nil {
		return res, err
	}
	return res, nil
}

func refNumbers(sweepDir string) ([]int, error) {
	batches, err := filepath.Glob(filepath.Join(sweepDir, "batch-*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(batches)
	var numbers []int
	seen := map[int]bool{}
	for _, path := range batches {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var items []batchItem
		if err := json.Unmarshal(data, &items); err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		for _, it := range items {
			for _, raw := range slices.Concat(it.Xlinks, it.Dups) {
				n, ok, err := refNumber(raw)
				if err != nil {
					return nil, fmt.Errorf("reading %s: %w", path, err)
				}
				if ok && !seen[n] {
					seen[n] = true
					numbers = append(numbers, n)
				}
			}
		}
	}
	slices.Sort(numbers)
	return numbers, nil
}

// refNumber reads one xlink/dup entry. Anything that is not an object with a
// number field is skipped: triagers also write bare prose refs there.
func refNumber(raw json.RawMessage) (int, bool, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return 0, false, nil
	}
	field, ok := obj["number"]
	if !ok {
		return 0, false, nil
	}
	var num json.Number
	if err := json.Unmarshal(field, &num); err == nil {
		return toInt(num)
	}
	var s string
	if err := json.Unmarshal(field, &s); err != nil {
		return 0, false, fmt.Errorf("ref number %s is not a number", field)
	}
	return toInt(json.Number(strings.TrimSpace(s)))
}

func toInt(num json.Number) (int, bool, error) {
	if n, err := num.Int64(); err == nil {
		return int(n), true, nil
	}
	f, err := num.Float64()
	if err != nil {
		return 0, false, fmt.Errorf("ref number %q is not a number", num)
	}
	return int(math.Trunc(f)), true, nil
}

func loadRefTypes(path string) (*Object, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return NewObject(), nil
	}
	if err != nil {
		return nil, err
	}
	types, err := DecodeObject(data)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return types, nil
}
