#!/usr/bin/env python3
"""rook-code-review sweep dashboard generator (contract: sweep.md phase 3).

Reads ONLY canonical sweep-dir inputs — snapshot.json (live metadata via
sweep_prefetch.py: titles, authors, labels, CI rollup), pr-*/findings.json
(verified findings with assigned IDs, recomputed verdict, backport
assessment) and sweep.json (per-PR status, skip rows) — and writes
<sweep-dir>/dashboard.html. Verdicts and finding counts are ALWAYS read
from findings.json, never parsed from agent prose; CI cells are always
passing/total from the snapshot.

pr-<N>/findings.json shape (the phase-3 write; fields beyond `findings`
are optional and render as "—" when absent):

  {"pr": 18123, "verdict": "ACCEPT|REQUEST_CHANGES|REJECT",
   "bug": "REAL|FABRICATED|N/A", "rationale": "one paragraph",
   "backport": {"eligible": true, "label": "...", "reason": "..."},
   "needs_proposal_review": {"flag": false, "paths": []},
   "takeover_candidate": {"flag": false, "reason": ""},
   "findings": [{"id": "B1", "severity": "blocker", "domain": "bug",
                 "path": "pkg/...", "line": 42, "summary": "...",
                 "confidence": 85, "status": "pending|approved|posted|dropped"}],
   "clean": ["areas audited and found correct"]}

Usage: python3 gen_review_dashboard.py SWEEP_DIR
"""
import glob
import html
import json
import os
import sys

sw = os.path.abspath(sys.argv[1])
date = os.path.basename(sw)[:10]


def load(path, default):
    return json.load(open(path)) if os.path.exists(path) else default


snap = load(f"{sw}/snapshot.json", {"items": {}})["items"]
sweep = load(f"{sw}/sweep.json", {})
repo = load(f"{sw}/snapshot.json", {}).get("repo", "rook/rook")

prs = []
for d in sorted(glob.glob(f"{sw}/pr-*/findings.json")):
    raw = json.load(open(d))
    rec: dict = {"findings": raw} if isinstance(raw, list) else raw
    if "pr" not in rec:
        rec["pr"] = int(os.path.basename(os.path.dirname(d)).removeprefix("pr-"))
    prs.append(rec)
prs.sort(key=lambda r: r["pr"])

SEV = {"blocker": ("B", "sev-b", "blocker"),
       "changes-requested": ("C", "sev-c", "changes requested"),
       "nit": ("N", "sev-n", "nit"),
       "question": ("Q", "sev-q", "design question")}
ORDER = ["blocker", "changes-requested", "nit", "question"]
VERDICT_RANK = {"REJECT": 0, "REQUEST_CHANGES": 1, "ACCEPT": 2}


# ---------------------------------------------------------------------------
# Row-class policy: what a row's colour means at a glance.
#
# Verdict wins over severity here, deliberately — the sweep's phase-4 walk
# is ordered REJECT, REQUEST_CHANGES, ACCEPT (worst first), so the colour
# tracks the order the maintainer will actually work the list in. The two
# escalation flags outrank the verdict because both mean the verdict is not
# yet final: needs_proposal_review holds it provisional until proposal mode
# runs, and a takeover candidate changes who carries the PR, not whether
# the diff is correct.
# ---------------------------------------------------------------------------
def row_class(r):
    if (r.get("needs_proposal_review") or {}).get("flag"):
        return "prop"
    if (r.get("takeover_candidate") or {}).get("flag"):
        return "take"
    return {"REJECT": "reject", "REQUEST_CHANGES": "chg",
            "ACCEPT": "accept"}.get(r.get("verdict"), "mon")


def live(r):
    return [f for f in r.get("findings", []) if f.get("status") != "dropped"]


def sev_chips(r):
    counts = {}
    for f in live(r):
        counts[f.get("severity", "nit")] = counts.get(f.get("severity", "nit"), 0) + 1
    if not counts:
        return '<span class="chip none" title="no findings survived verification">clean</span>'
    out = []
    for s in ORDER:
        if counts.get(s):
            letter, klass, label = SEV[s]
            out.append(f'<span class="chip {klass}" title="{counts[s]} {label}">'
                       f'{letter}&#8202;{counts[s]}</span>')
    return "".join(out)


