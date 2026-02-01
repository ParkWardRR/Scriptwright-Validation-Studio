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

  const metaName = document.getElementById('meta-name');
  const metaMatch = document.getElementById('meta-match');
  const metaDuration = document.getElementById('meta-duration');
  const metaProfile = document.getElementById('meta-profile');

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

  document.getElementById('simulate-run').addEventListener('click', () => {
    appendLog('runner', 'Starting simulated run…');
    setTimeout(() => appendLog('browser', 'Persistent context launched (Chromium)'), 300);
    setTimeout(() => appendLog('browser', 'Userscript installed (stub: init script injection)'), 700);
    setTimeout(() => appendLog('page', 'Navigated to target; waiting for toggle button'), 1200);
    setTimeout(() => appendLog('page', 'Clicked "Toggle Dark Mode"'), 1600);
    setTimeout(() => appendLog('artifact', `Captured screenshot & video (trace=${captureTrace.checked}, har=${captureHar.checked})`), 2000);
    setTimeout(() => appendLog('runner', 'Run complete ✓'), 2300);
  });

  // bootstrap
  appendLog('ui', 'Web UI ready. Load sample artifacts or launch a page.');
  if (window.sampleRun) loadSample();
})();
