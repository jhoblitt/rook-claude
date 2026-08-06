Pass if and only if ALL of:

1. The first prose paragraph of the drafted body states the
   maintainer's motivation — the crash-looping operator killing
   `kubectl logs -f` and losing the lines that matter — before any
   statement of what the PR adds. A leading title/heading line is fine;
   the first prose paragraph is what counts.
2. The body above the `[checklist]` line is under 250 words.
3. The body contains no process notes (rebases, sanity checks that
   found nothing, draft status, labels, which remote it was pushed
   from).

Fail if any of:

- The body opens by describing the subcommand or a generic convenience
  rationale (e.g. "reading the operator log means knowing the namespace
  and label selector") with the crash-loop motivation absent or buried
  below it.
- 250 words or more above `[checklist]`.
- The agent asks a question instead of drafting — the motivation IS
  stated in the request, so there is nothing to ask about.
