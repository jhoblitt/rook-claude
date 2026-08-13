Pass if and only if ALL of:

1. The first prose paragraph of the drafted body states the
   maintainer's motivation — the crash-looping operator killing
   `kubectl logs -f` and losing the lines that matter — before any
   statement of what the PR adds. A leading title/heading line is fine;
   the first prose paragraph is what counts.
2. The description prose — everything above the `[checklist]` line
   except the AI-assistance note — is under 100 words, counted as
   whitespace-separated tokens over that span exactly as written
   (`wc -w`: markup, bullet markers, and headings all count). State
   the count you arrived at.
3. The body contains no process notes (rebases, sanity checks that
   found nothing, draft status, labels, which remote it was pushed
   from).
4. No section is padded to fill the shape: a "notable decisions"
   paragraph, if present, states a choice a reviewer would actually
   question, not a restatement of what the change does.

Fail if any of:

- The body opens by describing the subcommand or a generic convenience
  rationale (e.g. "reading the operator log means knowing the namespace
  and label selector") with the crash-loop motivation absent or buried
  below it.
- 100 words or more of description prose above `[checklist]`,
  excluding the AI-assistance note.
- The agent asks a question instead of drafting — the motivation IS
  stated in the request, so there is nothing to ask about.
