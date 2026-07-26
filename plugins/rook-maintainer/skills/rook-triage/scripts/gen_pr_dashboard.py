#!/usr/bin/env python3
"""rook-triage PR dashboard generator (format contract: references/reporting.md).

Reads ONLY canonical sweep-dir inputs — snapshot.json (live metadata via
sweep_prefetch.py: titles, labels, assignees, reviews, CI rollup),
batch-*.json (triager assessments), refs-types.json, skips.json — and
writes <sweep-dir>/dashboard.html. CI cells are ALWAYS passing/total from
the snapshot's statusCheckRollup summary, never parsed from agent prose;
the agent's why-plausible analysis rides in the tooltip.

Usage: python3 gen_pr_dashboard.py SWEEP_DIR
"""
import glob
import html
import json
import os
import re
import sys

sw = os.path.abspath(sys.argv[1])
date = os.path.basename(sw)[:10]

items = []
for b in sorted(glob.glob(f"{sw}/batch-*.json")):
    items += json.load(open(b))
items.sort(key=lambda i: i["number"])
snap = json.load(open(f"{sw}/snapshot.json"))["items"]
skips = json.load(open(f"{sw}/skips.json")) if os.path.exists(f"{sw}/skips.json") else []
reftypes = json.load(open(f"{sw}/refs-types.json")) if os.path.exists(f"{sw}/refs-types.json") else {}


def refs_cell(it):
    nums, seen = [], set()
    for x in (it.get("xlinks") or []) + (it.get("dups") or []):
        if isinstance(x, dict) and "number" in x and x["number"] not in seen:
            seen.add(x["number"])
            if reftypes.get(str(x["number"])) == "Issue":
                nums.append(x["number"])
    lis = "".join(f'<li><a href="https://github.com/rook/rook/issues/{n}">#{n}</a></li>' for n in nums)
    return f'<ul class="list">{lis}</ul>' if lis else "—"


def linkify(text):
    esc = html.escape(text)
    return re.sub(r"#?\b(1[3-8]\d{3})\b",
                  lambda m: f'<a href="https://github.com/rook/rook/issues/{m.group(1)}">#{m.group(1)}</a>', esc)


def cls(it):
    n = it.get("next", "")
    if it.get("skip"): return "wip"
    if "close" in n or "CLOSE CANDIDATE" in it.get("disposition", ""): return "close"
    if it.get("takeover"): return "take"
    if "request-reviewers" in n: return "route"
    if "merge" in n: return "ready"
    if any(k in n for k in ("comment", "rebase", "dup-link", "fill-template")): return "act"
    return "mon"


def clean_assessment(s):
    s = re.sub(r"^(green|red)\b[:x ]*\d*[:, ]*", "", s or "").strip()
    return "" if re.fullmatch(r"[\d/,:. ]*(passing|failing)?", s) else s


def ci_chip(it):
    ci = snap.get(str(it["number"]), {}).get("ci") or {}
    total, passing = ci.get("total", 0), ci.get("passing", 0)
    failing, pending = ci.get("failing", 0), ci.get("pending", 0)
    parts = []
    if total == 0:
        short, kind = "…", "amber"
        parts.append("no checks recorded")
    else:
        short = f"{passing}/{total}"
        kind = "red" if failing else ("amber" if pending else "green")
        parts.append(f"{passing}/{total} passing")
        if failing:
            names = ", ".join(ci.get("failed", [])[:6])
            more = len(ci.get("failed", [])) - 6
            parts.append(f"{failing} failing: {names}" + (f" +{more} more" if more > 0 else ""))
        if pending:
            parts.append(f"{pending} pending")
    assessment = clean_assessment(it.get("ci", ""))
    tip = " · ".join(parts) + (f" — assessment: {assessment}" if assessment else "")
    return f'<span class="ci {kind}" title="{html.escape(tip)}">{html.escape(short)}</span>'


STATE = {"APPROVED": ("ok", "✔", "approved"),
         "CHANGES_REQUESTED": ("chg", "±", "changes requested"),
         "COMMENTED": ("com", "💬", "commented"),
         "DISMISSED": ("dis", "◌", "review dismissed")}


def ulogin(login):
    e = html.escape(login)
    if login == "copilot-pull-request-reviewer":
        return '<a class="u" href="https://github.com/apps/copilot-pull-request-reviewer">Copilot</a>'
    return f'<a class="u" href="https://github.com/{e}">{e}</a>'


