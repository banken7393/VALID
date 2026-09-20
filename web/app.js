/* VALID dashboard: polls /api/data every 1s (board + how-it-works.mmd). */

(function () {
  'use strict';

  const els = {
    feature: document.getElementById('feature'),
    lifecycle: document.getElementById('lifecycle'),
    mode: document.getElementById('mode'),
    autonomy: document.getElementById('autonomy'),
    gate: document.getElementById('gate'),
    isolation: document.getElementById('isolation'),
    northStar: document.getElementById('north-star'),
    acceptance: document.getElementById('acceptance'),
    phasesList: document.getElementById('phases-list'),
    diagram: document.getElementById('diagram'),
    tasksList: document.getElementById('tasks-list'),
    decisionsList: document.getElementById('decisions-list'),
    assumptionsList: document.getElementById('assumptions-list'),
    promotionsList: document.getElementById('promotions-list'),
    testsSummary: document.getElementById('tests-summary'),
    testsList: document.getElementById('tests-list'),
    findings: document.getElementById('findings'),
    output: document.getElementById('output'),
    error: document.getElementById('error'),
    liveStamp: document.getElementById('live-stamp'),
  };

  let lastDiagram = '';
  let lastUpdatedAt = '';
  let mermaidReady = null;

  function waitForMermaid() {
    if (mermaidReady) {
      return mermaidReady;
    }
    mermaidReady = new Promise((resolve) => {
      if (window.mermaid) {
        resolve(window.mermaid);
        return;
      }
      const started = Date.now();
      const timer = setInterval(() => {
        if (window.mermaid) {
          clearInterval(timer);
          resolve(window.mermaid);
        } else if (Date.now() - started > 15000) {
          clearInterval(timer);
          resolve(null);
        }
      }, 50);
    });
    return mermaidReady;
  }

  async function refresh() {
    try {
      const res = await fetch('/api/data', { cache: 'no-store' });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error || `HTTP ${res.status}`);
      }
      const data = await res.json();
      els.error.hidden = true;
      await render(data);
    } catch (err) {
      els.error.hidden = false;
      els.error.textContent = `Waiting for board: ${err.message}`;
      if (els.liveStamp) {
        els.liveStamp.textContent = 'Board unloadable — fix data.json (poll every 1s)';
      }
    }
  }

  function fillList(el, items, mapper) {
    el.innerHTML = '';
    (items || []).forEach((item) => {
      const li = document.createElement('li');
      const mapped = mapper(item);
      if (mapped && typeof mapped === 'object') {
        li.textContent = mapped.text;
        if (mapped.className) li.className = mapped.className;
      } else {
        li.textContent = mapped;
      }
      el.appendChild(li);
    });
    if (!(items || []).length) {
      const li = document.createElement('li');
      li.textContent = '—';
      el.appendChild(li);
    }
  }

  async function render(data) {
    els.feature.textContent = data.feature || '—';
    const life = data.lifecycle || '—';
    els.lifecycle.textContent = life;
    els.lifecycle.dataset.status = life;
    els.mode.textContent = data.mode || '—';
    const autonomy = data.autonomy || 'in_the_loop';
    els.autonomy.textContent = autonomy;
    els.autonomy.dataset.status = autonomy;

    const passed = !!data.audit?.passed;
    // Empty findings [] is truthy in JS; only show pass/fail after a real gate run (audit.at).
    const gateRan = !!data.audit?.at;
    els.gate.textContent = gateRan ? (passed ? 'passed' : 'failed') : '—';
    els.gate.dataset.passed = gateRan ? (passed ? 'true' : 'false') : '';

    const env = data.environment || {};
    els.isolation.textContent = env.isolation_warning ? 'warning' : (env.worktree_path ? 'quarantine' : '—');

    els.northStar.textContent = data.north_star || '—';
    fillList(els.acceptance, data.what, (item) => `${item.id}: ${item.description}`);
    fillList(els.phasesList, data.phases, (p) => `${p.order}. ${p.id}: ${p.title}`);
    fillList(els.tasksList, data.tasks, (t) => `${t.id} [${t.status}] ${t.title} covers=${(t.covers || []).join(',')}`);
    fillList(els.decisionsList, data.decisions, (d) => `${d.id}: ${d.title} — ${d.detail || ''}`);
    fillList(els.assumptionsList, data.assumptions, (a) => `${a.id}: ${a.detail}`);
    fillList(els.promotionsList, data.pending_promotions, (p) => `${p.id} [${p.kind}] ${p.description} applied=${!!p.applied}`);

    const tdd = data.tdd || {};
    els.testsSummary.textContent = `${tdd.passed ?? 0}/${tdd.total ?? 0} pass · ${tdd.failed ?? 0} fail · phase ${tdd.phase || '—'}`;
    fillList(els.testsList, tdd.cases, (c) => {
      const liClass = c.status === 'pass' ? 'test-pass' : c.status === 'fail' ? 'test-fail' : 'test-pending';
      return { text: `${c.name}: ${c.status}`, className: `test-case ${liClass}` };
    });

    els.findings.innerHTML = '';
    (data.audit?.findings || []).forEach((f) => {
      const li = document.createElement('li');
      li.textContent = `[${f.severity}] ${f.code}: ${f.message}`;
      els.findings.appendChild(li);
    });
    if (!(data.audit?.findings || []).length) {
      const li = document.createElement('li');
      li.textContent = '—';
      els.findings.appendChild(li);
    }

    els.output.textContent = tdd.output || 'No test output yet.';

    const updatedAt = data.updated_at || '';
    if (els.liveStamp) {
      const changed = updatedAt && updatedAt !== lastUpdatedAt;
      lastUpdatedAt = updatedAt;
      const when = updatedAt ? new Date(updatedAt).toLocaleTimeString() : '—';
      els.liveStamp.textContent = changed
        ? `Live · board updated ${when}`
        : `Live · polling · last board touch ${when}`;
    }

    const diagram = (data.how_it_works || '').trim();
    if (!diagram) {
      els.diagram.textContent = '—';
      lastDiagram = '';
      return;
    }
    if (diagram === lastDiagram) {
      return;
    }
    lastDiagram = diagram;
    await renderMermaid(diagram);
  }

  async function renderMermaid(source) {
    const mermaid = await waitForMermaid();
    els.diagram.innerHTML = '';
    if (!mermaid) {
      els.diagram.textContent = `Mermaid CDN not loaded yet.\n\n${source}`;
      return;
    }
    const id = `mmd-${Date.now()}`;
    try {
      const { svg } = await mermaid.render(id, source);
      els.diagram.innerHTML = svg;
    } catch (err) {
      els.diagram.textContent = `Mermaid error: ${err.message}\n\n${source}`;
    }
  }

  waitForMermaid().then(() => {
    refresh();
    setInterval(refresh, 1000);
  });
})();
