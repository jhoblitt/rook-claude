package sweepprefetch

import "strconv"

var (
	passConclusions = map[string]bool{"SUCCESS": true, "NEUTRAL": true, "SKIPPED": true}
	failConclusions = map[string]bool{
		"FAILURE": true, "TIMED_OUT": true, "CANCELLED": true,
		"ACTION_REQUIRED": true, "STARTUP_FAILURE": true,
	}
)

type workflow struct {
	Name string `json:"name"`
}

type workflowRun struct {
	Workflow *workflow `json:"workflow"`
}

type checkSuite struct {
	DatabaseID  *int64       `json:"databaseId"`
	CreatedAt   string       `json:"createdAt"`
	WorkflowRun *workflowRun `json:"workflowRun"`
}

// ContextNode is one member of a statusCheckRollup's context union: a
// CheckRun (GitHub Actions and friends) or a StatusContext (the legacy commit
// status API, which rook still gets from Jenkins).
type ContextNode struct {
	Typename   string      `json:"__typename"`
	Name       string      `json:"name"`
	Status     string      `json:"status"`
	Conclusion string      `json:"conclusion"`
	CheckSuite *checkSuite `json:"checkSuite"`
	Context    string      `json:"context"`
	State      string      `json:"state"`
}

// CI is the summarized rollup: the only CI numbers a dashboard may print.
type CI struct {
	State     *string  `json:"state"`
	Passing   int      `json:"passing"`
	Failing   int      `json:"failing"`
	Pending   int      `json:"pending"`
	Total     int      `json:"total"`
	Failed    []string `json:"failed"`
	Truncated bool     `json:"truncated"`
}

type ctxKey struct {
	kind string
	name string
}

// suiteKey groups a CheckRun by the workflow it belongs to, falling back to
// the suite's own id for suites with no workflow run (third-party checks).
func suiteKey(c ContextNode) (key, created string) {
	if c.CheckSuite == nil {
		return "suite:None", ""
	}
	created = c.CheckSuite.CreatedAt
	if wr := c.CheckSuite.WorkflowRun; wr != nil && wr.Workflow != nil && wr.Workflow.Name != "" {
		return wr.Workflow.Name, created
	}
	id := "None"
	if c.CheckSuite.DatabaseID != nil {
		id = strconv.FormatInt(*c.CheckSuite.DatabaseID, 10)
	}
	return "suite:" + id, created
}

// ClassifyContexts summarizes context nodes the way GitHub's merge box does.
//
// The rollup keeps every CheckRun from every check suite ever run on the
// commit — a CI re-run months later under a changed workflow/matrix leaves the
// old generation's (differently-named) runs in the list, silently doubling job
// counts. So: keep only the NEWEST check suite per workflow, then dedupe by
// check name (re-run attempts within a suite), keeping the last occurrence.
func ClassifyContexts(state *string, nodes []ContextNode, truncated bool) CI {
	newest := map[string]string{}
	for _, c := range nodes {
		if c.Typename != "CheckRun" {
			continue
		}
		key, created := suiteKey(c)
		if prev, ok := newest[key]; !ok || created > prev {
			newest[key] = created
		}
	}

	var order []ctxKey
	latest := map[ctxKey]ContextNode{}
	for _, c := range nodes {
		var key ctxKey
		if c.Typename == "CheckRun" {
			suite, created := suiteKey(c)
			if created != newest[suite] {
				continue
			}
			key = ctxKey{"run", orUnnamed(c.Name)}
		} else {
			key = ctxKey{"ctx", orUnnamed(c.Context)}
		}
		if _, ok := latest[key]; !ok {
			order = append(order, key)
		}
		latest[key] = c
	}

	ci := CI{State: state, Failed: []string{}, Truncated: truncated}
	for _, key := range order {
		c := latest[key]
		if key.kind == "run" {
			switch {
			case c.Status != "COMPLETED":
				ci.Pending++
			case passConclusions[c.Conclusion]:
				ci.Passing++
			case failConclusions[c.Conclusion]:
				ci.Failing++
				ci.Failed = append(ci.Failed, key.name)
			default:
				ci.Pending++
			}
			continue
		}
		switch c.State {
		case "SUCCESS":
			ci.Passing++
		case "FAILURE", "ERROR":
			ci.Failing++
			ci.Failed = append(ci.Failed, key.name)
		default:
			ci.Pending++
		}
	}
	ci.Total = ci.Passing + ci.Failing + ci.Pending
	return ci
}

func orUnnamed(s string) string {
	if s == "" {
		return "?"
	}
	return s
}