def rv_row(login, klass, icon, text, title=""):
    t = f' title="{html.escape(title)}"' if title else ""
    return (f'<div class="r"><span class="nm">{ulogin(login)}</span>'
            f'<span class="st {klass}"{t}><i>{icon}</i>{html.escape(text)}</span></div>')


def reviewers_cell(it):
    rv = snap.get(str(it["number"]), {}).get("reviews") or {"latest": [], "requested": []}
    latest = {r["login"]: r["state"] for r in rv["latest"]}
    requested = rv["requested"]
    rows = []
    for login in requested:
        note = "re-requested" if login in latest else "review requested"
        rows.append(rv_row(login, "pend", "●", "", note))
    for login, state in latest.items():
        if login in requested:
            continue
        k, icon, text = STATE.get(state, ("com", "·", state.lower()))
        rows.append(rv_row(login, k, icon, "", text))
    for p in (it.get("reviewers_proposed") or []):
        s = str(p)
        m = re.match(r"([\w.-]+)\s*(?:\((.*)\))?", s)
        login, note = (m.group(1), m.group(2) or "") if m else (s, "")
        rows.append(rv_row(login, "prop", "◇", "", "proposed reviewer" + (f" — {note}" if note else " (triage routing)")))
    return f'<div class="rv">{"".join(rows)}</div>' if rows else "—"


def ul(entries):
    lis = "".join(f"<li>{html.escape(e)}</li>" for e in entries) or "<li>—</li>"
    return f'<ul class="list">{lis}</ul>'


def assignees_of(n):
    logins = snap.get(str(n), {}).get("assignees", [])
    lis = "".join(f"<li>{ulogin(l)}</li>" for l in logins) or "<li>—</li>"
    return f'<ul class="list">{lis}</ul>'


assessed, wiprows = [], []
for it in items:
    n = it["number"]
    s = snap.get(str(n), {})
    row = (f'<tr class="{cls(it)}"><td><a href="https://github.com/rook/rook/pull/{n}">#{n}</a></td>'
           f'<td>{it.get("kind","")}</td><td class="sum">{html.escape(s.get("title", ""))}</td>'
           f'<td>{ci_chip(it)}</td>'
           f'<td><ul class="list"><li>{html.escape(it.get("next","") or "—")}</li></ul></td>'
           f'<td class="d">{linkify(it.get("disposition",""))}</td>'
           f'<td>{refs_cell(it)}</td>'
           f'<td>{assignees_of(n)}</td>'
           f'<td>{reviewers_cell(it)}</td><td>{ul(s.get("labels", []))}</td></tr>')
    (wiprows if it.get("skip") else assessed).append(row)

skiprows = "\n".join(
    f'<tr class="skip"><td><a href="https://github.com/rook/rook/pull/{s["number"]}">#{s["number"]}</a></td>'
    f'<td colspan="9">{html.escape(s["class"])} — {html.escape(s["author"])} — {linkify(s["title"])}</td></tr>'
    for s in skips)

