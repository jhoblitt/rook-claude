There is no rook checkout, no network,
and no `gh`, and subagents cannot be spawned in this environment: treat the
PR metadata below as complete — there is nothing further to fetch — and run
the passes of the rook-code-review skill's review spine inline.

Review the following pull request. Your entire final answer is the review
report — verdict line, findings in the skill's finding contract, and the
audited-and-clean statement. Nothing else.

---

**PR #18086** — base `master`, author `dependabot[bot]`

Title: `build(deps): bump go.uber.org/zap from 1.27.1 to 1.28.0`

Body:

```text
Bumps [go.uber.org/zap](https://github.com/uber-go/zap) from 1.27.1 to
1.28.0.

<details>
<summary>Release notes</summary>

<ul>
<li><a href="https://redirect.github.com/uber-go/zap/issues/1534">#1534</a>:
Add <code>zapcore.CheckPreWriteHook</code>.</li>
<li>Fixes <a href="https://redirect.github.com/uber-go/zap/issues/1502">#1502</a>:
data race in <code>Sugar()</code>.</li>
<li>Closes <a href="https://redirect.github.com/uber-go/zap/issues/1488">#1488</a>:
drop Go 1.21 support.</li>
</ul>
</details>
```

Commits (one):

```text
build(deps): bump go.uber.org/zap from 1.27.1 to 1.28.0

Signed-off-by: dependabot[bot] <support@github.com>
```

Diff:

```diff
--- a/go.mod
+++ b/go.mod
@@ -30,2 +30,2 @@ require (
-	go.uber.org/zap v1.27.1
+	go.uber.org/zap v1.28.0
 )
```
