package prdash

// The static HTML, CSS and JS is byte-identical to what gen_pr_dashboard.py
// emitted. The dashboard is republished to the same per-sweep URL, so cosmetic
// drift here is a visible change to a maintainer's working surface.
const pageTemplate = `<title>rook-triage — PR sweep {{.Date}}</title>
<style>
:root{--bg:#fff;--fg:#1a1d21;--mut:#67707b;--line:#e3e6ea;--chip:#f2f4f6;
--mon:#3d6ea8;--route:#2e7d55;--act:#8a6d1f;--close:#a04545;--take:#7a5ba6;--ready:#1d8a5f;--skip:#999;
--cigreen:#d9efe2;--cigreenfg:#14532d;--cired:#f6dcdc;--ciredfg:#7f1d1d;--ciamber:#f3e8cf;--ciamberfg:#713f12}
@media(prefers-color-scheme:dark){:root{--bg:#15181b;--fg:#e6e9ec;--mut:#98a1ab;--line:#2c3238;--chip:#22272c;
--cigreen:#173b28;--cigreenfg:#86efac;--cired:#3f1d1d;--ciredfg:#fca5a5;--ciamber:#3a2f14;--ciamberfg:#fde68a}}
:root[data-theme="dark"]{--bg:#15181b;--fg:#e6e9ec;--mut:#98a1ab;--line:#2c3238;--chip:#22272c;
--cigreen:#173b28;--cigreenfg:#86efac;--cired:#3f1d1d;--ciredfg:#fca5a5;--ciamber:#3a2f14;--ciamberfg:#fde68a}
:root[data-theme="light"]{--bg:#fff;--fg:#1a1d21;--mut:#67707b;--line:#e3e6ea;--chip:#f2f4f6;
--cigreen:#d9efe2;--cigreenfg:#14532d;--cired:#f6dcdc;--ciredfg:#7f1d1d;--ciamber:#f3e8cf;--ciamberfg:#713f12}
body{background:var(--bg);color:var(--fg);font:14px/1.45 system-ui,sans-serif;margin:2rem auto;max-width:84rem;padding:0 1rem}
h1{font-size:1.3rem}h2{font-size:1.05rem;color:var(--mut);margin-top:2rem}p{color:var(--mut)}
.wrap{overflow-x:auto}table{border-collapse:collapse;width:100%;font-variant-numeric:tabular-nums}
th,td{text-align:left;padding:.4rem .55rem;border-bottom:1px solid var(--line);vertical-align:top}
th{cursor:pointer;white-space:nowrap;color:var(--mut);font-weight:600}
td.d{min-width:22rem}td.sum{min-width:13rem;max-width:19rem}tr td:first-child{white-space:nowrap;font-weight:600}a{color:inherit}
.ci{display:inline-block;padding:.1rem .45rem;border-radius:.5rem;font-size:.85em}
.ci.green{background:var(--cigreen);color:var(--cigreenfg)}.ci.green a{color:var(--cigreenfg)}
.ci.red{background:var(--cired);color:var(--ciredfg)}.ci.red a{color:var(--ciredfg)}
.ci.amber{background:var(--ciamber);color:var(--ciamberfg)}.ci.amber a{color:var(--ciamberfg)}
a.u{font:inherit;font-weight:600;color:inherit;text-decoration:none}
a.u:hover{text-decoration:underline}
.rv{max-width:17rem}
.rv .r{display:flex;align-items:baseline;border-bottom:1px dotted var(--line);padding:.08rem 0}
.rv .r:last-child{border-bottom:0}
.rv .nm{min-width:7.5rem;white-space:nowrap}
.rv .st{border-left:1px solid var(--line);padding-left:.5rem;color:var(--mut);font-size:.9em;white-space:nowrap}
.rv .st i{font-style:normal;margin-right:.35rem}
.rv .ok i{color:#2da44e}.rv .chg i{color:#cf222e;font-weight:700}
.rv .pend i{color:#d29922}.rv .prop i{color:var(--mut)}
@media(prefers-color-scheme:dark){.rv .ok i{color:#3fb950}.rv .chg i{color:#f85149}.rv .pend i{color:#d29922}}
.list{margin:0;padding:0;list-style:none}
.list li{white-space:nowrap;padding:.05rem 0}
tr.mon td:first-child{box-shadow:inset 3px 0 var(--mon)}
tr.route td:first-child{box-shadow:inset 3px 0 var(--route)}
tr.act td:first-child{box-shadow:inset 3px 0 var(--act)}
tr.close td:first-child{box-shadow:inset 3px 0 var(--close)}
tr.take td:first-child{box-shadow:inset 3px 0 var(--take)}
tr.ready td:first-child{box-shadow:inset 3px 0 var(--ready)}
tr.wip td:first-child,tr.skip td:first-child{box-shadow:inset 3px 0 var(--skip)}
tr.skip td{color:var(--mut)}
.legend span{display:inline-block;margin-right:.7rem;padding:.15rem .55rem;border-radius:.6rem;background:var(--chip);cursor:pointer;user-select:none}
.legend span.active{outline:2px solid var(--fg)}
.legend i{display:inline-block;width:.6rem;height:.6rem;border-radius:50%;margin-right:.35rem}
.cols span.off{opacity:.45;text-decoration:line-through}
</style>
<h1>rook-triage — open PRs · {{.Date}} · report-only</h1>
<p>{{len .Assessed}} assessed + {{len .WIP}} WIP signal-rows + {{len .Skipped}} skipped (draft/bot).
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
{{template "rows" .Assessed}}
</tbody></table></div>
<h2>Skipped (assessed-signals WIP rows first, then draft/bot)</h2>
<div class="wrap"><table id="s"><tbody>
{{template "rows" .WIP}}
{{template "skiprows" .Skipped}}
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
</script>`

