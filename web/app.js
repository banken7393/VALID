/* VALID dashboard: polls /api/data every 1s (board v2). */

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
  };

  let lastDiagram = '';

  async function refresh() {
    try {
      const res = await fetch('/api/data', { cache: 'no-store' });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error || `HTTP ${res.status}`);
      }
      const data = await res.json();
      els.error.hidden = true;
      render(data);
    } catch (err) {
      els.error.hidden = false;
      els.error.textContent = `Waiting for board: ${err.message}`;
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

  function render(data) {
    els.feature.textContent = data.feature || '—';
    const life = data.lifecycle || '—';
    els.lifecycle.textContent = life;
    els.lifecycle.dataset.status = life;
    els.mode.textContent = data.mode || '—';
    const autonomy = data.autonomy || 'in_the_loop';
    els.autonomy.textContent = autonomy;
    els.autonomy.dataset.status = autonomy;

    const passed = !!data.audit?.passed;
    els.gate.textContent = data.audit?.findings ? (passed ? 'passed' : 'failed') : '—';
    els.gate.dataset.passed = passed ? 'true' : 'false';

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

    const diagram = (data.how_it_works || '').trim();
    if (diagram && diagram !== lastDiagram && window.mermaid) {
      lastDiagram = diagram;
      renderMermaid(diagram);
    }
  }

  async function renderMermaid(source) {
    els.diagram.innerHTML = '';
    const id = `mmd-${Date.now()}`;
    try {
      const { svg } = await window.mermaid.render(id, source);
      els.diagram.innerHTML = svg;
    } catch (err) {
      els.diagram.textContent = `Mermaid error: ${err.message}\n\n${source}`;
    }
  }

  refresh();
  setInterval(refresh, 1000);
})();
