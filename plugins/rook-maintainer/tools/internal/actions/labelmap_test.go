package actions

import (
	"os"
	"strings"
	"testing"
)

const labelMapFile = "../../../skills/rook-triage/references/label-map.md"

// The parser's input is one specific file, so the file is the fixture: a
// restructured table has to fail here rather than silently mapping nothing.
func TestParseLabelMapReadsTheShippedTable(t *testing.T) {
	md, err := os.ReadFile(labelMapFile)
	if err != nil {
		t.Fatal(err)
	}
	mapped, err := ParseLabelMap(md)
	if err != nil {
		t.Fatal(err)
	}
	in := map[string]bool{}
	for _, label := range mapped {
		in[label] = true
	}
	// The second label of a two-label cell is the one a lazier parser drops.
	for _, label := range []string{"object", "ceph-rgw", "ceph-mds", "api", "networking",
		"ceph-exporter", "operator"} {
		if !in[label] {
			t.Errorf("%q is in the table's label column, not in %v", label, mapped)
		}
	}
	// Column 1 is paths and column 2 is the area taxonomy; neither is a label.
	for _, notALabel := range []string{"pkg/operator/ceph/file/**", "core", "UX", "technical debt"} {
		if in[notALabel] {
			t.Errorf("%q was read as a label", notALabel)
		}
	}
	if len(mapped) < 25 {
		t.Errorf("read only %d labels from the table: %v", len(mapped), mapped)
	}
}

func TestParseLabelMapIgnoresEverythingButThatTable(t *testing.T) {
	md := "Prose naming `bug` and `UX` in passing.\n\n" +
		"| Thing | Other |\n|---|---|\n| `not-a-label` | `nor-this` |\n\n" +
		"| Paths touched | Area | Issue label |\n|---|---|---|\n" +
		"| `pkg/**` | `core` | `operator` |\n" +
		"| `x/**` | `file` | `filesystem` (MDS-specific: `ceph-mds`) |\n\n" +
		"More prose about `stale`.\n"
	mapped, err := ParseLabelMap([]byte(md))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(mapped, ","); got != "operator,filesystem,ceph-mds" {
		t.Errorf("mapped = %q", got)
	}
}

func TestParseLabelMapRejectsADocumentWithNoSuchTable(t *testing.T) {
	if _, err := ParseLabelMap([]byte("| a | b |\n|---|---|\n| `x` | `y` |\n")); err == nil {
		t.Error("accepted a document with no Issue label column; the gate would map nothing and pass")
	}
}

func TestDiffLabels(t *testing.T) {
	missing, unmapped := DiffLabels(
		[]string{"bug", "invented", "docs"},
		[]string{"docs", "bug", "Bug", "keepalive"})
	if strings.Join(missing, ",") != "invented" {
		t.Errorf("missing = %v, want the label the repo lacks", missing)
	}
	// "Bug" is drift both ways: the repo has it, the map names "bug".
	if strings.Join(unmapped, ",") != "Bug,keepalive" {
		t.Errorf("unmapped = %v, want the repo labels the map does not name", unmapped)
	}
}
