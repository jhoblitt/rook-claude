// Package ghx runs the authenticated `gh` CLI on behalf of the tools.
//
// Shelling out rather than speaking to the REST/GraphQL APIs directly is
// deliberate: `gh` already holds the maintainer's credentials, honors their
// enterprise host config, and is the thing the sandbox notes are written
// about. Reimplementing auth here would create a second credential path to
// keep correct.
package ghx

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const DefaultTimeout = 180 * time.Second

// Run executes gh with args and returns stdout. stderr is truncated into the
// error: gh reports rate limiting and auth failures there, and a caller that
// swallows it leaves the maintainer guessing which of the two happened.
func Run(ctx context.Context, timeout time.Duration, args ...string) ([]byte, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, "gh", args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if len(msg) > 3000 {
			msg = msg[:3000] + "..."
		}
		return nil, fmt.Errorf("gh %s: %w: %s", strings.Join(args, " "), err, msg)
	}
	return out, nil
}

type graphQLError struct {
	Message string `json:"message"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphQLError  `json:"errors"`
}

// GraphQL runs one query and unmarshals the `data` object into out.
//
// A GraphQL response can carry HTTP 200 with a populated `errors` array and a
// partially-null `data` — the failure mode that silently produces an empty
// sweep — so errors are surfaced rather than left for the caller to notice.
func GraphQL(ctx context.Context, query string, out any) error {
	raw, err := Run(ctx, DefaultTimeout, "api", "graphql", "-f", "query="+query)
	if err != nil {
		return err
	}
	var resp graphQLResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return fmt.Errorf("decoding gh graphql response: %w", err)
	}
	if len(resp.Errors) > 0 {
		msgs := make([]string, 0, len(resp.Errors))
		for _, e := range resp.Errors {
			msgs = append(msgs, e.Message)
		}
		return fmt.Errorf("graphql errors: %s", strings.Join(msgs, "; "))
	}
	if len(resp.Data) == 0 {
		return fmt.Errorf("graphql response carried no data object")
	}
	return json.Unmarshal(resp.Data, out)
}