page = f"""<title>rook-triage — PR sweep {date}</title>
<style>
:root{{--bg:#fff;--fg:#1a1d21;--mut:#67707b;--line:#e3e6ea;--chip:#f2f4f6;
--mon:#3d6ea8;--route:#2e7d55;--act:#8a6d1f;--close:#a04545;--take:#7a5ba6;--ready:#1d8a5f;--skip:#999;
--cigreen:#d9efe2;--cigreenfg:#14532d;--cired:#f6dcdc;--ciredfg:#7f1d1d;--ciamber:#f3e8cf;--ciamberfg:#713f12}}
@media(prefers-color-scheme:dark){{:root{{--bg:#15181b;--fg:#e6e9ec;--mut:#98a1ab;--line:#2c3238;--chip:#22272c;
--cigreen:#173b28;--cigreenfg:#86efac;--cired:#3f1d1d;--ciredfg:#fca5a5;--ciamber:#3a2f14;--ciamberfg:#fde68a}}}}
:root[data-theme="dark"]{{--bg:#15181b;--fg:#e6e9ec;--mut:#98a1ab;--line:#2c3238;--chip:#22272c;
--cigreen:#173b28;--cigreenfg:#86efac;--cired:#3f1d1d;--ciredfg:#fca5a5;--ciamber:#3a2f14;--ciamberfg:#fde68a}}
:root[data-theme="light"]{{--bg:#fff;--fg:#1a1d21;--mut:#67707b;--line:#e3e6ea;--chip:#f2f4f6;
--cigreen:#d9efe2;--cigreenfg:#14532d;--cired:#f6dcdc;--ciredfg:#7f1d1d;--ciamber:#f3e8cf;--ciamberfg:#713f12}}
body{{background:var(--bg);color:var(--fg);font:14px/1.45 system-ui,sans-serif;margin:2rem auto;max-width:84rem;padding:0 1rem}}
h1{{font-size:1.3rem}}h2{{font-size:1.05rem;color:var(--mut);margin-top:2rem}}p{{color:var(--mut)}}
.wrap{{overflow-x:auto}}table{{border-collapse:collapse;width:100%;font-variant-numeric:tabular-nums}}
th,td{{text-align:left;padding:.4rem .55rem;border-bottom:1px solid var(--line);vertical-align:top}}
th{{cursor:pointer;white-space:nowrap;color:var(--mut);font-weight:600}}
td.d{{min-width:22rem}}td.sum{{min-width:13rem;max-width:19rem}}tr td:first-child{{white-space:nowrap;font-weight:600}}a{{color:inherit}}
.ci{{display:inline-block;padding:.1rem .45rem;border-radius:.5rem;font-size:.85em}}
.ci.green{{background:var(--cigreen);color:var(--cigreenfg)}}.ci.green a{{color:var(--cigreenfg)}}
.ci.red{{background:var(--cired);color:var(--ciredfg)}}.ci.red a{{color:var(--ciredfg)}}
.ci.amber{{background:var(--ciamber);color:var(--ciamberfg)}}.ci.amber a{{color:var(--ciamberfg)}}
a.u{{font:inherit;font-weight:600;color:inherit;text-decoration:none}}
a.u:hover{{text-decoration:underline}}
.rv{{max-width:17rem}}
.rv .r{{display:flex;align-items:baseline;border-bottom:1px dotted var(--line);padding:.08rem 0}}
.rv .r:last-child{{border-bottom:0}}
.rv .nm{{min-width:7.5rem;white-space:nowrap}}
.rv .st{{border-left:1px solid var(--line);padding-left:.5rem;color:var(--mut);font-size:.9em;white-space:nowrap}}
.rv .st i{{font-style:normal;margin-right:.35rem}}
.rv .ok i{{color:#2da44e}}.rv .chg i{{color:#cf222e;font-weight:700}}
.rv .pend i{{color:#d29922}}.rv .prop i{{color:var(--mut)}}
@media(prefers-color-scheme:dark){{.rv .ok i{{color:#3fb950}}.rv .chg i{{color:#f85149}}.rv .pend i{{color:#d29922}}}}
.list{{margin:0;padding:0;list-style:none}}
.list li{{white-space:nowrap;padding:.05rem 0}}
tr.mon td:first-child{{box-shadow:inset 3px 0 var(--mon)}}
tr.route td:first-child{{box-shadow:inset 3px 0 var(--route)}}
tr.act td:first-child{{box-shadow:inset 3px 0 var(--act)}}
tr.close td:first-child{{box-shadow:inset 3px 0 var(--close)}}
tr.take td:first-child{{box-shadow:inset 3px 0 var(--take)}}
tr.ready td:first-child{{box-shadow:inset 3px 0 var(--ready)}}
tr.wip td:first-child,tr.skip td:first-child{{box-shadow:inset 3px 0 var(--skip)}}
tr.skip td{{color:var(--mut)}}
.legend span{{display:inline-block;margin-right:.7rem;padding:.15rem .55rem;border-radius:.6rem;background:var(--chip);cursor:pointer;user-select:none}}
.legend span.active{{outline:2px solid var(--fg)}}
.legend i{{display:inline-block;width:.6rem;height:.6rem;border-radius:50%;margin-right:.35rem}}
.cols span.off{{opacity:.45;text-decoration:line-through}}
</style>
<h1>rook-triage — open PRs · {date} · report-only</h1>
<p>{len(assessed)} assessed + {len(wiprows)} WIP signal-rows + {len(skips)} skipped (draft/bot).
Reviewer sets post-cap, every set ≥1 approver. Triage proposes no PR labels (rook convention); the labels
column shows what is currently on each PR. CI cells are live passing/total from the snapshot (hover for
failing checks and the triage assessment). Nothing posted. Click a legend chip to filter (click again to
clear); click a column chip to hide/show that column.</p>
<p class="legend" id="legend">
<span data-f="ready"><i style="background:var(--ready)"></i>ready-to-merge</span>
<span data-f="close"><i style="background:var(--close)"></i>close candidate</span>
<span data-f="take"><i style="background:var(--take)"></i>takeover</span>
<span data-f="route"><i style="background:var(--route)"></i>request reviewers</span>
<span data-f="act"><i style="background:var(--act)"></i>hygiene comment</span>
<span data-f="mon"><i style="background:var(--mon)"></i>monitor</span>
<span data-f="wip skip"><i style="background:var(--skip)"></i>WIP/skipped</span></p>
<p class="legend cols" id="cols">columns:
<span data-n="#">#</span><span data-n="kind">kind</span><span data-n="summary">summary</span><span data-n="CI">CI</span><span data-n="actions">actions</span><span data-n="disposition">disposition</span><span data-n="issue #">issue #</span><span data-n="assignees">assignees</span><span data-n="reviewers">reviewers</span><span data-n="labels">labels</span></p>
<div class="wrap"><table id="t"><thead>
<tr><th>#</th><th>kind</th><th>summary</th><th>CI</th><th>actions</th><th>disposition</th><th>issue #</th><th>assignees</th><th>reviewers</th><th>labels</th></tr>
</thead><tbody>
""" + "\n".join(assessed) + """
</tbody></table></div>
<h2>Skipped (assessed-signals WIP rows first, then draft/bot)</h2>
<div class="wrap"><table id="s"><tbody>
""" + "\n".join(wiprows) + "\n" + skiprows + """
</tbody></table></div>
<script>
document.querySelectorAll('#t th').forEach((h,i)=>h.onclick=()=>{
const tb=document.querySelector('#t tbody');const dir=h.dataset.d=-(+h.dataset.d||-1);
[...tb.rows].sort((a,b)=>dir*a.cells[i].innerText.localeCompare(b.cells[i].innerText,undefined,{numeric:true}))
.forEach(r=>tb.appendChild(r));});
const legend=document.getElementById('legend');let active=null;
legend.querySelectorAll('span').forEach(sp=>sp.onclick=()=>{
const f=sp.dataset.f.split(' ');
if(active===sp){active=null;legend.querySelectorAll('span').forEach(x=>x.classList.remove('active'));
document.querySelectorAll('#t tbody tr, #s tbody tr').forEach(r=>r.style.display='');return;}
active=sp;legend.querySelectorAll('span').forEach(x=>x.classList.toggle('active',x===sp));
document.querySelectorAll('#t tbody tr, #s tbody tr').forEach(r=>{
r.style.display=f.some(c=>r.classList.contains(c))?'':'none';});});
const COLNAMES=[...document.querySelectorAll('#t thead th')].map(h=>h.innerText.trim());
const CKEY='rook-triage-cols-prs';
let saved=[];try{saved=JSON.parse(localStorage.getItem(CKEY)||'[]')}catch(e){}
let hid=new Set(saved.filter(n=>COLNAMES.includes(n)));
function applyCols(){
document.querySelectorAll('#t thead th').forEach((h,i)=>h.style.display=hid.has(COLNAMES[i])?'none':'');
document.querySelectorAll('#t tbody tr, #s tbody tr').forEach(r=>{
if(r.cells.length===COLNAMES.length){COLNAMES.forEach((n,i)=>r.cells[i].style.display=hid.has(n)?'none':'');}
else if(r.cells.length===2&&r.cells[1].hasAttribute('colspan')){
r.cells[0].style.display=hid.has(COLNAMES[0])?'none':'';
r.cells[1].colSpan=Math.max(1,COLNAMES.slice(1).filter(n=>!hid.has(n)).length);}});
document.querySelectorAll('#cols span').forEach(sp=>sp.classList.toggle('off',hid.has(sp.dataset.n)));
try{localStorage.setItem(CKEY,JSON.stringify([...hid]))}catch(e){}}
document.querySelectorAll('#cols span').forEach(sp=>sp.onclick=()=>{
const n=sp.dataset.n;hid.has(n)?hid.delete(n):hid.add(n);applyCols();});
applyCols();
</script>"""
open(f"{sw}/dashboard.html", "w").write(page)
print(f"{os.path.basename(sw)}: {len(assessed)} assessed, {len(wiprows)} WIP, {len(skips)} skipped; CI/titles/labels/reviews from snapshot")
