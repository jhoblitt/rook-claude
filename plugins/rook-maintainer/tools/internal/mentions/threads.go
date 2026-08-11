package mentions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/ghx"
)

// chunk is how many issues one aliased GraphQL query asks for. Larger chunks
// trip GitHub's query-cost limit.
const chunk = 75

type Thread struct {
	Number   int      `json:"number"`
	Body     string   `json:"body"`
	Comments Comments `json:"comments"`
}

type Comments struct {
	TotalCount int       `json:"totalCount"`
	PageInfo   PageInfo  `json:"pageInfo"`
	Nodes      []Comment `json:"nodes"`
}

// PageInfo keeps endCursor nullable so threads.json round-trips through this
// tool byte-for-byte: the API sends null on the last page.
type PageInfo struct {
	HasNextPage bool    `json:"hasNextPage"`
	EndCursor   *string `json:"endCursor"`
}

func (p PageInfo) Cursor() string {
	if p.EndCursor == nil {
		return ""
	}
	return *p.EndCursor
}

type Comment struct {
	Body string `json:"body"`
}

// Docs returns the thread's comment-documents: issue body first, then every
// comment, each of which GitHub renders — and so this tool strips — alone.
func (t *Thread) Docs() []string {
	docs := make([]string, 0, len(t.Comments.Nodes)+1)
	docs = append(docs, t.Body)
	for _, c := range t.Comments.Nodes {
		docs = append(docs, c.Body)
	}
	return docs
}

// ThreadSet is threads.json: GraphQL aliases to threads, in fetch order. A nil
// thread is an alias the API returned null for.
type ThreadSet struct {
	Keys    []string
	Threads map[string]*Thread
}

func (ts *ThreadSet) Set(key string, t *Thread) {
	if ts.Threads == nil {
		ts.Threads = make(map[string]*Thread)
	}
	if _, ok := ts.Threads[key]; !ok {
		ts.Keys = append(ts.Keys, key)
	}
	ts.Threads[key] = t
}

func (ts *ThreadSet) UnmarshalJSON(data []byte) error {
	ts.Keys = nil
	ts.Threads = make(map[string]*Thread)
	return decodeObject(data, func(key string, dec *json.Decoder) error {
		var t *Thread
		if err := dec.Decode(&t); err != nil {
			return err
		}
		ts.Set(key, t)
		return nil
	})
}

func (ts ThreadSet) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range ts.Keys {
		if i > 0 {
			b.WriteByte(',')
		}
		kb, err := marshalNoEscape(k)
		if err != nil {
			return nil, err
		}
		b.Write(kb)
		b.WriteByte(':')
		vb, err := marshalNoEscape(ts.Threads[k])
		if err != nil {
			return nil, err
		}
		b.Write(vb)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// marshalNoEscape keeps issue bodies legible: the default encoder turns every
// `<`, `>` and `&` in them into a \u escape, and threads.json is a file a
// maintainer greps.
func marshalNoEscape(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(b.Bytes(), []byte("\n")), nil
}

func LoadThreads(path string) (*ThreadSet, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ts := &ThreadSet{}
	if err := json.Unmarshal(b, ts); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return ts, nil
}

func SaveThreads(path string, ts *ThreadSet) error {
	b, err := json.Marshal(ts)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

type repositoryResponse struct {
	Repository ThreadSet `json:"repository"`
}

type pageResponse struct {
	Repository struct {
		Issue struct {
			Comments Comments `json:"comments"`
		} `json:"issue"`
	} `json:"repository"`
}

// FetchThreads pulls each issue's body and ALL of its comments. Mentions live
// as often in comment 300 as in the body, so the first page is followed per
// issue rather than truncated at 100.
func FetchThreads(ctx context.Context, repo string, numbers []int) (*ThreadSet, error) {
	owner, name, ok := strings.Cut(repo, "/")
	if !ok {
		return nil, fmt.Errorf("repo %q is not owner/name", repo)
	}
	merged := &ThreadSet{}
	for start := 0; start < len(numbers); start += chunk {
		end := min(start+chunk, len(numbers))
		var fields []string
		for _, n := range numbers[start:end] {
			fields = append(fields, fmt.Sprintf(
				"i%d: issue(number: %d) { number body comments(first: 100) "+
					"{ totalCount pageInfo { hasNextPage endCursor } nodes { body } } }", n, n))
		}
		query := fmt.Sprintf("query { repository(owner: %q, name: %q) { %s } }",
			owner, name, strings.Join(fields, " "))
		var resp repositoryResponse
		if err := ghx.GraphQL(ctx, query, &resp); err != nil {
			return nil, err
		}
		for _, k := range resp.Repository.Keys {
			merged.Set(k, resp.Repository.Threads[k])
		}
	}

	for _, k := range merged.Keys {
		t := merged.Threads[k]
		if t == nil {
			continue
		}
		cursor := t.Comments.PageInfo.Cursor()
		for t.Comments.PageInfo.HasNextPage {
			query := fmt.Sprintf("query { repository(owner: %q, name: %q) { "+
				"issue(number: %d) { comments(first: 100, after: %q) "+
				"{ pageInfo { hasNextPage endCursor } nodes { body } } } } }",
				owner, name, t.Number, cursor)
			var resp pageResponse
			if err := ghx.GraphQL(ctx, query, &resp); err != nil {
				return nil, err
			}
			page := resp.Repository.Issue.Comments
			t.Comments.Nodes = append(t.Comments.Nodes, page.Nodes...)
			t.Comments.PageInfo = page.PageInfo
			cursor = page.PageInfo.Cursor()
		}
	}
	return merged, nil
}
