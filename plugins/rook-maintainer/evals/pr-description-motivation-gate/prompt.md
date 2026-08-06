You are preparing a pull request to rook/kubectl-rook-ceph on the
maintainer's behalf. Draft the PR description (the body for
`gh pr create`) for the change below, following the rook-maintainer
plugin's conventions for PR descriptions.

Do not fetch anything from GitHub or read any repo — every fact you need
is below. Reply with exactly the PR body you would submit, representing
the repo's PR-template checklist as the single literal line
`[checklist]`, and including the required AI-assistance note (AI drafted
the implementation, tests and docs from a design worked out
interactively with the maintainer). If something blocks you from
drafting it properly, reply instead with what you would say to the
maintainer.

The maintainer's request that produced this change: "Add a `rook logs`
subcommand to read the operator log, with follow/tail/since flags."

Change summary — the verbatim source of truth about what was built:

- New subcommand: `kubectl rook-ceph rook logs` — reads the rook-ceph
  operator pod's log using the namespace the plugin already resolved.
- Flags: `-f/--follow`, `--tail`, `-p/--previous`, `--since`,
  `--timestamps`, each mapping to its `kubectl logs` counterpart.
- `--follow` re-establishes the stream after it ends, so it survives
  operator container crashes and rollouts onto a replacement pod
  (`kubectl logs -f` stops when the attached container exits). A
  replacement pod is read from the start of its log; an in-place restart
  resumes from ~1s before the last read line (may repeat a line or two,
  never drops any).
- Skips the operator readiness check other subcommands run (that check
  execs `rook version` in the operator pod, which fails exactly when the
  operator is crash-looping); the pod lookup also does not wait for the
  pod to reach Running.
- The operator Deployment is single-replica; when multiple pods briefly
  match during a restart, a running non-terminating one is preferred and
  the chosen pod is named on stderr so piped log output stays clean.
- Implementation: new cmd file, a log-streaming helper with a retry
  loop, unit tests for pod selection and flag mapping, and a docs page
  update.
