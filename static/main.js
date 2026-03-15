// ── WASM bootstrap ────────────────────────────────────────────────────────────
let wasmReady = false;

function setWasmStatus(state, text) {
  const badge = document.getElementById('wasm-status');
  const label = document.getElementById('wasm-status-text');
  badge.className = 'wasm-badge wasm-' + state;
  label.textContent = text;
  if (state === 'ready') {
    wasmReady = true;
    updateGenerateButton();
  }
}

function updateGenerateButton() {
  const btn = document.getElementById('btn-generate');
  if (!btn) return;
  const chips = document.querySelectorAll('#qr-chips .chip');
  btn.disabled = !wasmReady || chips.length === 0;
}

(function loadWasm() {
  if (!window.WebAssembly) {
    setWasmStatus('error', 'WebAssembly not supported');
    return;
  }
  const go = new window.Go();
  WebAssembly.instantiateStreaming(fetch('engine.wasm'), go.importObject)
    .then(obj => {
      go.run(obj.instance);
      wasmReady = true;
      setWasmStatus('ready', 'Engine ready');
    })
    .catch(err => {
      console.error('WASM load failed:', err);
      setWasmStatus('error', 'Engine failed to load');
    });
})();

// ── Navigation ────────────────────────────────────────────────────────────────
const titles = { qr: 'QR Code Generator', pdf: 'PDF Sheet', settings: 'Settings' };

function showPanel(name, el) {
  document.querySelectorAll('.panel').forEach(p => p.classList.remove('active'));
  document.querySelectorAll('.nav-item').forEach(n => n.classList.remove('active'));
  document.getElementById('panel-' + name).classList.add('active');
  el.classList.add('active');
  document.getElementById('page-title').textContent = titles[name];
}

// ── Input parsing ─────────────────────────────────────────────────────────────
function parseWBInput(raw) {
  raw = raw.trim();
  const urlMatch = raw.match(/\/whisky\/(\d+)/i);
  if (urlMatch) return urlMatch[1];
  const wbMatch = raw.match(/^wb(\d+)$/i);
  if (wbMatch) return wbMatch[1];
  if (/^\d+$/.test(raw)) return raw;
  return null;
}

function wbUrl(id) {
  const base = loadSettings().baseUrl || 'https://www.whiskybase.com/whiskies/whisky/';
  return base + id;
}

// ── Chip inputs ───────────────────────────────────────────────────────────────
function handleChipInput(e, tab) {
  if (e.key === 'Enter') { e.preventDefault(); addChip(tab); }
}

function addChip(tab) {
  const input = document.getElementById(tab + '-id-input');
  const val = parseWBInput(input.value);
  if (!val) {
    input.style.borderColor = '#c0392b';
    setTimeout(() => input.style.borderColor = '', 800);
    return;
  }
  addChipValue(tab, val);
  input.value = '';
  input.focus();
}

function addChipValue(tab, val) {
  const container = document.getElementById(tab + '-chips');
  const placeholder = container.querySelector('span');
  if (placeholder) placeholder.remove();
  if (container.querySelector(`.chip[data-val="${val}"]`)) return;
  const chip = document.createElement('div');
  chip.className = 'chip';
  chip.dataset.val = val;
  chip.innerHTML = `WB${val}<span class="chip-remove" onclick="removeChip(this)">×</span>`;
  container.appendChild(chip);
  if (tab === 'qr') updateGenerateButton();
}

function removeChip(el) {
  const chip = el.parentElement;
  const container = chip.parentElement;
  const tab = container.id.replace('-chips', '');
  chip.remove();
  if (!container.querySelector('.chip')) {
    container.innerHTML = '<span style="font-size:0.65rem;color:var(--text-muted);align-self:center;letter-spacing:0.06em;">Added IDs will appear here…</span>';
  }
  if (tab === 'qr') updateGenerateButton();
}

['qr', 'pdf'].forEach(tab => {
  document.getElementById(tab + '-bulk').addEventListener('blur', function() {
    this.value.split(/[\n,]+/).map(s => s.trim()).filter(Boolean).forEach(entry => {
      const val = parseWBInput(entry);
      if (val) addChipValue(tab, val);
    });
    this.value = '';
  });
});

function getChipIds(tab) {
  return [...document.querySelectorAll(`#${tab}-chips .chip`)].map(c => c.dataset.val);
}

// ── Logo file handling ────────────────────────────────────────────────────────
let logoBase64 = '';

