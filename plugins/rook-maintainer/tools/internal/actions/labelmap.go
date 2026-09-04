package actions

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

const (
	labelColumn = "Issue label"
	areaColumn  = "Area"
)

var labelToken = regexp.MustCompile("`([^`]+)`")

// ParseLabelMap returns every label named in the "Issue label" column of
// label-map.md's area table, in table order and deduplicated. A cell may name
// more than one (`filesystem` with an MDS-specific `ceph-mds`); each counts.
//
// Deliberately narrow: only the table whose header carries that column is read.
// The file names labels in prose too — the Category and Lifecycle paragraphs —
// and a gate that demanded those would fail on labels the table never claimed
// and on the ones the prose says do not exist yet.
func ParseLabelMap(md []byte) ([]string, error) {
	var mapped []string
	seen := map[string]bool{}
	col := -1
	for _, line := range strings.Split(string(md), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			col = -1
			continue
		}
		cells := tableCells(line)
		if col < 0 {
			for i, c := range cells {
				if c == labelColumn {
					col = i
				}
			}
			continue
		}
		if col >= len(cells) {
			continue
		}
		for _, label := range backtickSpans(cells[col]) {
			if !seen[label] {
				seen[label] = true
				mapped = append(mapped, label)
			}
		}
	}
	if len(mapped) == 0 {
		return nil, fmt.Errorf("no %q column with labels in it", labelColumn)
	}
	return mapped, nil
}

func tableCells(row string) []string {
	var cells []string
	for _, c := range strings.Split(strings.Trim(row, "|"), "|") {
		cells = append(cells, strings.TrimSpace(c))
	}
	return cells
}

// DiffLabels compares the labels label-map.md names against the repo's live
// list. missing is drift the map has to answer for: a proposal naming one of
// those cannot be applied, and phase 5's membership check would reject it one
// item at a time instead of naming the map as the cause. unmapped is the repo's
// own growth — reported so the map can catch up, never a failure, since a
// maintainer may create a label this plugin has no rule for.
//
// Comparison is exact: GitHub shows a label as its author spelled it, and a
// case difference between the map and the repo is drift too.
func DiffLabels(mapped, live []string) (missing, unmapped []string) {
	inMap := make(map[string]bool, len(mapped))
	for _, label := range mapped {
		inMap[label] = true
	}
	inRepo := make(map[string]bool, len(live))
	for _, label := range live {
		inRepo[label] = true
	}
	for _, label := range mapped {
		if !inRepo[label] {
			missing = append(missing, label)
		}
	}
	for _, label := range live {
		if !inMap[label] {
			unmapped = append(unmapped, label)
		}
	}
	return missing, unmapped
}

// ParseLabelAreas maps each label of the "Issue label" column to the areas
// whose rows name it, in table order and deduplicated. It is ParseLabelMap's
// other direction — same table, same backtick-span rule, read as the label →
// area translation the table's own prose says a caller must make rather than
// as a flat label list.
//
// A label may reach more than one area (`networking` and `multus` share a row,
// but nothing stops a later row from naming an existing label), so the value
// is a list; an Area cell naming several areas maps its row's labels to each.
func ParseLabelAreas(md []byte) (map[string][]string, error) {
	byLabel := map[string][]string{}
	labelCol, areaCol := -1, -1
	for _, line := range strings.Split(string(md), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			labelCol, areaCol = -1, -1
			continue
		}
		cells := tableCells(line)
		if labelCol < 0 || areaCol < 0 {
			for i, c := range cells {
				switch c {
				case labelColumn:
					labelCol = i
				case areaColumn:
					areaCol = i
				}
			}
			continue
		}
		if labelCol >= len(cells) || areaCol >= len(cells) {
			continue
		}
		for _, label := range backtickSpans(cells[labelCol]) {
			for _, area := range backtickSpans(cells[areaCol]) {
				if !slices.Contains(byLabel[label], area) {
					byLabel[label] = append(byLabel[label], area)
				}
			}
		}
	}
	if len(byLabel) == 0 {
		return nil, fmt.Errorf("no table with both a %q and an %q column", labelColumn, areaColumn)
	}
	return byLabel, nil
}

func backtickSpans(cell string) []string {
	var out []string
	for _, m := range labelToken.FindAllStringSubmatch(cell, -1) {
		if s := strings.TrimSpace(m[1]); s != "" {
			out = append(out, s)
		}
	}
	return out
}
