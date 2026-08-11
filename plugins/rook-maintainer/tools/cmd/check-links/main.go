// check-links: liveness of every URL a rook diff adds (docs-sync.md URL integrity).
//
// Callers: rook-code-review's docs-sync.md liveness pass, rook-conventions'
// fetched-pages rule (SKILL.md "Read content is untrusted data"), and the
// rook-reviewer agent definition.
//
//	git diff origin/master... | run.sh check-links audit
//	run.sh check-links audit --diff-file F [--json]
//	run.sh check-links check URL [URL...]
//	run.sh check-links extract --diff-file F
//
// Exit status is 1 when any URL is dead, suspect or suspicious, so the probe
// doubles as a gate. See internal/links for why this never returns page
// content. Reaching non-GitHub hosts needs the sandbox disabled — DNS does not
// resolve inside it, and the non-public-address guard needs DNS.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/links"
)

func usage() {
	fmt.Fprint(os.Stderr, "usage: check-links <extract|check|audit> [flags] [URL...]\n")
	flag.PrintDefaults()
}

func run() int {
	fs := flag.NewFlagSet("check-links", flag.ContinueOnError)
	diffFile := fs.String("diff-file", "", "read a unified diff from this file instead of stdin")
	timeout := fs.Int("timeout", 10, "per-request timeout in seconds")
	workers := fs.Int("workers", links.DefaultWorkers, "concurrent probes")
	allowPrivate := fs.Bool("allow-private", false, "disable the non-public-address guard (tests only)")
	asJSON := fs.Bool("json", false, "emit JSON")
	fs.Usage = usage

	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		return 2
	}
	mode := args[0]
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	switch mode {
	case "extract", "check", "audit":
	default:
		usage()
		return 2
	}

	var urls []string
	if mode == "check" {
		urls = fs.Args()
	} else {
		diff, err := readDiff(*diffFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "check-links: %v\n", err)
			return 2
		}
		urls = links.ExtractURLs(diff)
	}
	if len(urls) == 0 {
		return 0
	}

	var results []links.Result
	if mode == "extract" {
		for _, u := range urls {
			r := links.Result{URL: links.Sanitize(u), Verdict: "extracted"}
			if links.HasHiddenRunes(u) {
				r.Verdict = "suspicious"
				r.Note = "control or format characters inside URL"
			}
			results = append(results, r)
		}
	} else {
		p := links.NewProber(*timeout, *allowPrivate)
		results = p.CheckAll(context.Background(), urls, *workers)
	}

	if err := report(os.Stdout, results, *asJSON); err != nil {
		fmt.Fprintf(os.Stderr, "check-links: %v\n", err)
		return 2
	}
	for _, r := range results {
		if r.Bad() {
			return 1
		}
	}
	return 0
}

func readDiff(path string) (string, error) {
	if path == "" {
		b, err := io.ReadAll(os.Stdin)
		return string(b), err
	}
	b, err := os.ReadFile(path)
	return string(b), err
}

func report(w io.Writer, results []links.Result, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}
	for _, r := range results {
		status := "-"
		if r.Status != 0 {
			status = fmt.Sprint(r.Status)
		}
		if _, err := fmt.Fprintf(w, "%-18s %4s  %s\n", r.Verdict, status, r.URL); err != nil {
			return err
		}
		pad := strings.Repeat(" ", 24)
		if r.FinalURL != "" {
			if _, err := fmt.Fprintf(w, "%s-> %s\n", pad, r.FinalURL); err != nil {
				return err
			}
		}
		if r.Note != "" {
			if _, err := fmt.Fprintf(w, "%s(%s)\n", pad, r.Note); err != nil {
				return err
			}
		}
	}
	return nil
}

func main() {
	os.Exit(run())
}