function handleLogoFile(input) {
  const file = input.files[0];
  if (!file) return;
  const reader = new FileReader();
  reader.onload = e => {
    logoBase64 = e.target.result;
    document.getElementById('qr-logo-preview').src = e.target.result;
    document.getElementById('qr-logo-preview').style.display = '';
    document.getElementById('qr-logo-placeholder').style.display = 'none';
    document.getElementById('qr-logo-name').textContent = file.name;
    document.getElementById('qr-logo-clear').style.display = '';
    document.getElementById('qr-logo-pad').checked = true;
  };
  reader.readAsDataURL(file);
}

function clearLogo() {
  logoBase64 = '';
  document.getElementById('qr-logo-preview').src = '';
  document.getElementById('qr-logo-preview').style.display = 'none';
  document.getElementById('qr-logo-placeholder').style.display = '';
  document.getElementById('qr-logo-name').textContent = '';
  document.getElementById('qr-logo-file').value = '';
  document.getElementById('qr-logo-clear').style.display = 'none';
}

// ── Pattern selection ─────────────────────────────────────────────────────────
const patternState = { module: 'square', finder: 'square' };

function selectPattern(type, el) {
  document.querySelectorAll(`#pattern-${type} .pattern-option`).forEach(o => o.classList.remove('active'));
  el.classList.add('active');
  patternState[type] = el.dataset.value;
}

const settingsPatternState = { module: 'square', finder: 'square' };

function selectSettingsPattern(type, el) {
  document.querySelectorAll(`#s-pattern-${type} .pattern-option`).forEach(o => o.classList.remove('active'));
  el.classList.add('active');
  settingsPatternState[type] = el.dataset.value;
}

function syncColor(channel, val) {
  if (/^#[0-9a-f]{6}$/i.test(val)) {
    document.getElementById(`qr-color-${channel}`).value = val;
  }
}

function syncSettingsColor(channel, val) {
  if (/^#[0-9a-f]{6}$/i.test(val)) {
    document.getElementById(`s-color-${channel}`).value = val;
  }
}

document.addEventListener('DOMContentLoaded', () => {
  // Color picker sync
  document.getElementById('qr-color-fg').addEventListener('input', function() {
    document.getElementById('qr-color-fg-hex').value = this.value;
  });
  document.getElementById('qr-color-bg').addEventListener('input', function() {
    document.getElementById('qr-color-bg-hex').value = this.value;
  });
  document.getElementById('s-color-fg').addEventListener('input', function() {
    document.getElementById('s-color-fg-hex').value = this.value;
  });
  document.getElementById('s-color-bg').addEventListener('input', function() {
    document.getElementById('s-color-bg-hex').value = this.value;
  });

  // Watch the codes container for any changes and update button state
  const observer = new MutationObserver(updateGenerateButton);
  observer.observe(document.getElementById('qr-chips'), { childList: true, subtree: true });

  // Ensure button starts disabled
  updateGenerateButton();
});

// ── QR Generation ─────────────────────────────────────────────────────────────
function getQRParams() {
  return {
    fmt:           document.querySelector('input[name="qr-fmt"]:checked').value,
    size:          parseInt(document.getElementById('qr-size').value),
    ec:            document.getElementById('qr-ec').value,
    logoBase64:    logoBase64,
    logoSize:      parseInt(document.getElementById('qr-logo-size').value),
    logoRadius:    parseInt(document.getElementById('qr-logo-radius').value),
    quietZone:     document.getElementById('qr-quiet-zone').checked,
    logoPad:       document.getElementById('qr-logo-pad').checked,
    modulePattern: patternState.module,
    finderPattern: patternState.finder,
    colorFg:       document.getElementById('qr-color-fg-hex').value || '#000000',
    colorBg:       document.getElementById('qr-color-bg-hex').value || '#ffffff',
  };
}

