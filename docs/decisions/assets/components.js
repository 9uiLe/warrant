/* ============================================================
   tech-docs / Components JS
   ------------------------------------------------------------
   提供する公開 API (window.DocsADR):
     initAll(opts)         — 全機能を一括初期化
     initCopyButtons(root) — code-block にコピーボタン
     buildTOC(opts)        — 見出しから TOC を生成
     initScrollspy(opts)   — 現在地ハイライト
     renderBars(root)      — data-bars 駆動の棒グラフ
     renderDonuts(root)    — data-donut 駆動のドーナツ
     renderLines(root)     — data-line 駆動の折れ線 (SVG)
     initMermaid()         — Mermaid のテーマ初期化
     initPrism()           — Prism のハイライト実行
   ============================================================ */

(function () {
  'use strict';

  const DocsADR = (window.DocsADR = window.DocsADR || {});

  // ----------------------------------------------------------
  // Internal helpers
  // ----------------------------------------------------------

  /** querySelector 引数を許容しつつ要素を返す。 */
  function resolveElement(target) {
    if (target == null) return null;
    return typeof target === 'string' ? document.querySelector(target) : target;
  }

  /** dataset から JSON を読む。失敗時は null。 */
  function readDataset(el, key) {
    const raw = el.dataset[key];
    if (!raw) return null;
    try {
      return JSON.parse(raw);
    } catch {
      return null;
    }
  }

  /**
   * 同じ要素に対する二重初期化を防ぐ。
   * 既に初期化済みなら false、そうでなければマークして true を返す。
   */
  function markOnce(el, flagKey) {
    if (el.dataset[flagKey]) return false;
    el.dataset[flagKey] = '1';
    return true;
  }

  /** クリップボードへの書き込み。Promise を返し失敗時は legacy 経路。 */
  async function writeToClipboard(text) {
    try {
      await navigator.clipboard.writeText(text);
    } catch {
      const range = document.createRange();
      const tmp = document.createElement('textarea');
      tmp.value = text;
      tmp.style.position = 'fixed';
      tmp.style.opacity = '0';
      document.body.appendChild(tmp);
      tmp.select();
      try {
        document.execCommand('copy');
      } finally {
        document.body.removeChild(tmp);
      }
    }
  }

  /** CSS カスタムプロパティを root から取得。なければ fallback。 */
  function readCssVar(name, fallback) {
    const value = getComputedStyle(document.documentElement)
      .getPropertyValue(name)
      .trim();
    return value || fallback;
  }

  /** 文字列を heading id 用のスラグに整形。 */
  function slugify(text, fallback) {
    const slug = (text || '')
      .trim()
      .toLowerCase()
      .replace(/[^\p{L}\p{N}]+/gu, '-')
      .replace(/(^-|-$)/g, '');
    return slug || fallback;
  }

  // ----------------------------------------------------------
  // 1. Copy buttons on code blocks
  // ----------------------------------------------------------

  function ensureCopyButton(block) {
    let btn = block.querySelector('.code-block__copy');
    if (btn) return btn;

    let header = block.querySelector('.code-block__header');
    if (!header) {
      header = document.createElement('div');
      header.className = 'code-block__header';
      const spacer = document.createElement('span');
      spacer.className = 'code-block__spacer';
      header.appendChild(spacer);
      block.insertBefore(header, block.firstChild);
    }

    btn = document.createElement('button');
    btn.className = 'code-block__copy';
    btn.type = 'button';
    btn.textContent = 'Copy';
    header.appendChild(btn);
    return btn;
  }

  function attachCopyHandler(block, btn) {
    btn.addEventListener('click', async () => {
      const pre = block.querySelector('pre');
      if (!pre) return;
      await writeToClipboard(pre.innerText);

      const prev = btn.textContent;
      btn.textContent = 'Copied';
      btn.classList.add('is-copied');
      setTimeout(() => {
        btn.textContent = prev;
        btn.classList.remove('is-copied');
      }, 1400);
    });
  }

  function initCopyButtons(root) {
    const scope = root || document;
    scope.querySelectorAll('.code-block').forEach((block) => {
      if (!markOnce(block, 'copyInit')) return;
      const btn = ensureCopyButton(block);
      attachCopyHandler(block, btn);
    });
  }
  DocsADR.initCopyButtons = initCopyButtons;

  // ----------------------------------------------------------
  // 2. Table of Contents — generation
  // ----------------------------------------------------------

  function buildTOC(opts) {
    const config = opts || {};
    const selector = config.selector || 'h2, h3';
    const tocEl = resolveElement(config.container);
    const contentEl = resolveElement(config.content);
    if (!tocEl || !contentEl) return;

    const headings = [...contentEl.querySelectorAll(selector)];
    if (!headings.length) return;

    headings.forEach((heading, i) => {
      if (!heading.id) {
        heading.id = slugify(heading.textContent, `section-${i}`);
      }
    });

    const ul = document.createElement('ul');
    headings.forEach((heading) => {
      const li = document.createElement('li');
      const a = document.createElement('a');
      const level = parseInt(heading.tagName.substring(1), 10);
      a.href = `#${heading.id}`;
      a.textContent = heading.textContent;
      a.className = `lvl-${level}`;
      a.dataset.target = heading.id;
      li.appendChild(a);
      ul.appendChild(li);
    });

    const existing = tocEl.querySelector('ul');
    if (existing) existing.replaceWith(ul);
    else tocEl.appendChild(ul);

    return headings;
  }
  DocsADR.buildTOC = buildTOC;

  // ----------------------------------------------------------
  // 3. Scrollspy for TOC
  // ----------------------------------------------------------

  function pickActiveHeading(visible) {
    let bestId = null;
    let bestY = Infinity;
    visible.forEach((id) => {
      const el = document.getElementById(id);
      if (!el) return;
      const y = el.getBoundingClientRect().top;
      if (y >= 0 && y < bestY) {
        bestY = y;
        bestId = id;
      }
    });
    return bestId;
  }

  function findLastHeadingAbove(headings) {
    let last = null;
    headings.forEach((h) => {
      if (h.getBoundingClientRect().top < 80) last = h.id;
    });
    return last;
  }

  function smoothScrollTo(el) {
    const top = el.getBoundingClientRect().top + window.scrollY - 40;
    window.scrollTo({ top, behavior: 'smooth' });
  }

  function initScrollspy(opts) {
    const config = opts || {};
    const tocEl = resolveElement(config.toc);
    if (!tocEl) return;

    const links = [...tocEl.querySelectorAll('a[data-target]')];
    if (!links.length) return;

    const headings =
      config.headings ||
      links
        .map((link) => document.getElementById(link.dataset.target))
        .filter(Boolean);

    let activeId = null;
    const setActive = (id) => {
      if (id === activeId) return;
      activeId = id;
      links.forEach((link) =>
        link.classList.toggle('is-active', link.dataset.target === id)
      );
    };

    const visible = new Set();
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) visible.add(entry.target.id);
          else visible.delete(entry.target.id);
        });

        if (visible.size) {
          const id = pickActiveHeading(visible);
          if (id) setActive(id);
        } else {
          const id = findLastHeadingAbove(headings);
          if (id) setActive(id);
        }
      },
      { rootMargin: '-80px 0px -60% 0px', threshold: [0, 1] }
    );
    headings.forEach((h) => observer.observe(h));

    links.forEach((link) => {
      link.addEventListener('click', (ev) => {
        const id = link.dataset.target;
        const el = document.getElementById(id);
        if (!el) return;
        ev.preventDefault();
        smoothScrollTo(el);
        history.replaceState(null, '', `#${id}`);
        setActive(id);
      });
    });
  }
  DocsADR.initScrollspy = initScrollspy;

  // ----------------------------------------------------------
  // 4. Bar chart — CSS-only width, JS sets percentages
  // ----------------------------------------------------------

  function renderBars(root) {
    const scope = root || document;
    scope.querySelectorAll('.bar-chart[data-bars]').forEach((chart) => {
      if (!markOnce(chart, 'barsInit')) return;
      const data = readDataset(chart, 'bars');
      if (!data) return;

      const max = Math.max(
        ...data.map((d) => Number(d.max != null ? d.max : d.value) || 0),
        1
      );

      chart.innerHTML = data
        .map((d) => {
          const variantCls = d.variant ? `bar-row__fill--${d.variant}` : '';
          const unit = d.unit ? `<span class="unit"> ${d.unit}</span>` : '';
          return `<div class="bar-row">
            <div class="bar-row__label">${d.label}</div>
            <div class="bar-row__track"><div class="bar-row__fill ${variantCls}" style="width:0%"></div></div>
            <div class="bar-row__value">${d.value}${unit}</div>
          </div>`;
        })
        .join('');

      requestAnimationFrame(() => {
        chart.querySelectorAll('.bar-row__fill').forEach((fill, i) => {
          const pct = Math.min(100, Math.round((Number(data[i].value) / max) * 100));
          fill.style.width = pct + '%';
        });
      });
    });
  }
  DocsADR.renderBars = renderBars;

  // ----------------------------------------------------------
  // 5. Donut chart — conic-gradient + legend
  // ----------------------------------------------------------

  const DEFAULT_PALETTE = [
    '#4f46e5',
    '#0284c7',
    '#059669',
    '#b45309',
    '#dc2626',
    '#7c3aed',
    '#0891b2',
  ];

  function pickPaletteColor(seg, index) {
    return seg.color || DEFAULT_PALETTE[index % DEFAULT_PALETTE.length];
  }

  function buildDonutGradient(segments, total) {
    let acc = 0;
    const stops = segments.map((seg, i) => {
      const start = (acc / total) * 100;
      acc += Number(seg.value);
      const end = (acc / total) * 100;
      return `${pickPaletteColor(seg, i)} ${start}% ${end}%`;
    });
    return `conic-gradient(${stops.join(', ')})`;
  }

  function buildDonutLegend(segments, total) {
    return segments
      .map((seg, i) => {
        const color = pickPaletteColor(seg, i);
        const precision = seg.precision != null ? seg.precision : 0;
        const pct = ((Number(seg.value) / total) * 100).toFixed(precision);
        return `<div class="donut-legend__item">
          <span class="donut-legend__swatch" style="background:${color}"></span>
          <span>${seg.label}</span>
          <span class="donut-legend__value">${pct}%</span>
        </div>`;
      })
      .join('');
  }

  function renderDonuts(root) {
    const scope = root || document;
    scope.querySelectorAll('.donut-chart[data-donut]').forEach((chart) => {
      if (!markOnce(chart, 'donutInit')) return;
      const cfg = readDataset(chart, 'donut');
      if (!cfg) return;

      const segments = cfg.segments || [];
      const total = segments.reduce((sum, s) => sum + Number(s.value), 0) || 1;

      const gradient = buildDonutGradient(segments, total);
      const center = cfg.center != null ? cfg.center : '';
      const label = cfg.label != null ? cfg.label : '';
      const legend = buildDonutLegend(segments, total);

      chart.innerHTML = `
        <div class="donut" style="background:${gradient}">
          <div class="donut__center">
            <span class="num">${center}</span>
            <span class="lbl">${label}</span>
          </div>
        </div>
        <div class="donut-legend">${legend}</div>
      `;
    });
  }
  DocsADR.renderDonuts = renderDonuts;

  // ----------------------------------------------------------
  // 6. Line chart — SVG
  // ----------------------------------------------------------

  const LINE_CHART_GEOMETRY = {
    width: 720,
    height: 260,
    padLeft: 44,
    padRight: 16,
    padTop: 16,
    padBottom: 28,
  };

  const SERIES_COLOR_MAP = {
    accent: 'var(--color-accent)',
    success: 'var(--color-success)',
    neutral: 'var(--color-text-muted)',
    warning: 'var(--color-warning)',
    danger: 'var(--color-danger)',
  };

  function computeLineScale(cfg, series) {
    const allValues = series.flatMap((s) => s.values).filter((v) => v != null);
    let yMin = cfg.yMin != null ? cfg.yMin : Math.min(0, Math.min(...allValues));
    let yMax = cfg.yMax != null ? cfg.yMax : Math.max(...allValues);
    if (yMin === yMax) yMax = yMin + 1;
    yMax += (yMax - yMin) * 0.1;
    return { yMin, yMax };
  }

  function buildLineProjector(n, geometry, yMin, yMax) {
    const { width, height, padLeft, padRight, padTop, padBottom } = geometry;
    const innerW = width - padLeft - padRight;
    const innerH = height - padTop - padBottom;
    const xAt = (i) =>
      padLeft + (n === 1 ? innerW / 2 : (innerW * i) / (n - 1));
    const yAt = (v) =>
      padTop + innerH - ((v - yMin) / (yMax - yMin)) * innerH;
    return { xAt, yAt };
  }

  function buildLineAxes(cfg, xLabels, geometry, yMin, yMax, xAt) {
    const { width, height, padLeft, padRight, padTop, padBottom } = geometry;
    const innerH = height - padTop - padBottom;
    const precision = cfg.yPrecision != null ? cfg.yPrecision : 0;

    const gridLines = [];
    const yTicks = [];
    for (let k = 0; k <= 4; k++) {
      const y = padTop + (innerH * k) / 4;
      const value = yMax - ((yMax - yMin) * k) / 4;
      gridLines.push(
        `<line x1="${padLeft}" x2="${width - padRight}" y1="${y}" y2="${y}" />`
      );
      yTicks.push(
        `<text x="${padLeft - 8}" y="${y + 3}" text-anchor="end">${Number(value).toFixed(precision)}</text>`
      );
    }

    const n = xLabels.length;
    const step = Math.max(1, Math.ceil(n / 7));
    const xTicks = xLabels
      .map((label, i) =>
        i % step === 0 || i === n - 1
          ? `<text x="${xAt(i)}" y="${height - padBottom + 16}" text-anchor="middle">${label}</text>`
          : ''
      )
      .join('');

    return { gridLines, yTicks, xTicks };
  }

  function buildLineSeries(cfg, series, xAt, yAt, yMin, n) {
    return series
      .map((s) => {
        const variant = s.variant || 'accent';
        const points = s.values.map((v, i) => `${xAt(i)},${yAt(v)}`).join(' ');
        const areaPath =
          `M ${xAt(0)},${yAt(yMin)} ` +
          s.values.map((v, i) => `L ${xAt(i)},${yAt(v)}`).join(' ') +
          ` L ${xAt(n - 1)},${yAt(yMin)} Z`;
        const dots = s.values
          .map(
            (v, i) =>
              `<circle class="series-dot" cx="${xAt(i)}" cy="${yAt(v)}" r="3.5" />`
          )
          .join('');

        return `<g class="series series--${variant}">
          ${cfg.area === false ? '' : `<path class="series-area" d="${areaPath}" />`}
          <polyline class="series-line" points="${points}" />
          ${cfg.dots === false ? '' : dots}
        </g>`;
      })
      .join('');
  }

  function buildLineLegend(series) {
    return series
      .map((s) => {
        const variant = s.variant || 'accent';
        const swatch = SERIES_COLOR_MAP[variant] || SERIES_COLOR_MAP.accent;
        return `<span class="lg" style="--swatch:${swatch}">${s.name}</span>`;
      })
      .join('');
  }

  function renderLines(root) {
    const scope = root || document;
    scope.querySelectorAll('.line-chart[data-line]').forEach((chart) => {
      if (!markOnce(chart, 'lineInit')) return;
      const cfg = readDataset(chart, 'line');
      if (!cfg) return;

      const series = cfg.series || [];
      const xLabels = cfg.xLabels || [];
      const n = xLabels.length || (series[0] && series[0].values.length) || 0;
      if (!n) return;

      const { yMin, yMax } = computeLineScale(cfg, series);
      const { xAt, yAt } = buildLineProjector(n, LINE_CHART_GEOMETRY, yMin, yMax);
      const { gridLines, yTicks, xTicks } = buildLineAxes(
        cfg,
        xLabels,
        LINE_CHART_GEOMETRY,
        yMin,
        yMax,
        xAt
      );
      const seriesSVG = buildLineSeries(cfg, series, xAt, yAt, yMin, n);
      const { width, height } = LINE_CHART_GEOMETRY;

      chart.innerHTML = `
        ${cfg.title ? `<div class="chart-title">${cfg.title}</div>` : ''}
        <svg viewBox="0 0 ${width} ${height}" preserveAspectRatio="xMidYMid meet" role="img">
          <g class="grid">${gridLines.join('')}</g>
          <g class="axis">${yTicks.join('')}${xTicks}</g>
          ${seriesSVG}
        </svg>
        ${series.length > 1 ? `<div class="chart-legend">${buildLineLegend(series)}</div>` : ''}
      `;
    });
  }
  DocsADR.renderLines = renderLines;

  // ----------------------------------------------------------
  // 7. Mermaid — bootstrap with project tokens
  // ----------------------------------------------------------

  function buildMermaidTheme() {
    return {
      background:           readCssVar('--color-bg-elevated',    '#fff'),
      primaryColor:         readCssVar('--color-accent-soft',    '#eef2ff'),
      primaryBorderColor:   readCssVar('--color-accent-border',  '#c7d2fe'),
      primaryTextColor:     readCssVar('--color-text',           '#0f172a'),
      secondaryColor:       readCssVar('--color-bg-inset',       '#eef1f4'),
      secondaryBorderColor: readCssVar('--color-border',         '#e4e7ec'),
      secondaryTextColor:   readCssVar('--color-text',           '#0f172a'),
      tertiaryColor:        readCssVar('--color-bg-subtle',      '#f4f6f8'),
      tertiaryBorderColor:  readCssVar('--color-border',         '#e4e7ec'),
      tertiaryTextColor:    readCssVar('--color-text',           '#0f172a'),
      lineColor:            readCssVar('--color-border-strong',  '#cbd2da'),
      textColor:            readCssVar('--color-text',           '#0f172a'),
      fontSize:             '14px',
      // sequence
      actorBkg:             readCssVar('--color-accent-soft',    '#eef2ff'),
      actorBorder:          readCssVar('--color-accent-border',  '#c7d2fe'),
      actorTextColor:       readCssVar('--color-text',           '#0f172a'),
      signalColor:          readCssVar('--color-text-secondary', '#3f4a5d'),
      signalTextColor:      readCssVar('--color-text',           '#0f172a'),
      labelBoxBkgColor:     readCssVar('--color-bg-subtle',      '#f4f6f8'),
      labelBoxBorderColor:  readCssVar('--color-border',         '#e4e7ec'),
      labelTextColor:       readCssVar('--color-text',           '#0f172a'),
      noteBkgColor:         readCssVar('--color-warning-soft',   '#fffbeb'),
      noteBorderColor:      readCssVar('--color-warning-border', '#fcd34d'),
      noteTextColor:        readCssVar('--color-warning-text',   '#92400e'),
      activationBkgColor:   readCssVar('--color-accent',         '#4f46e5'),
      activationBorderColor:readCssVar('--color-accent-hover',   '#4338ca'),
    };
  }

  function initMermaid() {
    if (!window.mermaid) return;

    window.mermaid.initialize({
      startOnLoad: false,
      securityLevel: 'loose',
      fontFamily: readCssVar('--font-sans', 'sans-serif'),
      themeVariables: buildMermaidTheme(),
      flowchart: {
        curve: 'basis',
        useMaxWidth: true,
        htmlLabels: true,
        padding: 12,
      },
      sequence: {
        useMaxWidth: true,
        mirrorActors: false,
        boxMargin: 8,
        messageMargin: 28,
      },
      er:    { useMaxWidth: true },
      gantt: { useMaxWidth: true },
    });
    window.mermaid.run({ querySelector: '.mermaid' }).catch(() => {});
  }
  DocsADR.initMermaid = initMermaid;

  // ----------------------------------------------------------
  // 8. Prism — highlight all
  // ----------------------------------------------------------

  function initPrism() {
    if (window.Prism && typeof window.Prism.highlightAll === 'function') {
      try {
        window.Prism.highlightAll();
      } catch {
        /* ignore */
      }
    }
  }
  DocsADR.initPrism = initPrism;

  // ----------------------------------------------------------
  // 9. Init all
  // ----------------------------------------------------------

  DocsADR.initAll = function (opts) {
    const options = opts || {};
    initCopyButtons();
    renderBars();
    renderDonuts();
    renderLines();
    initPrism();
    initMermaid();
    if (options.toc) {
      const heads = buildTOC(options.toc);
      initScrollspy({ toc: options.toc.container, headings: heads });
    }
  };

  // ----------------------------------------------------------
  // Auto-init non-interactive bits on DOM ready
  // ----------------------------------------------------------

  function autoInit() {
    initCopyButtons();
    renderBars();
    renderDonuts();
    renderLines();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', autoInit);
  } else {
    autoInit();
  }
})();
