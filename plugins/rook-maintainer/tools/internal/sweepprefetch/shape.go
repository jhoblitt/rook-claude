package sweepprefetch

import (
	"encoding/json"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/rtanalyze"
)

type login struct {
	Login string `json:"login"`
}

type pageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type prNode struct {
	Number            int     `json:"number"`
	Title             string  `json:"title"`
	State             string  `json:"state"`
	IsDraft           bool    `json:"isDraft"`
	UpdatedAt         string  `json:"updatedAt"`
	CreatedAt         string  `json:"createdAt"`
	Author            *login  `json:"author"`
	AuthorAssociation *string `json:"authorAssociation"`
	BaseRefName       *string `json:"baseRefName"`
	Mergeable         *string `json:"mergeable"`
	ReviewDecision    *string `json:"reviewDecision"`
	Additions         *int    `json:"additions"`
	Deletions         *int    `json:"deletions"`
	ChangedFiles      *int    `json:"changedFiles"`
	Labels            struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	Assignees struct {
		Nodes []login `json:"nodes"`
	} `json:"assignees"`
	Files struct {
		PageInfo pageInfo `json:"pageInfo"`
		Nodes    []struct {
			Path string `json:"path"`
		} `json:"nodes"`
	} `json:"files"`
	LatestReviews struct {
		Nodes []struct {
			Author *login `json:"author"`
			State  string `json:"state"`
		} `json:"nodes"`
	} `json:"latestReviews"`
	ReviewRequests struct {
		Nodes []struct {
			RequestedReviewer map[string]json.RawMessage `json:"requestedReviewer"`
		} `json:"nodes"`
	} `json:"reviewRequests"`
	Commits struct {
		Nodes []struct {
			Commit *struct {
				StatusCheckRollup *struct {
					State    *string `json:"state"`
					Contexts struct {
						PageInfo pageInfo      `json:"pageInfo"`
						Nodes    []ContextNode `json:"nodes"`
					} `json:"contexts"`
				} `json:"statusCheckRollup"`
			} `json:"commit"`
		} `json:"nodes"`
	} `json:"commits"`
}

type issueNode struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	State     string `json:"state"`
	UpdatedAt string `json:"updatedAt"`
	CreatedAt string `json:"createdAt"`
	Author    *login `json:"author"`
	Labels    struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	Assignees struct {
		Nodes []login `json:"nodes"`
	} `json:"assignees"`
	Comments struct {
		TotalCount int `json:"totalCount"`
	} `json:"comments"`
}

type Review struct {
	Login string `json:"login"`
	State string `json:"state"`
}

type Reviews struct {
	Latest []Review `json:"latest"`
	// A requested reviewer that is neither a User nor a Team resolves to no
	// name at all, and the entry stays as a null the caller has to expect.
	Requested []*string `json:"requested"`
}

// PRItem is one entry of snapshot.json's items map. Field order and names are
// a contract with the dashboard generators.
type PRItem struct {
	Number            int      `json:"number"`
	Title             string   `json:"title"`
	State             string   `json:"state"`
	IsDraft           bool     `json:"isDraft"`
	UpdatedAt         string   `json:"updatedAt"`
	CreatedAt         string   `json:"createdAt"`
	Author            string   `json:"author"`
	AuthorAssociation *string  `json:"authorAssociation"`
	BaseRefName       *string  `json:"baseRefName"`
	Mergeable         *string  `json:"mergeable"`
	ReviewDecision    *string  `json:"reviewDecision"`
	Additions         *int     `json:"additions"`
	Deletions         *int     `json:"deletions"`
	ChangedFiles      *int     `json:"changedFiles"`
	Labels            []string `json:"labels"`
	Assignees         []string `json:"assignees"`
	Files             []string `json:"files"`
	FilesTruncated    bool     `json:"files_truncated"`
	// Areas is the path-glob classification of Files — the deterministic
	// layer of rook-triage's references/label-map.md, stamped once here so no
	// triager re-derives it per item by eye. [] means classified and matched
	// nothing; null means NOT classified, which is what a truncated Files
	// gets: areas read off a partial file list are a subset, and a consumer
	// treating this as the answer cannot tell that subset from a whole one.
	Areas   []string `json:"areas"`
	Reviews Reviews  `json:"reviews"`
	CI      CI       `json:"ci"`
}