def verdict_cell(r):
    v = r.get("verdict") or "—"
    bug = r.get("bug")
    extra = ""
    if bug and bug != "N/A":
        k = "bug-real" if bug == "REAL" else "bug-fab"
        extra = f'<span class="chip {k}" title="claimed defect judged {bug}">bug&#8202;{bug.lower()}</span>'
    flags = []
    if (r.get("needs_proposal_review") or {}).get("flag"):
        paths = ", ".join((r["needs_proposal_review"].get("paths") or [])[:3]) or "design doc"
        flags.append(f'<span class="chip prop" title="verdict provisional until proposal mode runs on {html.escape(paths)}">proposal</span>')
    if (r.get("takeover_candidate") or {}).get("flag"):
        reason = (r.get("takeover_candidate") or {}).get("reason", "")
        flags.append(f'<span class="chip take" title="{html.escape(reason)}">takeover</span>')
    return f'<b>{html.escape(v)}</b> {extra}{"".join(flags)}'


def backport_cell(r):
    b = r.get("backport") or {}
    if not b.get("eligible"):
        return "—"
    label = b.get("label") or "eligible"
    return (f'<span class="chip bp" title="{html.escape(b.get("reason", ""))}">'
            f'{html.escape(label)}</span>')


def ci_chip(n):
    ci = (snap.get(str(n)) or {}).get("ci") or {}
    total, passing = ci.get("total", 0), ci.get("passing", 0)
    failing, pending = ci.get("failing", 0), ci.get("pending", 0)
    if total == 0:
        return '<span class="ci amber" title="no checks recorded">…</span>'
    kind = "red" if failing else ("amber" if pending else "green")
    parts = [f"{passing}/{total} passing"]
    if failing:
        names = ", ".join(ci.get("failed", [])[:6])
        more = len(ci.get("failed", [])) - 6
        parts.append(f"{failing} failing: {names}" + (f" +{more} more" if more > 0 else ""))
    if pending:
        parts.append(f"{pending} pending")
    return (f'<span class="ci {kind}" title="{html.escape(" · ".join(parts))}">'
            f'{passing}/{total}</span>')


def findings_detail(r):
    rows = []
    for f in sorted(live(r), key=lambda f: (ORDER.index(f.get("severity", "nit"))
                                            if f.get("severity") in ORDER else 9, f.get("id", ""))):
        letter, klass, _ = SEV.get(f.get("severity", "nit"), ("?", "sev-n", ""))
        anchor = f.get("path") or "PR-level"
        if f.get("path") and f.get("line"):
            anchor = f'{f["path"]}:{f["line"]}'
        conf = f.get("confidence")
        conf_txt = ("CONFIRMED" if isinstance(conf, int) and conf >= 80
                    else "PLAUSIBLE" if isinstance(conf, int) and conf >= 50 else "—")
        status = f.get("status", "pending")
        rows.append(
            f'<tr><td><span class="chip {klass}">{html.escape(f.get("id", letter))}</span></td>'
            f'<td>{html.escape(f.get("domain", ""))}</td>'
            f'<td class="anch">{html.escape(anchor)}</td>'
            f'<td>{html.escape(f.get("summary", ""))}</td>'
            f'<td>{conf_txt}{f" ({conf})" if isinstance(conf, int) and conf else ""}</td>'
            f'<td class="st-{html.escape(status)}">{html.escape(status)}</td></tr>')
    clean = r.get("clean") or []
    clean_html = ("<p class='clean'><b>Audited and clean:</b> "
                  + html.escape("; ".join(clean)) + "</p>") if clean else ""
    if not rows:
        return f"<p class='clean'>No findings survived verification.</p>{clean_html}"
    return (f'<table class="inner"><thead><tr><th>id</th><th>domain</th><th>anchor</th>'
            f'<th>summary</th><th>confidence</th><th>status</th></tr></thead>'
            f'<tbody>{"".join(rows)}</tbody></table>{clean_html}')


body = []
for r in prs:
    n = r["pr"]
    s = snap.get(str(n)) or {}
    author = s.get("author", "")
    assoc = s.get("authorAssociation") or ""
    body.append(
        f'<tr class="{row_class(r)}">'
        f'<td><a href="https://github.com/{repo}/pull/{n}">#{n}</a></td>'
        f'<td>{verdict_cell(r)}</td>'
        f'<td>{sev_chips(r)}</td>'
        f'<td class="sum">{html.escape(s.get("title", ""))}</td>'
        f'<td>{ci_chip(n)}</td>'
        f'<td>{backport_cell(r)}</td>'
        f'<td class="who"><a class="u" href="https://github.com/{html.escape(author)}">'
        f'{html.escape(author)}</a>{f"<br><small>{html.escape(assoc.lower())}</small>" if assoc else ""}</td>'
        f'</tr>')
    body.append(f'<tr class="det {row_class(r)}"><td></td><td colspan="6">'
                f'<details><summary>{len(live(r))} finding(s) · '
                f'{html.escape((r.get("rationale") or "")[:160])}</summary>'
                f'{findings_detail(r)}</details></td></tr>')

