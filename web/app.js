/* VALID dashboard: polls /api/data every 1s (board + how-it-works.mmd + theme). */

(function () {
  'use strict';

  const VALUE_BASE = 'mt-1.5 text-lg font-semibold';

  const MERMAID_FLOW = {
    curve: 'basis',
    padding: 12,
    htmlLabels: true,
    nodeSpacing: 36,
    rankSpacing: 40,
  };

  const MERMAID_THEMES = {
    dark: {
      darkMode: true,
      background: '#222222',
      fontFamily: 'IBM Plex Sans, Segoe UI, sans-serif',
      fontSize: '14px',
      primaryColor: '#2c2c2c',
      primaryTextColor: '#e8e8e8',
      primaryBorderColor: '#4a4a4a',
      secondaryColor: '#262626',
      secondaryTextColor: '#e8e8e8',
      secondaryBorderColor: '#404040',
      tertiaryColor: '#1e1e1e',
      tertiaryTextColor: '#c8c8c8',
      tertiaryBorderColor: '#3a3a3a',
      lineColor: '#6e6e6e',
      textColor: '#e8e8e8',
      mainBkg: '#2c2c2c',
      nodeBkg: '#2c2c2c',
      nodeBorder: '#4a4a4a',
      clusterBkg: '#1e1e1e',
      clusterBorder: '#3a3a3a',
      titleColor: '#e8e8e8',
      edgeLabelBackground: '#222222',
      actorBkg: '#2c2c2c',
      actorBorder: '#4a4a4a',
      actorTextColor: '#e8e8e8',
      labelBoxBkgColor: '#2c2c2c',
      labelBoxBorderColor: '#4a4a4a',
      labelTextColor: '#e8e8e8',
      noteBkgColor: '#2a2a2a',
      noteTextColor: '#d0d0d0',
      noteBorderColor: '#454545',
    },
    light: {
      darkMode: false,
      background: '#ffffff',
      fontFamily: 'IBM Plex Sans, Segoe UI, sans-serif',
      fontSize: '14px',
      primaryColor: '#f0f0f2',
      primaryTextColor: '#1c1c1e',
      primaryBorderColor: '#c8c8ce',
      secondaryColor: '#e8e8ec',
      secondaryTextColor: '#1c1c1e',
      secondaryBorderColor: '#c0c0c6',
      tertiaryColor: '#f7f7f8',
      tertiaryTextColor: '#3a3a3e',
      tertiaryBorderColor: '#d0d0d6',
      lineColor: '#8a8a90',
      textColor: '#1c1c1e',
      mainBkg: '#f0f0f2',
      nodeBkg: '#f0f0f2',
      nodeBorder: '#c8c8ce',
      clusterBkg: '#f7f7f8',
      clusterBorder: '#d0d0d6',
      titleColor: '#1c1c1e',
      edgeLabelBackground: '#ffffff',
      actorBkg: '#f0f0f2',
      actorBorder: '#c8c8ce',
      actorTextColor: '#1c1c1e',
      labelBoxBkgColor: '#f0f0f2',
      labelBoxBorderColor: '#c8c8ce',
      labelTextColor: '#1c1c1e',
      noteBkgColor: '#f5f5f7',
      noteTextColor: '#3a3a3e',
      noteBorderColor: '#d0d0d6',
    },
  };

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
    diagramViewport: document.getElementById('diagram-viewport'),
    zoomIn: document.getElementById('diagram-zoom-in'),
    zoomOut: document.getElementById('diagram-zoom-out'),
    zoomFit: document.getElementById('diagram-zoom-fit'),
    zoomReset: document.getElementById('diagram-zoom-reset'),
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
  let currentTheme = 'dark';
  let mermaidReady = null;
  let mermaidConfiguredFor = '';

  // Diagram pan/zoom state (applied to #diagram inside #diagram-viewport).
  const zoom = {
    scale: 1,
    tx: 0,
    ty: 0,
    min: 0.25,
    max: 4,
    dragging: false,
    lastX: 0,
    lastY: 0,
  };

  function applyDiagramTransform() {
    if (!els.diagram) {
      return;
    }
    els.diagram.style.transform = `translate(${zoom.tx}px, ${zoom.ty}px) scale(${zoom.scale})`;
  }

  function resetDiagramView() {
    zoom.scale = 1;
    zoom.tx = 0;
    zoom.ty = 0;
    applyDiagramTransform();
  }

  function clampScale(s) {
    return Math.min(zoom.max, Math.max(zoom.min, s));
  }

  function zoomAt(clientX, clientY, nextScale) {
    const vp = els.diagramViewport;
    if (!vp) {
      return;
    }
    const rect = vp.getBoundingClientRect();
    const x = clientX - rect.left;
    const y = clientY - rect.top;
    const prev = zoom.scale;
    const scale = clampScale(nextScale);
    // Keep the point under the cursor stable while scaling.
    zoom.tx = x - ((x - zoom.tx) * scale) / prev;
    zoom.ty = y - ((y - zoom.ty) * scale) / prev;
    zoom.scale = scale;
    applyDiagramTransform();
  }

  function zoomBy(factor) {
    const vp = els.diagramViewport;
    if (!vp) {
      return;
    }
    const rect = vp.getBoundingClientRect();
    zoomAt(rect.left + rect.width / 2, rect.top + rect.height / 2, zoom.scale * factor);
  }

  function fitDiagram() {
    const vp = els.diagramViewport;
    const svg = els.diagram && els.diagram.querySelector('svg');
    if (!vp || !svg) {
      resetDiagramView();
      return;
    }
    const pad = 24;
    const vw = Math.max(1, vp.clientWidth - pad * 2);
    const vh = Math.max(1, vp.clientHeight - pad * 2);
    // Prefer intrinsic SVG size; fall back to bounding box.
    let sw = Number(svg.viewBox.baseVal && svg.viewBox.baseVal.width) || 0;
    let sh = Number(svg.viewBox.baseVal && svg.viewBox.baseVal.height) || 0;
    if (!sw || !sh) {
      const bb = svg.getBBox();
      sw = bb.width || svg.clientWidth || 1;
      sh = bb.height || svg.clientHeight || 1;
    }
    const scale = clampScale(Math.min(vw / sw, vh / sh, 1));
    zoom.scale = scale;
    zoom.tx = (vp.clientWidth - sw * scale) / 2;
    zoom.ty = (vp.clientHeight - sh * scale) / 2;
    applyDiagramTransform();
  }

  function bindDiagramControls() {
    const vp = els.diagramViewport;
    if (!vp || vp.dataset.bound === '1') {
      return;
    }
    vp.dataset.bound = '1';

    vp.addEventListener('wheel', (ev) => {
      ev.preventDefault();
      const factor = ev.deltaY < 0 ? 1.12 : 1 / 1.12;
      zoomAt(ev.clientX, ev.clientY, zoom.scale * factor);
    }, { passive: false });

    vp.addEventListener('pointerdown', (ev) => {
      if (ev.button !== 0) {
        return;
      }
      zoom.dragging = true;
      zoom.lastX = ev.clientX;
      zoom.lastY = ev.clientY;
      vp.setPointerCapture(ev.pointerId);
    });
    vp.addEventListener('pointermove', (ev) => {
      if (!zoom.dragging) {
        return;
      }
      zoom.tx += ev.clientX - zoom.lastX;
      zoom.ty += ev.clientY - zoom.lastY;
      zoom.lastX = ev.clientX;
      zoom.lastY = ev.clientY;
      applyDiagramTransform();
    });
    const endDrag = (ev) => {
      zoom.dragging = false;
      try {
        vp.releasePointerCapture(ev.pointerId);
      } catch (_) {
        /* ignore */
      }
    };
    vp.addEventListener('pointerup', endDrag);
    vp.addEventListener('pointercancel', endDrag);
    vp.addEventListener('dblclick', (ev) => {
      ev.preventDefault();
      resetDiagramView();
    });

    if (els.zoomIn) {
      els.zoomIn.addEventListener('click', () => zoomBy(1.2));
    }
    if (els.zoomOut) {
      els.zoomOut.addEventListener('click', () => zoomBy(1 / 1.2));
    }
    if (els.zoomFit) {
      els.zoomFit.addEventListener('click', () => fitDiagram());
    }
    if (els.zoomReset) {
      els.zoomReset.addEventListener('click', () => resetDiagramView());
    }
  }

  function normalizeTheme(theme) {
    return theme === 'light' ? 'light' : 'dark';
  }

  function applyTheme(theme) {
    const next = normalizeTheme(theme);
    if (next === currentTheme && document.documentElement.dataset.theme === next) {
      return false;
    }
    currentTheme = next;
    document.documentElement.dataset.theme = next;
    // Force Mermaid re-init + diagram redraw on theme change.
    mermaidConfiguredFor = '';
    lastDiagram = '';
    return true;
  }

  function configureMermaid(mermaid, theme) {
    const t = normalizeTheme(theme);
    if (mermaidConfiguredFor === t) {
      return;
    }
    mermaid.initialize({
      startOnLoad: false,
      theme: 'base',
      securityLevel: 'loose',
      flowchart: MERMAID_FLOW,
      themeVariables: MERMAID_THEMES[t],
    });
    mermaidConfiguredFor = t;
  }

  function toneClass(kind) {
    switch (kind) {
      case 'accent':
        return 'text-accent';
      case 'progress':
        return 'text-progress';
      case 'warn':
        return 'text-warn';
      case 'fail':
        return 'text-fail';
      case 'mute':
        return 'text-mute';
      default:
        return 'text-ink';
    }
  }

  function lifecycleTone(life) {
    const v = String(life || '').toLowerCase();
    if (['done', 'finish', 'merged', 'green'].includes(v)) {
      return 'accent';
    }
    if (v === 'build') {
      return 'progress';
    }
    if (v === 'audit') {
      return 'warn';
    }
    if (['plan', 'spec'].includes(v)) {
      return 'warn';
    }
    if (v === 'failed') {
      return 'fail';
    }
    if (v === 'above_the_loop') {
      return 'progress';
    }
    if (v === 'in_the_loop') {
      return 'warn';
    }
    return 'ink';
  }

  function modeTone(mode) {
    const v = String(mode || '').toLowerCase();
    if (v === 'feature') {
      return 'ink';
    }
    if (v === 'patch') {
      return 'progress';
    }
    if (v === 'minipatch') {
      return 'warn';
    }
    return 'ink';
  }

  function gateTone(gateRan, passed) {
    if (!gateRan) {
      return 'mute';
    }
    return passed ? 'accent' : 'fail';
  }

  function isolationTone(env) {
    if (env.isolation_warning) {
      return 'warn';
    }
    if (env.worktree_path) {
      return 'progress';
    }
    return 'mute';
  }

  function testsSummaryTone(tdd) {
    const failed = Number(tdd.failed) || 0;
    const passed = Number(tdd.passed) || 0;
    const total = Number(tdd.total) || 0;
    const phase = String(tdd.phase || '').toLowerCase();
    if (failed > 0 || phase === 'red') {
      return 'fail';
    }
    if (total > 0 && passed === total) {
      return 'accent';
    }
    if (phase === 'refactor') {
      return 'progress';
    }
    if (phase === 'green') {
      return 'accent';
    }
    if (total > 0 || phase) {
      return 'warn';
    }
    return 'mute';
  }

  function findingTone(severity) {
    const v = String(severity || '').toLowerCase();
    if (v === 'error' || v === 'fail' || v === 'critical') {
      return 'fail';
    }
    if (v === 'warning' || v === 'warn') {
      return 'warn';
    }
    return 'mute';
  }

  function phaseStatusClass(status) {
    switch ((status || '').toLowerCase()) {
      case 'done':
        return 'text-accent';
      case 'doing':
      case 'in_progress':
        return 'text-progress';
      case 'agreed':
        return 'text-accent';
      case 'pending':
        return 'text-warn';
      case 'failed':
        return 'text-fail';
      default:
        return 'text-mute';
    }
  }

  /** AC id colour from task coverage: uncovered / in-flight / done / failed. */
  function acCoverageClass(acId, tasks) {
    const covering = (tasks || []).filter((t) => (t.covers || []).includes(acId));
    if (!covering.length) {
      return 'text-fail';
    }
    if (covering.some((t) => (t.status || '').toLowerCase() === 'failed')) {
      return 'text-fail';
    }
    if (covering.some((t) => (t.status || '').toLowerCase() === 'done')) {
      return 'text-accent';
    }
    if (covering.some((t) => (t.status || '').toLowerCase() === 'doing')) {
      return 'text-progress';
    }
    return 'text-warn';
  }

  function setValueTone(el, tone) {
    if (!el) {
      return;
    }
    el.className = [VALUE_BASE, toneClass(tone)].join(' ');
  }

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
      const payload = await res.json().catch(() => ({}));
      if (!res.ok) {
        if (payload.theme) {
          applyTheme(payload.theme);
        }
        throw new Error(payload.error || `HTTP ${res.status}`);
      }
      // Envelope: { theme, board } (board fields are never top-level).
      const theme = payload.theme || 'dark';
      const data = payload.board || payload;
      applyTheme(theme);
      els.error.hidden = true;
      if (els.liveStamp) {
        els.liveStamp.className = 'mt-2 font-mono text-xs text-mute';
      }
      await render(data);
    } catch (err) {
      els.error.hidden = false;
      els.error.textContent = `Waiting for board: ${err.message}`;
      if (els.liveStamp) {
        els.liveStamp.className = 'mt-2 font-mono text-xs text-fail';
        els.liveStamp.textContent = `Board unloadable: ${err.message}`;
      }
    }
  }

  function taskStatusClass(status) {
    switch ((status || '').toLowerCase()) {
      case 'done':
        return 'text-accent';
      case 'doing':
        return 'text-progress';
      case 'failed':
        return 'text-fail';
      case 'pending':
        return 'text-warn';
      default:
        return 'text-mute';
    }
  }

  function renderTasks(el, tasks, phases) {
    el.innerHTML = '';
    const list = tasks || [];
    if (!list.length) {
      const empty = document.createElement('li');
      empty.textContent = '—';
      el.appendChild(empty);
      return;
    }

    const phaseMeta = new Map();
    (phases || []).forEach((p, i) => {
      if (!p || !p.id) {
        return;
      }
      phaseMeta.set(p.id, {
        id: p.id,
        name: (p.name || p.title || '').trim(),
        n: p.order > 0 ? p.order : i + 1,
      });
    });

    const byPhase = new Map();
    const loose = [];
    list.forEach((t) => {
      const pid = (t.phase || t.phase_id || t.phaseId || '').trim();
      if (pid && phaseMeta.has(pid)) {
        if (!byPhase.has(pid)) {
          byPhase.set(pid, []);
        }
        byPhase.get(pid).push(t);
      } else {
        loose.push(t);
      }
    });

    function appendHeading(text) {
      const li = document.createElement('li');
      li.className = 'mb-2 mt-5 first:mt-0';
      const h = document.createElement('div');
      h.className = 'font-mono text-xs uppercase tracking-[0.08em] text-mute';
      h.textContent = text;
      li.appendChild(h);
      el.appendChild(li);
    }

    function appendTask(t) {
      const li = document.createElement('li');
      li.className = 'mb-3';

      const status = (t.status || 'pending').trim();
      const head = document.createElement('div');
      head.className = 'flex flex-wrap items-baseline gap-x-2 gap-y-0.5 font-medium text-ink';

      const badge = document.createElement('span');
      badge.className = `font-mono text-sm ${taskStatusClass(status)}`;
      badge.textContent = `[${status}]`;

      const title = document.createElement('span');
      const label = (t.title || '').trim() || '—';
      title.textContent = `${t.id || '—'}: ${label}`;

      head.appendChild(badge);
      head.appendChild(title);
      li.appendChild(head);

      const meta = document.createElement('div');
      meta.className = 'mt-1 font-mono text-[0.85rem] text-mute';
      const covers = (t.covers || []).filter(Boolean);
      const bits = [];
      const pid = (t.phase || t.phase_id || t.phaseId || '').trim();
      if (pid) {
        const ph = phaseMeta.get(pid);
        bits.push(ph && ph.name ? `phase ${ph.id} · ${ph.name}` : `phase ${pid}`);
      }
      bits.push(covers.length ? `covers ${covers.join(', ')}` : 'covers —');
      meta.textContent = bits.join(' · ');
      li.appendChild(meta);

      const desc = (t.description || '').trim();
      if (desc) {
        const d = document.createElement('div');
        d.className = 'mt-1 text-[0.95rem] leading-relaxed text-mute';
        d.textContent = desc;
        li.appendChild(d);
      }
      el.appendChild(li);
    }

    let grouped = false;
    (phases || []).forEach((p, i) => {
      if (!p || !p.id || !byPhase.has(p.id)) {
        return;
      }
      grouped = true;
      const n = p.order > 0 ? p.order : i + 1;
      const name = (p.name || p.title || '').trim();
      appendHeading(`${n}. ${p.id}${name ? ` · ${name}` : ''}`);
      byPhase.get(p.id).forEach(appendTask);
    });

    if (loose.length) {
      if (grouped) {
        appendHeading('Ungrouped');
      }
      loose.forEach(appendTask);
    }
  }

  function fillList(el, items, mapper) {
    el.innerHTML = '';
    (items || []).forEach((item, index) => {
      const li = document.createElement('li');
      const mapped = mapper(item, index);
      if (mapped && typeof mapped === 'object') {
        if (mapped.className) {
          li.className = mapped.className;
        }
        if (Array.isArray(mapped.children)) {
          mapped.children.forEach((child) => {
            const node = document.createElement(child.tag || 'div');
            if (child.className) {
              node.className = child.className;
            }
            node.textContent = child.text || '';
            li.appendChild(node);
          });
        } else {
          li.textContent = mapped.text || '';
        }
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

  function renderPhases(el, phases) {
    el.innerHTML = '';
    const list = phases || [];
    if (!list.length) {
      const empty = document.createElement('li');
      empty.textContent = '—';
      el.appendChild(empty);
      return;
    }
    list.forEach((p, i) => {
      const li = document.createElement('li');
      li.className = 'mb-3';

      const id = p.id || '—';
      const name = (p.name || p.title || '').trim();
      const status = (p.status || '').trim();
      const outcome = (p.outcome || '').trim();
      const n = p.order > 0 ? p.order : i + 1;

      const head = document.createElement('div');
      head.className = 'flex flex-wrap items-baseline gap-x-2 gap-y-0.5 font-medium text-ink';

      if (status) {
        const badge = document.createElement('span');
        badge.className = `font-mono text-sm ${phaseStatusClass(status)}`;
        badge.textContent = `[${status}]`;
        head.appendChild(badge);
      }

      const title = document.createElement('span');
      title.textContent = name ? `${n}. ${id}: ${name}` : `${n}. ${id}`;
      head.appendChild(title);
      li.appendChild(head);

      if (outcome) {
        const out = document.createElement('div');
        out.className = 'mt-1 text-[0.95rem] leading-relaxed text-mute';
        out.textContent = outcome;
        li.appendChild(out);
      }
      el.appendChild(li);
    });
  }

  function renderFindings(el, findings) {
    el.innerHTML = '';
    const list = findings || [];
    if (!list.length) {
      const empty = document.createElement('li');
      empty.textContent = '—';
      el.appendChild(empty);
      return;
    }
    list.forEach((f) => {
      const li = document.createElement('li');
      li.className = 'mb-1.5';
      const sev = (f.severity || 'info').toLowerCase();
      const badge = document.createElement('span');
      badge.className = `font-mono text-sm ${toneClass(findingTone(sev))}`;
      badge.textContent = `[${sev}]`;
      const rest = document.createElement('span');
      rest.className = 'text-mute';
      rest.textContent = ` ${f.code || '—'}: ${f.message || ''}`;
      li.appendChild(badge);
      li.appendChild(rest);
      el.appendChild(li);
    });
  }

  async function render(data) {
    els.feature.textContent = data.feature || '—';
    setValueTone(els.feature, 'ink');

    const life = data.lifecycle || '—';
    els.lifecycle.textContent = life;
    setValueTone(els.lifecycle, lifecycleTone(life));

    const mode = data.mode || '—';
    els.mode.textContent = mode;
    setValueTone(els.mode, modeTone(mode));

    const autonomy = data.autonomy || 'in_the_loop';
    els.autonomy.textContent = autonomy;
    setValueTone(els.autonomy, lifecycleTone(autonomy));

    const passed = !!data.audit?.passed;
    const gateRan = !!(data.audit?.at);
    // Ignore zero-time sentinel from empty boards.
    const gateAt = data.audit?.at ? String(data.audit.at) : '';
    const gateMeaningful = gateRan && gateAt && !gateAt.startsWith('0001-01-01');
    els.gate.textContent = gateMeaningful ? (passed ? 'passed' : 'failed') : '—';
    setValueTone(els.gate, gateTone(gateMeaningful, passed));

    const env = data.environment || {};
    els.isolation.textContent = env.isolation_warning
      ? 'warning'
      : (env.worktree_path ? 'quarantine' : '—');
    setValueTone(els.isolation, isolationTone(env));

    els.northStar.textContent = data.north_star || '—';
    fillList(els.acceptance, data.what, (item) => {
      const id = item.id || '—';
      const desc = item.description || '';
      return {
        className: 'mb-1.5 flex flex-wrap items-baseline gap-x-2',
        children: [
          { tag: 'span', className: `font-mono text-sm font-medium ${acCoverageClass(id, data.tasks)}`, text: id },
          { tag: 'span', className: 'text-mute', text: desc },
        ],
      };
    });
    renderPhases(els.phasesList, data.phases);
    renderTasks(els.tasksList, data.tasks, data.phases);
    fillList(els.decisionsList, data.decisions, (d) => {
      const id = d.id || '—';
      const title = (d.title || d.question || d.text || '').trim();
      let detail = (d.detail || d.choice || '').trim();
      const when = (d.decided_at || '').trim();
      if (when && detail && !detail.includes(when)) {
        detail = `${detail} · ${when}`;
      } else if (when && !detail) {
        detail = when;
      }
      const head = title ? `${id}: ${title}` : id;
      if (!detail) {
        return {
          className: 'mb-3',
          children: [{ tag: 'div', className: 'font-medium text-ink', text: head }],
        };
      }
      return {
        className: 'mb-3',
        children: [
          { tag: 'div', className: 'font-medium text-ink', text: head },
          { tag: 'div', className: 'mt-1 text-[0.95rem] leading-relaxed text-mute', text: detail },
        ],
      };
    });
    fillList(els.assumptionsList, data.assumptions, (a) => ({
      className: 'mb-1.5',
      children: [
        { tag: 'span', className: 'font-mono text-sm text-warn', text: a.id || '—' },
        { tag: 'span', className: 'text-mute', text: `: ${a.detail || ''}` },
      ],
    }));
    fillList(els.promotionsList, data.pending_promotions, (p) => {
      const applied = !!p.applied;
      return {
        className: 'mb-1.5 flex flex-wrap items-baseline gap-x-2',
        children: [
          {
            tag: 'span',
            className: `font-mono text-sm ${applied ? 'text-accent' : 'text-warn'}`,
            text: applied ? '[applied]' : '[pending]',
          },
          {
            tag: 'span',
            className: 'text-mute',
            text: `${p.id || '—'} · ${p.kind || '—'} · ${p.description || ''}`,
          },
        ],
      };
    });

    const tdd = data.tdd || {};
    els.testsSummary.textContent = `${tdd.passed ?? 0}/${tdd.total ?? 0} pass · ${tdd.failed ?? 0} fail · phase ${tdd.phase || '—'}`;
    setValueTone(els.testsSummary, testsSummaryTone(tdd));
    fillList(els.testsList, tdd.cases, (c) => {
      let tone = 'text-mute';
      if (c.status === 'pass') {
        tone = 'text-accent';
      } else if (c.status === 'fail') {
        tone = 'text-fail';
      } else if (c.status === 'pending') {
        tone = 'text-warn';
      }
      return { text: `${c.name}: ${c.status}`, className: `font-mono text-[0.9rem] ${tone}` };
    });

    renderFindings(els.findings, data.audit?.findings);

    els.output.textContent = tdd.output || 'No test output yet.';
    const outFail = (tdd.failed > 0) || String(tdd.phase || '').toLowerCase() === 'red';
    els.output.className = [
      'm-0 max-h-60 overflow-auto whitespace-pre-wrap rounded-md border border-line bg-canvas-elev p-4 font-mono text-[0.85rem]',
      outFail ? 'text-fail' : 'text-mute',
    ].join(' ');

    const updatedAt = data.updated_at || '';
    if (els.liveStamp) {
      const changed = updatedAt && updatedAt !== lastUpdatedAt;
      lastUpdatedAt = updatedAt;
      const when = updatedAt ? new Date(updatedAt).toLocaleTimeString() : '—';
      els.liveStamp.className = changed
        ? 'mt-2 font-mono text-xs text-progress'
        : 'mt-2 font-mono text-xs text-mute';
      els.liveStamp.textContent = changed
        ? `Live · board updated ${when} · theme ${currentTheme}`
        : `Live · polling · last board touch ${when} · theme ${currentTheme}`;
    }

    const diagram = (data.how_it_works || '').trim();
    if (!diagram) {
      els.diagram.textContent = '—';
      lastDiagram = '';
      resetDiagramView();
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
    configureMermaid(mermaid, currentTheme);
    const id = `mmd-${Date.now()}`;
    try {
      const { svg } = await mermaid.render(id, source);
      els.diagram.innerHTML = svg;
      const node = els.diagram.querySelector('svg');
      if (node) {
        node.style.maxWidth = 'none';
        node.style.height = 'auto';
        node.setAttribute('width', node.viewBox.baseVal.width || node.width.baseVal.value || node.clientWidth);
        node.removeAttribute('height');
      }
      resetDiagramView();
      // Fit large diagrams so the full graph is visible on first load.
      requestAnimationFrame(() => fitDiagram());
    } catch (err) {
      els.diagram.textContent = `Mermaid error: ${err.message}\n\n${source}`;
      resetDiagramView();
    }
  }

  bindDiagramControls();
  waitForMermaid().then(() => {
    refresh();
    setInterval(refresh, 1000);
  });
})();