function runGenerate() {
  if (!wasmReady) return;

  const ids = getChipIds('qr');
  if (!ids.length) return;

  const p = getQRParams();
  const container = document.getElementById('qr-results');
  const countEl = document.getElementById('qr-result-count');

  // Show loading skeletons
  container.innerHTML = '';
  ids.forEach(id => {
    const card = makeResultCard(id, null, null, null);
    card.classList.add('loading');
    container.appendChild(card);
  });
  countEl.textContent = '';
  document.getElementById('btn-download-all').style.display = 'none';

  const urls = ids.map(id => wbUrl(id));

  let results;
  try {
    // Pass params as a plain JS object (second arg)
    const optsObj = {
      fmt:           p.fmt,
      size:          p.size,
      ec:            p.ec,
      logoBase64:    p.logoBase64,
      logoSize:      p.logoSize,
      logoRadius:    p.logoRadius,
      quietZone:     p.quietZone,
      logoPad:       p.logoPad,
      modulePattern: p.modulePattern,
      finderPattern: p.finderPattern,
      colorFg:       p.colorFg,
      colorBg:       p.colorBg,
    };
    results = generateQrWithLogo(urls, optsObj);
    // Debug — remove once working
    if (results && results.length > 0) {
      const r0 = results[0];
      console.log('WASM keys:', Object.keys(r0));
      console.log('png:', typeof r0.png, (r0.png || '').slice(0, 40));
      console.log('svg:', typeof r0.svg, (r0.svg || '').slice(0, 40));
      console.log('data:', typeof r0.data, (r0.data || '').slice(0, 40));
      console.log('error:', r0.error);
    }
  } catch (e) {
    container.innerHTML = `<div style="color:#c0392b;font-size:0.75rem;padding:1rem;">${e.message}</div>`;
    return;
  }

  container.innerHTML = '';
  const pngStore = {};
  const svgStore = {};

  for (let i = 0; i < results.length; i++) {
    const r = results[i];
    const code    = r.code  ?? r['code']  ?? '';
    const err     = r.error ?? r['error'] ?? '';
    // Support both new (png/svg) and old (data) field names
    const pngData = r.png   ?? r['png']   ?? r.data ?? r['data'] ?? '';
    const svgData = r.svg   ?? r['svg']   ?? '';
    const idMatch = code.match(/\/(\d+)$/);
    const id = idMatch ? idMatch[1] : ids[i];

    if (err) {
      const card = makeResultCard(id, null, null, err);
      container.appendChild(card);
      continue;
    }

    if (!pngData && !svgData) {
      const card = makeResultCard(id, null, null, 'No data returned');
      container.appendChild(card);
      continue;
    }

    if (pngData) pngStore[id] = pngData;
    if (svgData) svgStore[id] = svgData;

    const card = makeResultCard(id, pngData, svgData, null);
    container.appendChild(card);
  }

  countEl.textContent = `(${ids.length})`;
  const hasResults = Object.keys(pngStore).length || Object.keys(svgStore).length;
  if (hasResults) {
    const btn = document.getElementById('btn-download-all');
    btn.style.display = '';
    btn.textContent = p.fmt === 'both' ? 'Download all' : `Download all ${p.fmt.toUpperCase()}`;
    btn._pngStore = pngStore;
    btn._svgStore = svgStore;
  }
}

function makeResultCard(id, pngData, svgData, err) {
  const card = document.createElement('div');
  card.className = 'result-card';

  if (err) {
    card.classList.add('error');
    card.innerHTML = `
      <div class="qr-placeholder" style="width:120px;height:120px;"></div>
      <span class="qr-label">WB${id}</span>
      <span style="font-size:0.6rem;color:#c0392b;text-align:center;">${err}</span>`;
    return card;
  }

  if (!pngData && !svgData) {
    card.classList.add('loading');
    card.innerHTML = `
      <div class="qr-img" style="width:120px;height:120px;"></div>
      <span class="qr-label">WB${id}</span>`;
    return card;
  }

  // Preview: prefer PNG for the img tag, fall back to SVG data URL
  const previewSrc = pngData || `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svgData)}`;

  let actions = '';
  if (pngData) {
    actions += `<a class="btn btn-ghost btn-sm" href="${pngData}" download="WB${id}.png">PNG</a>`;
  }
  if (svgData) {
    const svgUrl = URL.createObjectURL(new Blob([svgData], { type: 'image/svg+xml' }));
    actions += `<a class="btn btn-ghost btn-sm" href="${svgUrl}" download="WB${id}.svg">SVG</a>`;
  }

  card.innerHTML = `
    <img class="qr-img" src="${previewSrc}" alt="QR WB${id}">
    <span class="qr-label">WB${id}</span>
    <div class="qr-actions">${actions}</div>`;
  return card;
}

