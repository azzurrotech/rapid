package web

// dashboardHTML is published through song (silo "rapid") and served by atp at
// /rapid/. It is a baseline document: plain HTML/CSS/JS, no framework and no
// external request, so it works installed and offline like the rest of the
// platform.
const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light dark">
<title>RAPID — AI analysis</title>
<style>
  :root {
    color-scheme: light dark;
    --bg: #f6f7f9; --fg: #16181d; --card: #fff; --line: #e4e7ec;
    --accent: #0a84ff; --good: #12b76a; --bad: #f04438; --muted: #667085;
  }
  @media (prefers-color-scheme: dark) {
    :root { --bg: #101216; --fg: #e8eaee; --card: #191c21; --line: #2a2f36; --muted: #98a2b3; }
  }
  * { box-sizing: border-box; }
  body { margin: 0; font: 15px/1.5 system-ui, sans-serif; background: var(--bg); color: var(--fg); }
  header { padding: 1rem 1.4rem; border-bottom: 1px solid var(--line); background: var(--card); }
  header h1 { margin: 0; font-size: 1.2rem; }
  header p { margin: .2rem 0 0; color: var(--muted); }
  main { max-width: 1000px; margin: 0 auto; padding: 1.2rem; display: grid; gap: 1rem; }
  section { background: var(--card); border: 1px solid var(--line); border-radius: 10px; padding: .9rem 1.1rem; }
  h2 { margin: .2rem 0 .7rem; font-size: .9rem; text-transform: uppercase; letter-spacing: .05em; color: var(--accent); }
  table { width: 100%; border-collapse: collapse; font-size: .9rem; }
  th, td { text-align: left; padding: .35rem .5rem; border-bottom: 1px solid var(--line); vertical-align: top; }
  .kpis { display: grid; grid-template-columns: repeat(auto-fit, minmax(130px, 1fr)); gap: .7rem; }
  .kpi { border: 1px solid var(--line); border-radius: 8px; padding: .6rem .8rem; }
  .kpi b { display: block; font-size: 1.5rem; }
  .muted { color: var(--muted); font-size: .85rem; }
  .ok { color: var(--good); } .bad { color: var(--bad); }
  button { font: inherit; border: 1px solid var(--line); background: var(--card); color: inherit; border-radius: 8px; padding: .3rem .7rem; cursor: pointer; }
  code { background: var(--bg); border: 1px solid var(--line); border-radius: 6px; padding: .05rem .3rem; }
</style>
</head>
<body>
<header>
  <h1>RAPID</h1>
  <p>Azzurro AI analysis and wrapper generation — served by atp through song</p>
</header>
<main>
  <section>
    <h2>Platform</h2>
    <div class="kpis" id="health"><span class="muted">loading…</span></div>
    <p class="muted" id="integration"></p>
  </section>
  <section>
    <h2>Projects</h2>
    <table><thead><tr><th>Name</th><th>Language</th><th>Status</th><th>Description</th></tr></thead>
    <tbody id="projects"><tr><td colspan="4" class="muted">loading…</td></tr></tbody></table>
  </section>
  <section>
    <h2>Analyses</h2>
    <table><thead><tr><th>Project</th><th>Files</th><th>Complexity</th><th>Tokens</th><th>When</th></tr></thead>
    <tbody id="analyses"><tr><td colspan="5" class="muted">loading…</td></tr></tbody></table>
  </section>
  <section>
    <h2>Wrappers</h2>
    <table><thead><tr><th>Project</th><th>Template</th><th>File</th><th>Cost</th></tr></thead>
    <tbody id="wrappers"><tr><td colspan="4" class="muted">loading…</td></tr></tbody></table>
  </section>
  <section>
    <h2>Metrics</h2>
    <div class="kpis" id="metrics"><span class="muted">loading…</span></div>
    <p class="muted" id="metrics-detail"></p>
  </section>
  <section>
    <h2>Exports</h2>
    <p class="muted">Every record set downloads in pod's own JSON or CSV shape — the same writer the platform uses.</p>
    <p id="exports"></p>
  </section>
  <section>
    <h2>Shares</h2>
    <p class="muted">Share links are shepherd capability tokens and can be cancelled by value.</p>
    <p>
      <input id="share-token" placeholder="paste a share token" size="48">
      <button type="button" id="revoke">Revoke</button>
    </p>
    <p class="muted" id="revoke-result"></p>
  </section>
  <section>
    <h2>API</h2>
    <p class="muted">
      <code>GET /api/health</code>
      <code>GET /api/metrics</code>
      <code>GET /api/export?table=projects&amp;format=csv</code>
      <code>GET /api/projects</code>
      <code>POST /api/projects/analyze</code>
      <code>GET /api/wrappers</code>
      <code>POST /api/share</code>
      <code>POST /api/share/revoke</code>
    </p>
  </section>
</main>
<script src="/_song/viewkit.js" defer></script>
<script>
function el(tag, text, cls) {
  const n = document.createElement(tag);
  if (text != null) n.textContent = text;
  if (cls) n.className = cls;
  return n;
}
async function getJSON(path) {
  const res = await fetch(path, { headers: { accept: "application/json" } });
  if (!res.ok) throw new Error(path + ": " + res.status);
  return res.json();
}
function fillRows(tbody, rows, cols) {
  tbody.textContent = "";
  if (!rows || rows.length === 0) {
    const tr = el("tr");
    const td = el("td", "nothing recorded yet", "muted");
    td.colSpan = cols;
    tr.appendChild(td);
    tbody.appendChild(tr);
    return;
  }
  for (const row of rows) {
    const tr = el("tr");
    for (const cell of row) tr.appendChild(el("td", cell));
    tbody.appendChild(tr);
  }
}
async function load() {
  const SV = window.SongView || {};
  const timeAgo = SV.timeAgo || ((s) => (s ? new Date(s).toLocaleString() : ""));
  try {
    const health = await getJSON("/api/health");
    const box = document.getElementById("health");
    box.textContent = "";
    for (const [name, up] of Object.entries(health.components || {})) {
      const card = el("div", null, "kpi");
      card.appendChild(el("b", up ? "up" : "down", up ? "ok" : "bad"));
      card.appendChild(el("span", name, "muted"));
      box.appendChild(card);
    }
    const prag = document.getElementById("integration");
    prag.textContent = "status: " + health.status + " · atp " + (health.module_report ? health.module_report.version : "") + " · data: " + health.data_dir;
  } catch (err) {
    document.getElementById("health").textContent = err.message;
  }
  try {
    const m = await getJSON("/api/metrics");
    const box = document.getElementById("metrics");
    box.textContent = "";
    const kpi = (label, value) => {
      const card = el("div", null, "kpi");
      card.appendChild(el("b", value));
      card.appendChild(el("span", label, "muted"));
      box.appendChild(card);
    };
    kpi("uptime", Math.round(m.uptime_seconds / 60) + " min");
    const counts = m.counts || {};
    for (const name of ["projects", "analyses", "wrappers"]) kpi(name, counts[name] == null ? "—" : counts[name]);
    const p = m.pricing || {};
    kpi("revenue", "$" + Number(p.total_revenue || 0).toFixed(4));
    if (p.total_transactions != null) {
      document.getElementById("metrics-detail").textContent =
        p.total_transactions + " pricing transaction(s), average $" + Number(p.average_cost || 0).toFixed(4);
    }
  } catch (err) {
    document.getElementById("metrics").textContent = err.message;
  }
  const exports = document.getElementById("exports");
  for (const table of ["projects", "analyses", "wrappers"]) {
    for (const format of ["json", "csv"]) {
      const a = el("a", table + " · " + format.toUpperCase());
      a.href = "/api/export?table=" + table + "&format=" + format;
      a.download = "";
      exports.appendChild(a);
      exports.appendChild(document.createTextNode("   "));
    }
  }
  document.getElementById("revoke").addEventListener("click", async () => {
    const token = document.getElementById("share-token").value.trim();
    const out = document.getElementById("revoke-result");
    if (!token) { out.textContent = "paste a token first"; return; }
    try {
      const res = await fetch("/api/share/revoke", {
        method: "POST",
        headers: { "content-type": "application/json", accept: "application/json" },
        body: JSON.stringify({ token: token }),
      });
      const body = await res.json();
      out.textContent = res.ok ? "revoked " + body.kind + " " + body.id : body.error || ("failed: " + res.status);
    } catch (err) {
      out.textContent = err.message;
    }
  });
  try {
    const projects = await getJSON("/api/projects");
    fillRows(document.getElementById("projects"), projects.map((p) => [p.name, p.language || "—", p.status || "—", p.description || ""]), 4);
  } catch (err) { fillRows(document.getElementById("projects"), [], 4); }
  try {
    const analyses = await getJSON("/api/analyses");
    fillRows(document.getElementById("analyses"), analyses.map((a) => [a.project_name, a.total_files, a.complexity.toFixed(2), a.tokens, timeAgo(a.analyzed_at)]), 5);
  } catch (err) { fillRows(document.getElementById("analyses"), [], 5); }
  try {
    const wrappers = await getJSON("/api/wrappers");
    fillRows(document.getElementById("wrappers"), wrappers.map((w) => [w.project_name, w.template_type, w.file_path, "$" + w.cost.toFixed(4)]), 4);
  } catch (err) { fillRows(document.getElementById("wrappers"), [], 4); }
}
load();
</script>
</body>
</html>
`
