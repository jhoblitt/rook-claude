// Package sweepprefetch takes the phase-0 metadata snapshot a rook-triage or
// rook-code-review sweep runs on: one batched GraphQL pass per corpus, written
// to <sweep-dir>/snapshot.json.
//
// Every triager, reviewer and dashboard generator reads that snapshot instead
// of fetching per-item metadata itself — one fetch, one consistent
// point-in-time view, ~100 fewer per-agent gh calls per sweep. Two payloads
// only this pass can supply:
//
//   - PR files (changed paths), which is what lets a sweep orchestrator route
//     reference files per PR without a per-PR fetch; `gh pr list` cannot
//     return them at any --json setting.
//   - a summarized statusCheckRollup, the ONLY source dashboards may use for
//     CI cells (deterministic passing/total, never parsed from agent prose).
//
// The written shape is a contract with consumers outside this module, so
// field names, nesting and null-vs-absent are all load-bearing.
package sweepprefetch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/ghx"
)

const ctxNode = `
__typename
... on CheckRun { name status conclusion
  checkSuite { databaseId createdAt workflowRun { workflow { name } } } }
... on StatusContext { context state }
`

const prFields = `
number title state isDraft updatedAt createdAt
author { login } authorAssociation baseRefName mergeable reviewDecision
additions deletions changedFiles
labels(first: 20) { nodes { name } }
assignees(first: 10) { nodes { login } }
files(first: 100) { pageInfo { hasNextPage endCursor } nodes { path } }
latestReviews(first: 20) { nodes { author { login } state } }
reviewRequests(first: 20) { nodes { requestedReviewer {
  ... on User { login } ... on Team { name } } } }
commits(last: 1) { nodes { commit { statusCheckRollup {
  state contexts(first: 100) { pageInfo { hasNextPage endCursor } nodes { ` + ctxNode + ` } } } } } }
`

const issueFields = `
number title state updatedAt createdAt
author { login }
labels(first: 20) { nodes { name } }
assignees(first: 10) { nodes { login } }
comments { totalCount }
`

// GraphQLFunc runs one query and unmarshals its data object into out.
type GraphQLFunc func(ctx context.Context, query string, out any) error

type Client struct {
	Repo  string
	Owner string
	Name  string

	// GraphQL and Now are injection points for tests; zero values use gh and
	// the wall clock.
	GraphQL GraphQLFunc
	Now     func() time.Time
}

func NewClient(repo string) (*Client, error) {
	owner, name, ok := strings.Cut(repo, "/")
	if !ok || owner == "" || name == "" {
		return nil, fmt.Errorf("repo %q is not owner/name", repo)
	}
	return &Client{Repo: repo, Owner: owner, Name: name}, nil
}

func (c *Client) query(ctx context.Context, q string, out any) error {
	if c.GraphQL != nil {
		return c.GraphQL(ctx, q, out)
	}
	return ghx.GraphQL(ctx, q, out)
}

func (c *Client) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

type SnapshotOptions struct {
	SweepDir string
	// Kind is "prs" or "issues".
	Kind string
	// ByNumber fetches exactly Numbers — including closed and merged items,
	// which is how a dashboard for a past sweep gets regenerated — instead of
	// enumerating every open item.
	ByNumber bool
	Numbers  []int
}

type SnapshotResult struct {
	Path      string
	Kind      string
	Count     int
	Truncated []string
}

func (r SnapshotResult) String() string {
	s := fmt.Sprintf("%d %s -> %s", r.Count, r.Kind, r.Path)
	if len(r.Truncated) == 0 {
		return s
	}
	quoted := make([]string, len(r.Truncated))
	for i, n := range r.Truncated {
		quoted[i] = "'" + n + "'"
	}
	return s + fmt.Sprintf(" (truncated fields on: [%s])", strings.Join(quoted, ", "))
}

type snapshotFile struct {
	FetchedAt string  `json:"fetched_at"`
	Repo      string  `json:"repo"`
	Kind      string  `json:"kind"`
	Items     *Object `json:"items"`
}

