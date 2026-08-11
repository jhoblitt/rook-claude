package sweepprefetch

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func prNodeWithFiles(number int, truncated bool, paths ...string) prNode {
	var n prNode
	n.Number = number
	n.Files.PageInfo.HasNextPage = truncated
	for _, p := range paths {
		n.Files.Nodes = append(n.Files.Nodes, struct {
			Path string `json:"path"`
		}{Path: p})
	}
	return n
}

func TestShapePRStampsAreas(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		want  []string
	}{
		{"union across paths, in taxonomy order", []string{
			"Documentation/CRDs/cluster.md",
			"pkg/operator/ceph/csi/spec.go",
			"pkg/operator/ceph/object/rgw.go",
		}, []string{"object", "csi", "docs"}},
		{"one path, several areas", []string{
			"pkg/operator/ceph/object/zone.go",
		}, []string{"object", "object-multisite"}},
		{"deliberately unbucketed paths", []string{
			"deploy/examples/cluster.yaml", "README.md",
		}, []string{}},
		{"no files at all", nil, []string{}},
		{"duplicate hits collapse", []string{
			"pkg/operator/ceph/csi/a.go", "pkg/operator/ceph/csi/b.go",
		}, []string{"csi"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shapePR(prNodeWithFiles(1, false, tc.paths...)).Areas
			if got == nil {
				t.Fatal("areas = null on a complete file list")
			}
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Errorf("areas = %v, want %v", got, tc.want)
			}
		})
	}
}

// A file list cut short by pagination can only produce a subset of the real
// areas, and a subset is indistinguishable from the whole answer once it is in
// the snapshot. It has to serialize as null — the absence of a classification
// — rather than as the areas the fetched page happens to imply.
func TestShapePRWithholdsAreasOnTruncatedFiles(t *testing.T) {
	item := shapePR(prNodeWithFiles(91, true, "pkg/operator/ceph/csi/spec.go"))
	if item.Areas != nil {
		t.Fatalf("areas = %v on a truncated file list, want null", item.Areas)
	}
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"areas":null`) {
		t.Errorf("marshalled item does not carry a null areas:\n%s", data)
	}
}

// The batched query caps files at one page, so a big PR's areas would be read
// off that first page unless the classifier re-runs on the paginated list.
// Page 1 here carries only pkg/operator/a.go (core); the full list adds
// Documentation/c.md (docs).
func TestSnapshotPRsClassifiesAfterFilesPagination(t *testing.T) {
	stub := newStub(t, "open-prs-page1.json", "open-prs-page2.json",
		"contexts-page1.json", "contexts-page2.json",
		"files-page1.json", "files-page2.json")
	items, err := testClient(t, stub).snapshotPRs(context.Background(), SnapshotOptions{Kind: "prs"})
	if err != nil {
		t.Fatal(err)
	}
	stub.done()

	for _, item := range items {
		if item.Number != 10 {
			continue
		}
		if got := strings.Join(item.Areas, ","); got != "docs,core" {
			t.Errorf("areas = %q, want %q — classified before the files walk finished?", got, "docs,core")
		}
		return
	}
	t.Fatal("PR 10 missing from the snapshot")
}
