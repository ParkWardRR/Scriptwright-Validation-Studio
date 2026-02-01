(() => {
  const logStream = document.getElementById('console-stream');
  const preview = document.getElementById('preview');
  const urlInput = document.getElementById('target-url');
  const engineInput = document.getElementById('engine');
  const captureTrace = document.getElementById('capture-trace');
  const captureHar = document.getElementById('capture-har');
  const chipUrl = document.getElementById('chip-url');
  const chipEngine = document.getElementById('chip-engine');
  const screenshot = document.getElementById('screenshot');
  const webp = document.getElementById('webp');
  const backendStatus = document.getElementById('backend-status');
  const harLink = document.getElementById('har-link');
  const traceLink = document.getElementById('trace-link');

  const metaName = document.getElementById('meta-name');
  const metaMatch = document.getElementById('meta-match');
  const metaDuration = document.getElementById('meta-duration');
  const metaProfile = document.getElementById('meta-profile');
  const flowStepsEl = document.getElementById('flow-steps');
  const flowJsonEl = document.getElementById('flow-json');

  function appendLog(scope, message) {
    const line = document.createElement('div');
    line.className = 'log-line';
    const now = new Date();
    line.innerHTML = `<span class="time">${now.toLocaleTimeString()}</span><span class="scope">${scope}</span>${message}`;
    logStream.prepend(line);
  }

  function loadSample() {
    if (!window.sampleRun) return;
    const run = window.sampleRun;
    metaName.textContent = run.script_meta.Name;
    metaMatch.textContent = (run.script_meta.Match || []).join(', ');
    const started = new Date(run.started_at);
    const finished = new Date(run.finished_at);
    metaDuration.textContent = `${((finished - started) / 1000).toFixed(1)}s`;
    metaProfile.textContent = run.profile_folder || 'temp profile';
    appendLog('runner', 'Loaded sample run manifest.');
    appendLog('browser', `Target ${run.target_url}`);
    screenshot.src = `../${run.screenshot}`;
    webp.src = `../${run.video_webp}`;
  }

  document.getElementById('load-artifacts').addEventListener('click', loadSample);

  document.getElementById('launch').addEventListener('click', () => {
    const url = urlInput.value.trim();
    if (!url) return;
    preview.src = url;
    chipUrl.textContent = url;
    chipEngine.textContent = engineInput.value;
    appendLog('runner', `Opening ${url} (headless=${document.getElementById('headless').value})`);
  });

  async function startRun() {
    if (!backendAvailable) {
      simulate();
      return;
    }
    appendLog('runner', 'Starting run via API…');
    let steps = [];
    try {
      steps = JSON.parse(flowJsonEl.textContent || '[]');
    } catch {
      steps = [];
    }
    const payload = {
      url: urlInput.value.trim(),
      script: document.getElementById('script-path').value.trim(),
      engine: engineInput.value,
      extension_dir: '',
      headless: document.getElementById('headless').value === 'true',
      har: captureHar.checked,
      replay_har: document.getElementById('replay-har').value.trim(),
      baseline: '',
      steps,
    };
    try {
      const res = await fetch(`${apiBase}/v1/runs`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      if (!res.ok) {
        const txt = await res.text();
        throw new Error(txt || `status ${res.status}`);
      }
      const manifest = await res.json();
      applyManifest(manifest);
      appendLog('runner', 'Run complete ✓');
    } catch (err) {
      appendLog('runner', `Run failed: ${err.message}`);
      simulate();
    }
  }

  function simulate() {
    appendLog('runner', 'Simulating run (backend unavailable)…');
    setTimeout(() => appendLog('browser', 'Persistent context launched (Chromium)'), 300);
    setTimeout(() => appendLog('browser', 'Userscript installed (stub: init script injection)'), 700);
    setTimeout(() => appendLog('page', 'Navigated to target; waiting for toggle button'), 1200);
    setTimeout(() => appendLog('page', 'Clicked "Toggle Dark Mode"'), 1600);
    setTimeout(() => appendLog('artifact', `Captured screenshot & video (trace=${captureTrace.checked}, har=${captureHar.checked})`), 2000);
    setTimeout(() => appendLog('runner', 'Run complete ✓'), 2300);
  }

  function applyManifest(manifest) {
    metaName.textContent = manifest.script_meta?.Name || manifest.script_meta?.name || '—';
    metaMatch.textContent = (manifest.script_meta?.Match || []).join(', ');
    const started = new Date(manifest.started_at || manifest.StartedAt);
    const finished = new Date(manifest.finished_at || manifest.FinishedAt);
    metaDuration.textContent = isNaN(started) || isNaN(finished) ? '—' : `${((finished - started) / 1000).toFixed(1)}s`;
    metaProfile.textContent = manifest.profile_folder || manifest.ProfileFolder || 'temp profile';
    if (manifest.screenshot) screenshot.src = manifest.screenshot;
    if (manifest.video_webp) webp.src = manifest.video_webp;
    if (manifest.har) {
      harLink.textContent = manifest.har;
      harLink.href = manifest.har;
      appendLog('artifact', `HAR: ${manifest.har}`);
    }
    if (manifest.trace_zip) {
      traceLink.textContent = manifest.trace_zip;
      traceLink.href = manifest.trace_zip;
      appendLog('artifact', `Trace: ${manifest.trace_zip}`);
    }
  }

  document.getElementById('start-run').addEventListener('click', startRun);

  document.getElementById('add-step').addEventListener('click', () => {
    const step = document.createElement('div');
    step.className = 'flow-step';
    step.innerHTML = `
      <input placeholder="Action (click, fill, wait)" />
      <input placeholder="Selector or value" />
      <input placeholder="Notes / assertion" />
    `;
    flowStepsEl.appendChild(step);
    syncFlowJson();
    step.querySelectorAll('input').forEach((input) => {
      input.addEventListener('input', syncFlowJson);
    });
  });

  function syncFlowJson() {
    const steps = [];
    flowStepsEl.querySelectorAll('.flow-step').forEach((node) => {
      const [action, target, note] = Array.from(node.querySelectorAll('input')).map((i) => i.value);
      if (action || target || note) steps.push({ action, target, note });
    });
    flowJsonEl.textContent = JSON.stringify(steps, null, 2);
  }

  // bootstrap
  let backendAvailable = false;
  let apiBase = 'http://localhost:8787';
  async function detectBackend() {
    try {
      const res = await fetch(`${apiBase}/health`, { mode: 'cors' });
      backendAvailable = res.ok;
    } catch {
      backendAvailable = false;
    }
    backendStatus.textContent = backendAvailable ? 'API: ready' : 'API: offline (simulated)';
    backendStatus.className = 'chip';
  }

  appendLog('ui', 'Web UI ready. Load sample artifacts or launch a page.');
  detectBackend();
  if (window.sampleRun) loadSample();
})();