func (c *Client) Snapshot(ctx context.Context, opts SnapshotOptions) (SnapshotResult, error) {
	res := SnapshotResult{
		Path: filepath.Join(opts.SweepDir, "snapshot.json"),
		Kind: opts.Kind,
	}
	items := NewObject()
	if opts.Kind == "prs" {
		prs, err := c.snapshotPRs(ctx, opts)
		if err != nil {
			return res, err
		}
		for _, item := range prs {
			items.Set(strconv.Itoa(item.Number), item)
			if item.FilesTruncated || item.CI.Truncated {
				res.Truncated = append(res.Truncated, strconv.Itoa(item.Number))
			}
		}
	} else {
		nodes, err := c.fetchIssueNodes(ctx, opts)
		if err != nil {
			return res, err
		}
		for _, n := range nodes {
			item := shapeIssue(n)
			items.Set(strconv.Itoa(item.Number), item)
		}
	}

	out := snapshotFile{
		FetchedAt: c.now().UTC().Format("2006-01-02T15:04:05.000000-07:00"),
		Repo:      c.Repo,
		Kind:      opts.Kind,
		Items:     items,
	}
	data, err := Encode(out)
	if err != nil {
		return res, err
	}
	if err := os.MkdirAll(opts.SweepDir, 0o755); err != nil {
		return res, err
	}
	if err := os.WriteFile(res.Path, data, 0o644); err != nil {
		return res, err
	}
	res.Count = items.Len()
	return res, nil
}

func (c *Client) snapshotPRs(ctx context.Context, opts SnapshotOptions) ([]*PRItem, error) {
	var nodes []prNode
	var err error
	if opts.ByNumber {
		nodes, err = fetchNumbers[prNode](ctx, c, "pullRequest", prFields, 15, opts.Numbers)
	} else {
		nodes, err = fetchOpen[prNode](ctx, c, "pullRequests", prFields, 25)
	}
	if err != nil {
		return nil, err
	}
	items := dedupePRs(nodes)
	// The batched query caps contexts and files at one page each; anything
	// that hit the cap is re-fetched on its own so the counts stay exact.
	for _, item := range items {
		if item.CI.Truncated {
			state, ctxNodes, err := c.paginateContexts(ctx, item.Number)
			if err != nil {
				return nil, err
			}
			item.CI = ClassifyContexts(state, ctxNodes, false)
		}
		if item.FilesTruncated {
			files, err := c.paginateFiles(ctx, item.Number)
			if err != nil {
				return nil, err
			}
			item.Files = files
			item.FilesTruncated = false
		}
	}
	return items, nil
}

// dedupePRs collapses repeated PR numbers exactly as the items map they end up
// in would: the last node wins, at the position the number was first seen.
//
// A cursor walk can hand back the same PR on two pages when the corpus shifts
// under it. Deduping has to precede the per-item re-fetch below: run that
// re-fetch per node instead and the duplicate burns a second gh call, and the
// copy that survives carries CI state and files from that later fetch rather
// than from the single pass every other item gets.
func dedupePRs(nodes []prNode) []*PRItem {
	items := make([]*PRItem, 0, len(nodes))
	at := make(map[int]int, len(nodes))
	for _, n := range nodes {
		item := shapePR(n)
		if i, seen := at[item.Number]; seen {
			items[i] = item
			continue
		}
		at[item.Number] = len(items)
		items = append(items, item)
	}
	return items
}

func (c *Client) fetchIssueNodes(ctx context.Context, opts SnapshotOptions) ([]issueNode, error) {
	if opts.ByNumber {
		return fetchNumbers[issueNode](ctx, c, "issue", issueFields, 75, opts.Numbers)
	}
	return fetchOpen[issueNode](ctx, c, "issues", issueFields, 100)
}

type repoWrapper struct {
	Repository map[string]json.RawMessage `json:"repository"`
}

type nodeBlock[T any] struct {
	PageInfo pageInfo `json:"pageInfo"`
	Nodes    []T      `json:"nodes"`
}

func fetchOpen[T any](ctx context.Context, c *Client, field, inner string, page int) ([]T, error) {
	var nodes []T
	cursor := ""
	for {
		q := fmt.Sprintf("query { repository(owner: %s, name: %s) { "+
			"%s(states: OPEN, first: %d%s, orderBy: {field: CREATED_AT, direction: ASC}) { "+
			"pageInfo { hasNextPage endCursor } nodes { %s } } } }",
			quote(c.Owner), quote(c.Name), field, page, after(cursor), inner)
		raw, err := c.repoField(ctx, q, field)
		if err != nil {
			return nil, err
		}
		var block nodeBlock[T]
		if err := json.Unmarshal(raw, &block); err != nil {
			return nil, fmt.Errorf("decoding %s page: %w", field, err)
		}
		nodes = append(nodes, block.Nodes...)
		if !block.PageInfo.HasNextPage {
			return nodes, nil
		}
		cursor = block.PageInfo.EndCursor
	}
}