// Only the partials interpolate untrusted PR text, and every one of them does
// it through an action so html/template picks the escaper for the context.
const partialTemplates = `
{{define "rows"}}{{range $i, $r := .}}{{if $i}}
{{end}}{{template "row" $r}}{{end}}{{end}}

{{define "row"}}<tr class="{{.Class}}"><td><a href="https://github.com/rook/rook/pull/{{.Number}}">#{{.Number}}</a></td><td>{{.Kind}}</td><td class="sum">{{.Title}}</td><td>{{template "chip" .Chip}}</td><td><ul class="list"><li>{{.Next}}</li></ul></td><td class="d">{{template "prose" .Disposition}}</td><td>{{template "refs" .Refs}}</td><td>{{template "users" .Assignees}}</td><td>{{template "reviewers" .Reviewers}}</td><td>{{template "list" .Labels}}</td></tr>{{end}}

{{define "skiprows"}}{{range $i, $s := .}}{{if $i}}
{{end}}{{template "skiprow" $s}}{{end}}{{end}}

{{define "skiprow"}}<tr class="skip"><td><a href="https://github.com/rook/rook/pull/{{.Number}}">#{{.Number}}</a></td><td colspan="9">{{.Class}} — {{.Author}} — {{template "prose" .Title}}</td></tr>{{end}}

{{define "chip"}}<span class="ci {{.Kind}}" title="{{.Tip}}">{{.Short}}</span>{{end}}

{{define "prose"}}{{range .}}{{if .Ref}}<a href="https://github.com/rook/rook/issues/{{.Ref}}">#{{.Ref}}</a>{{else}}{{.Plain}}{{end}}{{end}}{{end}}

{{define "refs"}}{{if .}}<ul class="list">{{range .}}<li><a href="https://github.com/rook/rook/issues/{{.}}">#{{.}}</a></li>{{end}}</ul>{{else}}—{{end}}{{end}}

{{define "user"}}<a class="u" href="{{.Href}}">{{.Name}}</a>{{end}}

{{define "users"}}<ul class="list">{{if .}}{{range .}}<li>{{template "user" .}}</li>{{end}}{{else}}<li>—</li>{{end}}</ul>{{end}}

{{define "list"}}<ul class="list">{{if .}}{{range .}}<li>{{.}}</li>{{end}}{{else}}<li>—</li>{{end}}</ul>{{end}}

{{define "reviewers"}}{{if .}}<div class="rv">{{range .}}<div class="r"><span class="nm">{{template "user" .User}}</span><span class="st {{.Class}}"{{if .Title}} title="{{.Title}}"{{end}}><i>{{.Icon}}</i></span></div>{{end}}</div>{{else}}—{{end}}{{end}}
`