skips = sweep.get("skipped") or []
skiprows = "\n".join(
    f'<tr class="skip"><td><a href="https://github.com/{repo}/pull/{s["number"]}">#{s["number"]}</a></td>'
    f'<td colspan="6">{html.escape(str(s.get("reason", "")))}</td></tr>' for s in skips)

TOKENS = """--bg:#fff;--fg:#1a1d21;--mut:#67707b;--line:#e3e6ea;--chip:#f2f4f6;
--reject:#a04545;--chg:#8a6d1f;--accept:#1d8a5f;--prop:#7a5ba6;--take:#3d6ea8;--skip:#999;
--cigreen:#d9efe2;--cigreenfg:#14532d;--cired:#f6dcdc;--ciredfg:#7f1d1d;
--ciamber:#f3e8cf;--ciamberfg:#713f12"""
DARK = """--bg:#15181b;--fg:#e6e9ec;--mut:#98a1ab;--line:#2c3238;--chip:#22272c;
--reject:#e08585;--chg:#d9b45e;--accept:#5fd39b;--prop:#b795e0;--take:#7aa8de;
--cigreen:#173b28;--cigreenfg:#86efac;--cired:#3f1d1d;--ciredfg:#fca5a5;
--ciamber:#3a2f14;--ciamberfg:#fde68a"""