function downloadAll() {
  const btn = document.getElementById('btn-download-all');
  const pngStore = btn._pngStore || {};
  const svgStore = btn._svgStore || {};
  const ids = [...new Set([...Object.keys(pngStore), ...Object.keys(svgStore)])];

  ids.forEach((id, i) => {
    setTimeout(() => {
      if (pngStore[id]) triggerDownload(`WB${id}.png`, pngStore[id]);
      if (svgStore[id]) {
        const svgUrl = URL.createObjectURL(new Blob([svgStore[id]], { type: 'image/svg+xml' }));
        triggerDownload(`WB${id}.svg`, svgUrl);
      }
    }, i * 100);
  });
}

function triggerDownload(filename, url) {
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
}

function clearAll() {
  ['qr', 'pdf'].forEach(tab => {
    const container = document.getElementById(tab + '-chips');
    container.innerHTML = '<span style="font-size:0.65rem;color:var(--text-muted);align-self:center;letter-spacing:0.06em;">Added IDs will appear here…</span>';
  });
  document.getElementById('qr-results').innerHTML = `
    <div class="empty" style="grid-column:1/-1;border:1px dashed var(--border);border-radius:4px;padding:3rem 1rem;">
      <span class="empty-icon">◈</span>
      <p>Add WhiskyBase codes and press Generate</p>
    </div>`;
  document.getElementById('qr-result-count').textContent = '';
  document.getElementById('btn-download-all').style.display = 'none';
  updateGenerateButton();
}

// ── Settings ──────────────────────────────────────────────────────────────────
const SETTINGS_KEY = 'wb-tools-settings';

let settingsLogoBase64 = '';

const DEFAULT_SETTINGS = {
  fmt:           'svg',
  size:          '512',
  ec:            'H',
  paper:         'a4',
  cols:          '2',
  prefix:        'wb-',
  baseUrl:       'https://www.whiskybase.com/whiskies/whisky/',
  logoName:      '',
  logoBase64:    '',
  logoSize:      '25',
  modulePattern: 'square',
  finderPattern: 'square',
  colorFg:       '#000000',
  colorBg:       '#ffffff',
};

function loadSettings() {
  try {
    const raw = localStorage.getItem(SETTINGS_KEY);
    return raw ? { ...DEFAULT_SETTINGS, ...JSON.parse(raw) } : { ...DEFAULT_SETTINGS };
  } catch {
    return { ...DEFAULT_SETTINGS };
  }
}

function applySettingsToUI(s) {
  // QR format radio
  const fmtRadio = document.querySelector(`input[name="s-fmt"][value="${s.fmt}"]`);
  if (fmtRadio) fmtRadio.checked = true;

  document.getElementById('s-size').value      = s.size;
  document.getElementById('s-ec').value        = s.ec;
  document.getElementById('s-paper').value     = s.paper;
  document.getElementById('s-cols').value      = s.cols;
  document.getElementById('s-prefix').value    = s.prefix;
  document.getElementById('s-base-url').value  = s.baseUrl;
  document.getElementById('s-logo-size').value = s.logoSize;

  // Pattern pickers
  ['module', 'finder'].forEach(type => {
    const val = type === 'module' ? s.modulePattern : s.finderPattern;
    settingsPatternState[type] = val;
    document.querySelectorAll(`#s-pattern-${type} .pattern-option`).forEach(o => {
      o.classList.toggle('active', o.dataset.value === val);
    });
  });

  // Colors
  document.getElementById('s-color-fg').value     = s.colorFg;
  document.getElementById('s-color-fg-hex').value = s.colorFg;
  document.getElementById('s-color-bg').value     = s.colorBg;
  document.getElementById('s-color-bg-hex').value = s.colorBg;

  settingsLogoBase64 = s.logoBase64 || '';
  if (s.logoName && s.logoBase64) {
    document.getElementById('s-logo-preview').src = s.logoBase64;
    document.getElementById('s-logo-preview').style.display = '';
    document.getElementById('s-logo-placeholder').style.display = 'none';
    document.getElementById('s-logo-name').textContent = s.logoName;
    document.getElementById('s-logo-clear').style.display = '';
  }

  // Apply all defaults to QR generator controls
  const qrFmtRadio = document.querySelector(`input[name="qr-fmt"][value="${s.fmt}"]`);
  if (qrFmtRadio) qrFmtRadio.checked = true;
  document.getElementById('qr-size').value       = s.size;
  document.getElementById('qr-ec').value         = s.ec;
  document.getElementById('qr-logo-size').value  = s.logoSize;
  document.getElementById('qr-color-fg-hex').value = s.colorFg;
  document.getElementById('qr-color-bg-hex').value = s.colorBg;
  document.getElementById('qr-color-fg').value   = s.colorFg;
  document.getElementById('qr-color-bg').value   = s.colorBg;

  // Apply pattern defaults to QR generator pickers
  ['module', 'finder'].forEach(type => {
    const val = type === 'module' ? s.modulePattern : s.finderPattern;
    patternState[type] = val;
    document.querySelectorAll(`#pattern-${type} .pattern-option`).forEach(o => {
      o.classList.toggle('active', o.dataset.value === val);
    });
  });

  // Apply default logo to QR generator
  if (s.logoBase64 && s.logoName) {
    logoBase64 = s.logoBase64;
    document.getElementById('qr-logo-preview').src = s.logoBase64;
    document.getElementById('qr-logo-preview').style.display = '';
    document.getElementById('qr-logo-placeholder').style.display = 'none';
    document.getElementById('qr-logo-name').textContent = s.logoName;
    document.getElementById('qr-logo-clear').style.display = '';
    document.getElementById('qr-logo-pad').checked = true;
  }
}

