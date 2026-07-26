#!/usr/bin/env python3
"""rook-triage issues dashboard generator (format contract: references/reporting.md).

Reads ONLY canonical sweep-dir inputs — snapshot.json (live metadata via
sweep_prefetch.py), batch-*.json (triager assessments), refs-types.json,
issues-mentions.json (mine_mentions.py output) — and writes
<sweep-dir>/dashboard.html.

Usage: python3 gen_issues_dashboard.py SWEEP_DIR
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
mentions = json.load(open(f"{sw}/issues-mentions.json")) if os.path.exists(f"{sw}/issues-mentions.json") else {}
reftypes = json.load(open(f"{sw}/refs-types.json")) if os.path.exists(f"{sw}/refs-types.json") else {}


def refs_cell(it):
    nums, seen = [], set()
    for x in (it.get("xlinks") or []) + (it.get("dups") or []):
        if isinstance(x, dict) and "number" in x and x["number"] not in seen:
            seen.add(x["number"])
            if reftypes.get(str(x["number"])) == "PullRequest":
                nums.append(x["number"])
    lis = "".join(f'<li><a href="https://github.com/rook/rook/pull/{n}">#{n}</a></li>' for n in nums)
    return f'<ul class="list">{lis}</ul>' if lis else "—"


def linkify(text):
    esc = html.escape(text)
    return re.sub(r"#?\b(1[0-8]\d{3}|[4-9]\d{3})\b",
                  lambda m: f'<a href="https://github.com/rook/rook/issues/{m.group(1)}">#{m.group(1)}</a>', esc)


def cls(d):
    dl = d.lower()
    if dl.startswith("needs-info"): return "info"
    if "close" in dl and ("candidate" in dl or "propose" in dl) or dl.startswith(("fixed-by-merged", "answered", "support", "adjudicated")): return "close"
    if "fix-open" in dl: return "fix"
    if "blocked-upstream" in dl or "upstream" in dl.split("—")[0]: return "up"
    return "keep"


def ul(entries, bold=False):
    fmt = (lambda e: f"<li><b>{html.escape(e)}</b></li>") if bold else (lambda e: f"<li>{html.escape(e)}</li>")
    lis = "".join(fmt(e) for e in entries) or "<li>—</li>"
    return f'<ul class="list">{lis}</ul>'


def labels_cell(it):
    cur = snap.get(str(it["number"]), {}).get("labels", [])
    prop = it.get("labels_proposed", []) or []
    out = f'<div class="rev"><div class="sub"><span class="k">current</span>{ul(cur)}</div>'
    if prop:
        out += f'<div class="sub"><span class="k">propose +</span>{ul(prop, bold=True)}</div>'
    return out + "</div>"


def ulogin(login):
    e = html.escape(login)
    if login == "copilot-pull-request-reviewer":
        return '<a class="u" href="https://github.com/apps/copilot-pull-request-reviewer">Copilot</a>'
    return f'<a class="u" href="https://github.com/{e}">{e}</a>'


def mentions_cell(it):
    n = str(it["number"])
    rows = []
    existing = mentions.get(n, [])
    for login in existing[:8]:
        rows.append(f'<div class="r"><span class="nm">{ulogin(login)}</span>'
                    f'<span class="st ment" title="mentioned in thread"><i>@</i></span></div>')
    if len(existing) > 8:
        rows.append(f'<div class="r"><span class="nm">…</span>'
                    f'<span class="st ment">+{len(existing)-8} more</span></div>')
    for x in (it.get("routing") or []):
        rows.append(f'<div class="r"><span class="nm">{ulogin(str(x))}</span>'
                    f'<span class="st prop" title="proposed @-mention (triage routing)"><i>◇</i></span></div>')
    return f'<div class="rv">{"".join(rows)}</div>' if rows else "—"


rows = []
for it in items:
    n = it["number"]
    s = snap.get(str(n), {})
    a_logins = s.get("assignees", [])
    assignees = '<ul class="list">' + ("".join(f"<li>{ulogin(l)}</li>" for l in a_logins) or "<li>—</li>") + "</ul>"
    acts = '<ul class="list">' + ("".join(f"<li>{html.escape(a)}</li>" for a in it.get("actions", [])) or "<li>—</li>") + "</ul>"
    rows.append(
        f'<tr class="{cls(it["disposition"])}"><td><a href="https://github.com/rook/rook/issues/{n}">#{n}</a></td>'
        f'<td>{it.get("kind","")}</td><td class="sum">{html.escape(s.get("title", ""))}</td>'
        f'<td>{acts}</td><td class="d">{linkify(it["disposition"])}</td>'
        f'<td>{refs_cell(it)}</td>'
        f'<td>{assignees}</td><td>{mentions_cell(it)}</td><td>{labels_cell(it)}</td></tr>')

page = f"""<title>rook-triage — issues sweep {date}</title>
<style>
:root{{--bg:#fff;--fg:#1a1d21;--mut:#67707b;--line:#e3e6ea;--chip:#f2f4f6;
--keep:#3d6ea8;--fix:#2e7d55;--up:#8a6d1f;--close:#a04545;--info:#7a5ba6}}
@media(prefers-color-scheme:dark){{:root{{--bg:#15181b;--fg:#e6e9ec;--mut:#98a1ab;--line:#2c3238;--chip:#22272c}}}}
:root[data-theme="dark"]{{--bg:#15181b;--fg:#e6e9ec;--mut:#98a1ab;--line:#2c3238;--chip:#22272c}}
:root[data-theme="light"]{{--bg:#fff;--fg:#1a1d21;--mut:#67707b;--line:#e3e6ea;--chip:#f2f4f6}}
body{{background:var(--bg);color:var(--fg);font:14px/1.45 system-ui,sans-serif;margin:2rem auto;max-width:84rem;padding:0 1rem}}
h1{{font-size:1.3rem}}p{{color:var(--mut)}}
.wrap{{overflow-x:auto}}table{{border-collapse:collapse;width:100%;font-variant-numeric:tabular-nums}}
th,td{{text-align:left;padding:.4rem .55rem;border-bottom:1px solid var(--line);vertical-align:top}}
th{{cursor:pointer;white-space:nowrap;color:var(--mut);font-weight:600}}
td.d{{min-width:24rem}}td.sum{{min-width:13rem;max-width:19rem}}tr td:first-child{{white-space:nowrap;font-weight:600}}a{{color:inherit}}
.rev .sub{{display:flex;gap:.4rem;align-items:baseline}}
.rev .k{{font-size:.72em;text-transform:uppercase;letter-spacing:.04em;color:var(--mut);min-width:4.6rem}}
.list{{margin:0;padding:0;list-style:none}}
.list li{{white-space:nowrap;padding:.05rem 0;border-bottom:1px dotted var(--line)}}
.list li:last-child{{border-bottom:0}}
a.u{{font:inherit;font-weight:600;color:inherit;text-decoration:none}}
a.u:hover{{text-decoration:underline}}
.rv{{max-width:17rem}}
.rv .r{{display:flex;align-items:baseline;border-bottom:1px dotted var(--line);padding:.08rem 0}}
.rv .r:last-child{{border-bottom:0}}
.rv .nm{{min-width:7.5rem;white-space:nowrap}}
.rv .st{{border-left:1px solid var(--line);padding-left:.5rem;color:var(--mut);font-size:.9em;white-space:nowrap}}
.rv .st i{{font-style:normal;margin-right:.35rem}}
tr.keep td:first-child{{box-shadow:inset 3px 0 var(--keep)}}
tr.fix td:first-child{{box-shadow:inset 3px 0 var(--fix)}}
tr.up td:first-child{{box-shadow:inset 3px 0 var(--up)}}
tr.close td:first-child{{box-shadow:inset 3px 0 var(--close)}}
tr.info td:first-child{{box-shadow:inset 3px 0 var(--info)}}
.legend span{{display:inline-block;margin-right:.7rem;padding:.15rem .55rem;border-radius:.6rem;background:var(--chip);cursor:pointer;user-select:none}}
.legend span.active{{outline:2px solid var(--fg)}}
.legend i{{display:inline-block;width:.6rem;height:.6rem;border-radius:50%;margin-right:.35rem}}
.cols span.off{{opacity:.45;text-decoration:line-through}}
</style>
<h1>rook-triage — all {len(rows)} open issues · {date} · report-only</h1>
<p>Advise-only sweep: nothing posted. Close-class rows carry their refutation verdicts in report.md.
Labels: current on the issue + proposed additions (issues are the only labeling surface — PRs are never labeled by triage).
Click a legend chip to filter (click again to clear); click a column chip to hide/show that column.</p>
<p class="legend" id="legend">
<span data-f="keep"><i style="background:var(--keep)"></i>keep-open</span>
<span data-f="fix"><i style="background:var(--fix)"></i>fix-PR open</span>
<span data-f="up"><i style="background:var(--up)"></i>blocked upstream</span>
<span data-f="close"><i style="background:var(--close)"></i>close/convert candidate</span>
<span data-f="info"><i style="background:var(--info)"></i>needs-info</span></p>
<p class="legend cols" id="cols">columns:
<span data-n="#">#</span><span data-n="kind">kind</span><span data-n="summary">summary</span><span data-n="actions">actions</span><span data-n="disposition">disposition</span><span data-n="pr #">pr #</span><span data-n="assignees">assignees</span><span data-n="mentions">mentions</span><span data-n="labels">labels</span></p>
<div class="wrap"><table id="t"><thead>
<tr><th>#</th><th>kind</th><th>summary</th><th>actions</th><th>disposition</th><th>pr #</th><th>assignees</th><th>mentions</th><th>labels</th></tr>
</thead><tbody>
""" + "\n".join(rows) + """
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
document.querySelectorAll('#t tbody tr').forEach(r=>r.style.display='');return;}
active=sp;legend.querySelectorAll('span').forEach(x=>x.classList.toggle('active',x===sp));
document.querySelectorAll('#t tbody tr').forEach(r=>{
r.style.display=f.some(c=>r.classList.contains(c))?'':'none';});});
const COLNAMES=[...document.querySelectorAll('#t thead th')].map(h=>h.innerText.trim());
const CKEY='rook-triage-cols-issues';
let saved=[];try{saved=JSON.parse(localStorage.getItem(CKEY)||'[]')}catch(e){}
let hid=new Set(saved.filter(n=>COLNAMES.includes(n)));
function applyCols(){
document.querySelectorAll('#t thead th').forEach((h,i)=>h.style.display=hid.has(COLNAMES[i])?'none':'');
document.querySelectorAll('#t tbody tr').forEach(r=>{
if(r.cells.length===COLNAMES.length)COLNAMES.forEach((n,i)=>r.cells[i].style.display=hid.has(n)?'none':'');});
document.querySelectorAll('#cols span').forEach(sp=>sp.classList.toggle('off',hid.has(sp.dataset.n)));
try{localStorage.setItem(CKEY,JSON.stringify([...hid]))}catch(e){}}
document.querySelectorAll('#cols span').forEach(sp=>sp.onclick=()=>{
const n=sp.dataset.n;hid.has(n)?hid.delete(n):hid.add(n);applyCols();});
applyCols();
</script>"""
open(f"{sw}/dashboard.html", "w").write(page)
print(f"{os.path.basename(sw)}: {len(rows)} rows; titles/labels/assignees from snapshot")
