package sweepprefetch

import (
	"encoding/json"
	"reflect"
	"testing"
)

func strp(s string) *string { return &s }

func checkRun(name, status, conclusion, wf, created string) ContextNode {
	return ContextNode{
		Typename: "CheckRun", Name: name, Status: status, Conclusion: conclusion,
		CheckSuite: &checkSuite{
			DatabaseID:  ptrInt64(1),
			CreatedAt:   created,
			WorkflowRun: &workflowRun{Workflow: &workflow{Name: wf}},
		},
	}
}

func suiteRun(name, status, conclusion string, id int64, created string) ContextNode {
	return ContextNode{
		Typename: "CheckRun", Name: name, Status: status, Conclusion: conclusion,
		CheckSuite: &checkSuite{DatabaseID: ptrInt64(id), CreatedAt: created},
	}
}

func statusCtx(context, state string) ContextNode {
	return ContextNode{Typename: "StatusContext", Context: context, State: state}
}

func ptrInt64(n int64) *int64 { return &n }

func TestClassifyContexts(t *testing.T) {
	tests := []struct {
		name  string
		nodes []ContextNode
		want  CI
	}{
		{
			name: "no contexts",
			want: CI{Failed: []string{}},
		},
		{
			name: "only the newest suite of a workflow counts",
			nodes: []ContextNode{
				checkRun("build", "COMPLETED", "SUCCESS", "canary", "2026-07-01T00:00:00Z"),
				checkRun("retired-matrix-job", "COMPLETED", "FAILURE", "canary", "2026-07-01T00:00:00Z"),
				checkRun("build", "COMPLETED", "FAILURE", "canary", "2026-08-01T00:00:00Z"),
			},
			want: CI{Failing: 1, Total: 1, Failed: []string{"build"}},
		},
		{
			name: "a re-run within a suite keeps the last attempt",
			nodes: []ContextNode{
				checkRun("unit", "IN_PROGRESS", "", "canary", "2026-08-01T00:00:00Z"),
				checkRun("unit", "COMPLETED", "SUCCESS", "canary", "2026-08-01T00:00:00Z"),
			},
			want: CI{Passing: 1, Total: 1, Failed: []string{}},
		},
		{
			name: "an unfinished or unrecognized run is pending",
			nodes: []ContextNode{
				checkRun("queued", "QUEUED", "", "canary", "2026-08-01T00:00:00Z"),
				checkRun("stale", "COMPLETED", "STALE", "canary", "2026-08-01T00:00:00Z"),
				checkRun("skipped", "COMPLETED", "SKIPPED", "canary", "2026-08-01T00:00:00Z"),
			},
			want: CI{Passing: 1, Pending: 2, Total: 3, Failed: []string{}},
		},
		{
			name: "legacy status contexts",
			nodes: []ContextNode{
				statusCtx("codecov/patch", "SUCCESS"),
				statusCtx("ci/jenkins", "FAILURE"),
				statusCtx("ci/other", "ERROR"),
				statusCtx("ci/waiting", "PENDING"),
			},
			want: CI{Passing: 1, Failing: 2, Pending: 1, Total: 4,
				Failed: []string{"ci/jenkins", "ci/other"}},
		},
		{
			name: "suites with no workflow run are keyed by id but still deduped by name",
			nodes: []ContextNode{
				suiteRun("lint", "COMPLETED", "FAILURE", 3, "2026-08-01T00:00:00Z"),
				suiteRun("lint", "COMPLETED", "SUCCESS", 4, "2026-08-02T00:00:00Z"),
			},
			want: CI{Passing: 1, Total: 1, Failed: []string{}},
		},
		{
			name: "a run with no check suite still counts",
			nodes: []ContextNode{
				{Typename: "CheckRun", Name: "orphan", Status: "COMPLETED", Conclusion: "TIMED_OUT"},
			},
			want: CI{Failing: 1, Total: 1, Failed: []string{"orphan"}},
		},
		{
			name: "nameless entries collapse onto one placeholder",
			nodes: []ContextNode{
				checkRun("", "COMPLETED", "FAILURE", "canary", "2026-08-01T00:00:00Z"),
				statusCtx("", "SUCCESS"),
			},
			want: CI{Passing: 1, Failing: 1, Total: 2, Failed: []string{"?"}},
		},
		{
			name: "failed keeps the order the contexts arrived in",
			nodes: []ContextNode{
				statusCtx("z-last", "FAILURE"),
				checkRun("a-first", "COMPLETED", "ACTION_REQUIRED", "canary", "2026-08-01T00:00:00Z"),
			},
			want: CI{Failing: 2, Total: 2, Failed: []string{"z-last", "a-first"}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyContexts(nil, tc.nodes, false)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ClassifyContexts() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestClassifyContextsCarriesStateAndTruncation(t *testing.T) {
	got := ClassifyContexts(strp("PENDING"), nil, true)
	if got.State == nil || *got.State != "PENDING" || !got.Truncated {
		t.Errorf("ClassifyContexts() = %+v, want state PENDING and truncated", got)
	}
}

func TestSummarizeCIWithoutRollup(t *testing.T) {
	tests := []struct {
		name string
		node string
	}{
		{"no commits", `{"commits": {"nodes": []}}`},
		{"no commit", `{"commits": {"nodes": [{"commit": null}]}}`},
		{"no rollup", `{"commits": {"nodes": [{"commit": {"statusCheckRollup": null}}]}}`},
	}
	want := CI{Failed: []string{}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var n prNode
			if err := json.Unmarshal([]byte(tc.node), &n); err != nil {
				t.Fatal(err)
			}
			if got := summarizeCI(n); !reflect.DeepEqual(got, want) {
				t.Errorf("summarizeCI() = %+v, want %+v", got, want)
			}
		})
	}
}