page = f"""<title>rook-code-review — PR sweep {date}</title>
<style>
:root{{{TOKENS}}}
@media(prefers-color-scheme:dark){{:root:not([data-theme="light"]){{{DARK}}}}}
:root[data-theme="dark"]{{{DARK}}}
:root[data-theme="light"]{{{TOKENS}}}
body{{background:var(--bg);color:var(--fg);font:14px/1.45 system-ui,sans-serif;margin:2rem auto;max-width:84rem;padding:0 1rem}}
h1{{font-size:1.3rem}}h2{{font-size:1.05rem;color:var(--mut);margin-top:2rem}}p{{color:var(--mut)}}
.wrap{{overflow-x:auto}}table{{border-collapse:collapse;width:100%;font-variant-numeric:tabular-nums}}
th,td{{text-align:left;padding:.4rem .55rem;border-bottom:1px solid var(--line);vertical-align:top}}
th{{cursor:pointer;white-space:nowrap;color:var(--mut);font-weight:600}}
td.sum{{min-width:16rem;max-width:26rem}}td.who{{white-space:nowrap}}
tr td:first-child{{white-space:nowrap;font-weight:600}}a{{color:inherit}}
a.u{{font-weight:600;text-decoration:none}}a.u:hover{{text-decoration:underline}}
small{{color:var(--mut)}}
.chip{{display:inline-block;padding:.05rem .4rem;border-radius:.5rem;font-size:.82em;background:var(--chip);margin-right:.25rem}}
.chip.sev-b{{background:var(--reject);color:#fff}}.chip.sev-c{{background:var(--chg);color:#fff}}
.chip.sev-n{{background:var(--chip);color:var(--mut)}}.chip.sev-q{{background:var(--prop);color:#fff}}
.chip.none{{color:var(--accept)}}.chip.bp{{background:var(--accept);color:#fff}}
.chip.prop{{background:var(--prop);color:#fff}}.chip.take{{background:var(--take);color:#fff}}
.chip.bug-real{{background:var(--reject);color:#fff}}.chip.bug-fab{{background:var(--chg);color:#fff}}
.ci{{display:inline-block;padding:.1rem .45rem;border-radius:.5rem;font-size:.85em}}
.ci.green{{background:var(--cigreen);color:var(--cigreenfg)}}
.ci.red{{background:var(--cired);color:var(--ciredfg)}}
.ci.amber{{background:var(--ciamber);color:var(--ciamberfg)}}
tr.reject td:first-child{{box-shadow:inset 3px 0 var(--reject)}}
tr.chg td:first-child{{box-shadow:inset 3px 0 var(--chg)}}
tr.accept td:first-child{{box-shadow:inset 3px 0 var(--accept)}}
tr.prop td:first-child{{box-shadow:inset 3px 0 var(--prop)}}
tr.take td:first-child{{box-shadow:inset 3px 0 var(--take)}}
tr.skip td:first-child{{box-shadow:inset 3px 0 var(--skip)}}tr.skip td{{color:var(--mut)}}
tr.det td{{border-bottom:2px solid var(--line);padding-top:0}}
tr.det summary{{cursor:pointer;color:var(--mut)}}
table.inner{{margin:.5rem 0;width:auto}}table.inner th,table.inner td{{padding:.25rem .5rem;font-size:.92em}}
td.anch{{font-family:ui-monospace,monospace;font-size:.9em;white-space:nowrap}}
.st-posted{{color:var(--accept)}}.st-approved{{color:var(--take)}}.st-dropped{{color:var(--mut)}}
p.clean{{margin:.4rem 0;font-size:.92em}}
.legend span{{display:inline-block;margin-right:.7rem;padding:.15rem .55rem;border-radius:.6rem;background:var(--chip);cursor:pointer;user-select:none}}
.legend span.active{{outline:2px solid var(--fg)}}
.legend i{{display:inline-block;width:.6rem;height:.6rem;border-radius:50%;margin-right:.35rem}}
</style>
<h1>rook-code-review — PR sweep · {date}</h1>
<p>{len(prs)} PR(s) reviewed and verified{f", {len(skips)} skipped" if skips else ""}.
Verdicts and finding counts come from each PR's <code>findings.json</code> after verification;
CI cells are live passing/total from the phase-0 snapshot (hover for failing checks).
Expand a row for its verified findings and the audited-and-clean statement.
Click a legend chip to filter (click again to clear).</p>
<p class="legend" id="legend">
<span data-f="reject"><i style="background:var(--reject)"></i>reject</span>
<span data-f="chg"><i style="background:var(--chg)"></i>request changes</span>
<span data-f="accept"><i style="background:var(--accept)"></i>accept</span>
<span data-f="prop"><i style="background:var(--prop)"></i>proposal review required</span>
<span data-f="take"><i style="background:var(--take)"></i>takeover candidate</span></p>
<div class="wrap"><table id="t"><thead>
<tr><th>#</th><th>verdict</th><th>findings</th><th>title</th><th>CI</th><th>backport</th><th>author</th></tr>
</thead><tbody>
""" + "\n".join(body) + """
</tbody></table></div>
""" + (f"""<h2>Skipped</h2>
<div class="wrap"><table id="s"><tbody>
{skiprows}
</tbody></table></div>""" if skiprows else "") + """
<script>
document.querySelectorAll('#t th').forEach((h,i)=>h.onclick=()=>{
const tb=document.querySelector('#t tbody');const dir=h.dataset.d=-(+h.dataset.d||-1);
const pairs=[];const rows=[...tb.rows];
for(let j=0;j<rows.length;j+=2)pairs.push([rows[j],rows[j+1]]);
pairs.sort((a,b)=>dir*a[0].cells[i].innerText.localeCompare(b[0].cells[i].innerText,undefined,{numeric:true}))
.forEach(p=>p.forEach(r=>r&&tb.appendChild(r)));});
const legend=document.getElementById('legend');let active=null;
legend.querySelectorAll('span').forEach(sp=>sp.onclick=()=>{
const f=sp.dataset.f.split(' ');
if(active===sp){active=null;legend.querySelectorAll('span').forEach(x=>x.classList.remove('active'));
document.querySelectorAll('#t tbody tr').forEach(r=>r.style.display='');return;}
active=sp;legend.querySelectorAll('span').forEach(x=>x.classList.toggle('active',x===sp));
document.querySelectorAll('#t tbody tr').forEach(r=>{
r.style.display=f.some(c=>r.classList.contains(c))?'':'none';});});
</script>"""

open(f"{sw}/dashboard.html", "w").write(page)
counts = {}
for r in prs:
    counts[r.get("verdict", "—")] = counts.get(r.get("verdict", "—"), 0) + 1
print(f"{os.path.basename(sw)}: {len(prs)} PRs — "
      + ", ".join(f"{v} {k}" for k, v in sorted(counts.items()))
      + f"; {sum(len(live(r)) for r in prs)} live findings -> {sw}/dashboard.html")