func fetchNumbers[T any](ctx context.Context, c *Client, field, inner string, chunk int, numbers []int) ([]T, error) {
	var nodes []T
	for start := 0; start < len(numbers); start += chunk {
		batch := numbers[start:min(start+chunk, len(numbers))]
		var aliases strings.Builder
		for i, n := range batch {
			if i > 0 {
				aliases.WriteString(" ")
			}
			fmt.Fprintf(&aliases, "n%d: %s(number: %d) { %s }", n, field, n, inner)
		}
		q := fmt.Sprintf("query { repository(owner: %s, name: %s) { %s } }",
			quote(c.Owner), quote(c.Name), aliases.String())
		var wrap repoWrapper
		if err := c.query(ctx, q, &wrap); err != nil {
			return nil, err
		}
		for _, n := range batch {
			raw, ok := wrap.Repository[fmt.Sprintf("n%d", n)]
			if !ok || isNull(raw) {
				continue
			}
			var node T
			if err := json.Unmarshal(raw, &node); err != nil {
				return nil, fmt.Errorf("decoding %s %d: %w", field, n, err)
			}
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
}

func (c *Client) paginateContexts(ctx context.Context, number int) (*string, []ContextNode, error) {
	var nodes []ContextNode
	cursor := ""
	for {
		q := fmt.Sprintf("query { repository(owner: %s, name: %s) { "+
			"pullRequest(number: %d) { commits(last: 1) { nodes { commit { "+
			"statusCheckRollup { state contexts(first: 100%s) { "+
			"pageInfo { hasNextPage endCursor } nodes { %s } } } } } } } } }",
			quote(c.Owner), quote(c.Name), number, after(cursor), ctxNode)
		var wrap struct {
			Repository *struct {
				PullRequest *struct {
					Commits struct {
						Nodes []struct {
							Commit struct {
								StatusCheckRollup *struct {
									State    *string                `json:"state"`
									Contexts nodeBlock[ContextNode] `json:"contexts"`
								} `json:"statusCheckRollup"`
							} `json:"commit"`
						} `json:"nodes"`
					} `json:"commits"`
				} `json:"pullRequest"`
			} `json:"repository"`
		}
		if err := c.query(ctx, q, &wrap); err != nil {
			return nil, nil, err
		}
		if wrap.Repository == nil || wrap.Repository.PullRequest == nil {
			return nil, nil, fmt.Errorf("graphql response for pull request %d contexts carried no repository.pullRequest", number)
		}
		commits := wrap.Repository.PullRequest.Commits.Nodes
		if len(commits) == 0 || commits[0].Commit.StatusCheckRollup == nil {
			return nil, nil, fmt.Errorf("pull request %d has no status check rollup to paginate", number)
		}
		rollup := commits[0].Commit.StatusCheckRollup
		nodes = append(nodes, rollup.Contexts.Nodes...)
		if !rollup.Contexts.PageInfo.HasNextPage {
			return rollup.State, nodes, nil
		}
		cursor = rollup.Contexts.PageInfo.EndCursor
	}
}

func (c *Client) paginateFiles(ctx context.Context, number int) ([]string, error) {
	paths := []string{}
	cursor := ""
	for {
		q := fmt.Sprintf("query { repository(owner: %s, name: %s) { "+
			"pullRequest(number: %d) { files(first: 100%s) { "+
			"pageInfo { hasNextPage endCursor } nodes { path } } } } }",
			quote(c.Owner), quote(c.Name), number, after(cursor))
		var wrap struct {
			Repository *struct {
				PullRequest *struct {
					Files nodeBlock[struct {
						Path string `json:"path"`
					}] `json:"files"`
				} `json:"pullRequest"`
			} `json:"repository"`
		}
		if err := c.query(ctx, q, &wrap); err != nil {
			return nil, err
		}
		// Without this the null unmarshals to an empty page that reads as
		// "pagination finished", and the caller replaces a real file list with
		// an empty one flagged complete.
		if wrap.Repository == nil || wrap.Repository.PullRequest == nil {
			return nil, fmt.Errorf("graphql response for pull request %d files carried no repository.pullRequest", number)
		}
		block := wrap.Repository.PullRequest.Files
		for _, f := range block.Nodes {
			paths = append(paths, f.Path)
		}
		if !block.PageInfo.HasNextPage {
			return paths, nil
		}
		cursor = block.PageInfo.EndCursor
	}
}

func (c *Client) repoField(ctx context.Context, q, field string) (json.RawMessage, error) {
	var wrap repoWrapper
	if err := c.query(ctx, q, &wrap); err != nil {
		return nil, err
	}
	raw, ok := wrap.Repository[field]
	if !ok || isNull(raw) {
		return nil, fmt.Errorf("graphql response carried no repository.%s", field)
	}
	return raw, nil
}

func after(cursor string) string {
	if cursor == "" {
		return ""
	}
	return ", after: " + quote(cursor)
}

// quote renders a GraphQL string literal; its escaping rules are JSON's.
func quote(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `""`
	}
	return string(b)
}

func isNull(raw json.RawMessage) bool {
	return string(bytes.TrimSpace(raw)) == "null"
}