type IssueItem struct {
	Number        int      `json:"number"`
	Title         string   `json:"title"`
	State         string   `json:"state"`
	UpdatedAt     string   `json:"updatedAt"`
	CreatedAt     string   `json:"createdAt"`
	Author        string   `json:"author"`
	Labels        []string `json:"labels"`
	Assignees     []string `json:"assignees"`
	CommentsTotal int      `json:"comments_total"`
}

func shapePR(n prNode) *PRItem {
	item := &PRItem{
		Number:            n.Number,
		Title:             n.Title,
		State:             n.State,
		IsDraft:           n.IsDraft,
		UpdatedAt:         n.UpdatedAt,
		CreatedAt:         n.CreatedAt,
		Author:            authorLogin(n.Author),
		AuthorAssociation: n.AuthorAssociation,
		BaseRefName:       n.BaseRefName,
		Mergeable:         n.Mergeable,
		ReviewDecision:    n.ReviewDecision,
		Additions:         n.Additions,
		Deletions:         n.Deletions,
		ChangedFiles:      n.ChangedFiles,
		Labels:            make([]string, 0, len(n.Labels.Nodes)),
		Assignees:         make([]string, 0, len(n.Assignees.Nodes)),
		Files:             make([]string, 0, len(n.Files.Nodes)),
		FilesTruncated:    n.Files.PageInfo.HasNextPage,
		Reviews: Reviews{
			Latest:    make([]Review, 0, len(n.LatestReviews.Nodes)),
			Requested: make([]*string, 0, len(n.ReviewRequests.Nodes)),
		},
		CI: summarizeCI(n),
	}
	for _, l := range n.Labels.Nodes {
		item.Labels = append(item.Labels, l.Name)
	}
	for _, a := range n.Assignees.Nodes {
		item.Assignees = append(item.Assignees, a.Login)
	}
	for _, f := range n.Files.Nodes {
		item.Files = append(item.Files, f.Path)
	}
	for _, r := range n.LatestReviews.Nodes {
		if r.Author == nil {
			continue
		}
		item.Reviews.Latest = append(item.Reviews.Latest, Review{Login: r.Author.Login, State: r.State})
	}
	for _, rr := range n.ReviewRequests.Nodes {
		if len(rr.RequestedReviewer) == 0 {
			continue
		}
		item.Reviews.Requested = append(item.Reviews.Requested, reviewerName(rr.RequestedReviewer))
	}
	item.classifyAreas()
	return item
}

// classifyAreas stamps Areas from the file list currently on the item. It has
// to run again whenever that list changes: snapshotPRs re-fetches a truncated
// PR's remaining file pages, and areas inferred before that walk describe only
// the first page.
func (item *PRItem) classifyAreas() {
	if item.FilesTruncated {
		item.Areas = nil
		return
	}
	item.Areas = rtanalyze.AreasForPaths(item.Files)
}

func shapeIssue(n issueNode) *IssueItem {
	item := &IssueItem{
		Number:        n.Number,
		Title:         n.Title,
		State:         n.State,
		UpdatedAt:     n.UpdatedAt,
		CreatedAt:     n.CreatedAt,
		Author:        authorLogin(n.Author),
		Labels:        make([]string, 0, len(n.Labels.Nodes)),
		Assignees:     make([]string, 0, len(n.Assignees.Nodes)),
		CommentsTotal: n.Comments.TotalCount,
	}
	for _, l := range n.Labels.Nodes {
		item.Labels = append(item.Labels, l.Name)
	}
	for _, a := range n.Assignees.Nodes {
		item.Assignees = append(item.Assignees, a.Login)
	}
	return item
}

func summarizeCI(n prNode) CI {
	if len(n.Commits.Nodes) == 0 {
		return emptyCI()
	}
	commit := n.Commits.Nodes[0].Commit
	if commit == nil || commit.StatusCheckRollup == nil {
		return emptyCI()
	}
	rollup := commit.StatusCheckRollup
	return ClassifyContexts(rollup.State, rollup.Contexts.Nodes, rollup.Contexts.PageInfo.HasNextPage)
}

func emptyCI() CI {
	return CI{Failed: []string{}}
}

func authorLogin(a *login) string {
	if a == nil {
		return ""
	}
	return a.Login
}

func reviewerName(r map[string]json.RawMessage) *string {
	for _, field := range []string{"login", "name"} {
		var s string
		if err := json.Unmarshal(r[field], &s); err == nil && s != "" {
			return &s
		}
	}
	return nil
}