function saveSettings() {
  const fmt = document.querySelector('input[name="s-fmt"]:checked')?.value || 'svg';
  const s = {
    fmt,
    size:          document.getElementById('s-size').value,
    ec:            document.getElementById('s-ec').value,
    paper:         document.getElementById('s-paper').value,
    cols:          document.getElementById('s-cols').value,
    prefix:        document.getElementById('s-prefix').value,
    baseUrl:       document.getElementById('s-base-url').value,
    logoName:      document.getElementById('s-logo-name').value,
    logoBase64:    settingsLogoBase64,
    logoSize:      document.getElementById('s-logo-size').value,
    modulePattern: settingsPatternState.module,
    finderPattern: settingsPatternState.finder,
    colorFg:       document.getElementById('s-color-fg-hex').value || '#000000',
    colorBg:       document.getElementById('s-color-bg-hex').value || '#ffffff',
  };
  localStorage.setItem(SETTINGS_KEY, JSON.stringify(s));
  applySettingsToUI(s);

  const msg = document.getElementById('settings-saved-msg');
  msg.style.display = 'block';
  setTimeout(() => msg.style.display = 'none', 2000);
}

function resetSettings() {
  localStorage.removeItem(SETTINGS_KEY);
  settingsLogoBase64 = '';
  clearSettingsLogo();
  applySettingsToUI({ ...DEFAULT_SETTINGS });
}

function handleSettingsLogoFile(input) {
  const file = input.files[0];
  if (!file) return;
  const reader = new FileReader();
  reader.onload = e => {
    settingsLogoBase64 = e.target.result;
    document.getElementById('s-logo-preview').src = e.target.result;
    document.getElementById('s-logo-preview').style.display = '';
    document.getElementById('s-logo-placeholder').style.display = 'none';
    document.getElementById('s-logo-name').textContent = file.name;
    document.getElementById('s-logo-clear').style.display = '';
  };
  reader.readAsDataURL(file);
}

function clearSettingsLogo() {
  settingsLogoBase64 = '';
  document.getElementById('s-logo-preview').src = '';
  document.getElementById('s-logo-preview').style.display = 'none';
  document.getElementById('s-logo-placeholder').style.display = '';
  document.getElementById('s-logo-name').textContent = '';
  document.getElementById('s-logo-file').value = '';
  document.getElementById('s-logo-clear').style.display = 'none';
}

// Load settings on startup — handled in the single DOMContentLoaded above

function updatePaperPreview() {
  const paper = document.getElementById('pdf-paper').value;
  const orient = document.getElementById('pdf-orient').value;
  const cols = parseInt(document.getElementById('pdf-cols').value);
  const mock = document.getElementById('paper-mock');
  const row = document.getElementById('paper-mock-row');
  const isLandscape = orient === 'landscape';
  const baseW = paper === 'a4' ? 90 : 96;
  const baseH = paper === 'a4' ? 127 : 124;
  mock.style.width  = (isLandscape ? baseH : baseW) + 'px';
  mock.style.height = (isLandscape ? baseW : baseH) + 'px';
  row.innerHTML = '';
  for (let i = 0; i < cols; i++) {
    row.innerHTML += `
      <div class="paper-cell">
        <div class="paper-qr" style="width:16px;height:16px;"></div>
        <div class="paper-lines">
          <div class="paper-line"></div>
          <div class="paper-line"></div>
          <div class="paper-line"></div>
        </div>
      </div>`;
  }
}