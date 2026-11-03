// Development-only renderer: xterm consumes native ANSI; HTML contains no app UI.
import { readFile, writeFile, mkdir, mkdtemp, rename, rm } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { dirname, resolve, join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { execFileSync } from 'node:child_process';
import { validateEvidence } from './evidence.mjs';
import { browserDeadline, closeBrowserResource, launchOwnedBrowser, pinnedBrowserCommand, browserCaptureScratch } from './browser-lifecycle.mjs';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../..');
const cache = join(root, '.cache/tui-visual');
const modules = join(cache, 'tools/node_modules');
const hash = data => createHash('sha256').update(data).digest('hex');
const json = value => JSON.stringify(value, null, 2) + '\n';
const privatePattern = /sk-capture-DUMMY-secret-never-publish|capture\.invalid|https?:\/\//i;
const themes = {
  dark: { background: '#111820', foreground: '#E6EAF0' },
  light: { background: '#F5F6F8', foreground: '#17212B' },
  mono: { background: '#111111', foreground: '#eeeeee' },
};

async function native(manifestPath, browser, finish = async () => {}) {
  const manifest = JSON.parse(await readFile(manifestPath, 'utf8'));
  if (manifest.kind !== 'native-pty') throw new Error('Expected native-pty manifest');
  if (!manifest.captures.some(e => e.status === 'captured-ansi')) return;
  const output = dirname(manifestPath);
  const css = await readFile(join(modules, '@xterm/xterm/css/xterm.css'), 'utf8');
  const js = await readFile(join(modules, '@xterm/xterm/lib/xterm.js'), 'utf8');
  let fontCSS = '';
  const fontAssets = [];
  for (const [weight, style] of [[400, 'normal'], [700, 'normal'], [400, 'italic'], [700, 'italic']]) {
    const name = `jetbrains-mono-latin-${weight}-${style}.woff2`;
    const data = await readFile(join(modules, '@fontsource/jetbrains-mono/files', name));
    fontAssets.push({ name, sha256: hash(data) });
    fontCSS += `@font-face{font-family:CaptureMono;font-style:${style};font-weight:${weight};src:url(data:font/woff2;base64,${data.toString('base64')}) format('woff2');font-display:block}`;
  }
  const environment = { kind: 'xterm.js DOM renderer', xterm: '5.5.0', playwright: '1.64.0',
    chromium: browser.version(), font: 'JetBrains Mono Latin 5.2.8', font_assets: fontAssets,
    xterm_js_sha256: hash(js), xterm_css_sha256: hash(css), font_size: 16, line_height: 1.2, device_scale_factor: 1,
    unicode_fallback: 'Non-Latin glyphs use Chromium host fallback; compare only identical font/browser environments',
    render_source_sha256: hash(await readFile(new URL('./render.mjs', import.meta.url))),
    evidence_source_sha256: hash(await readFile(new URL('./evidence.mjs', import.meta.url))),
    lifecycle_source_sha256: hash(await readFile(new URL('./browser-lifecycle.mjs', import.meta.url))),
    browser_job_source_sha256: hash(await readFile(new URL('./browser-job.py', import.meta.url))),
    browser_transport: 'owned Chromium process with loopback CDP' };
  // Freeze the environment before any resumed writes, including failure writes.
  // Legacy manifests lacking the complete fingerprint require a fresh run.
  const stable = value => JSON.stringify(value, (_key,v) => v && !Array.isArray(v) && typeof v === 'object'
    ? Object.fromEntries(Object.keys(v).sort().map(k => [k,v[k]])) : v);
  const refuseResume = message => { const error = new Error(message); error.code = 'RENDERER_PROVENANCE'; throw error; };
  if (manifest.renderer && stable(manifest.renderer) !== stable(environment)) refuseResume('Renderer provenance/environment mismatch; resume refused without modifying manifest');
  if (manifest.captures.some(e => e.status === 'rendered') && !manifest.renderer) refuseResume('Missing renderer provenance for existing PNGs; resume refused');
  manifest.renderer ??= environment;
  const staged = [];
  let staging;
  let browserFailure;
  for (const entry of manifest.captures) {
    if (entry.status !== 'captured-ansi') continue;
    let context, closing;
    const closeContext = () => closing ??= context ? closeBrowserResource(context, 'browser context cleanup') : Promise.resolve();
    try {
      if (browserFailure) throw browserFailure;
      const ansi = await readFile(join(output, entry.files.ansi.path));
      if (hash(ansi) !== entry.files.ansi.sha256) throw new Error('ANSI hash changed before render');
      if (privatePattern.test(ansi.toString())) throw new Error('Private fixture value in ANSI');
      context = await browserDeadline('browser context creation', () => browser.newContext({ viewport: { width: 2300, height: 1500 }, deviceScaleFactor: 1,
        serviceWorkers: 'block', locale: 'en-US', timezoneId: 'UTC', colorScheme: entry.theme === 'light' ? 'light' : 'dark' }),
        undefined, late => closeBrowserResource(late, 'late browser context cleanup'));
      const page = await browserDeadline('browser page creation', () => context.newPage(),
        undefined, late => closeBrowserResource(late, 'late browser page cleanup'));
      page.setDefaultTimeout(10000);
      await browserDeadline('native request routing', () => page.route('**/*', route => route.abort('blockedbyclient')));
      const errors = [];
      page.on('pageerror', error => errors.push(error.message));
      await browserDeadline('native page content', () => page.setContent('<!doctype html><meta charset="utf-8"><div id="terminal"></div>'));
      await browserDeadline('native page styles', () => page.addStyleTag({ content: css + fontCSS + '\nhtml,body{margin:0;padding:0}#terminal{width:max-content}.xterm-viewport{overflow:hidden!important}' }));
      await browserDeadline('native terminal script', () => page.addScriptTag({ content: js }));
      const frame = await browserDeadline('terminal font/paint', () => page.evaluate(async ({ cols, rows, theme, bytes }) => {
        await Promise.all(['16px CaptureMono', 'bold 16px CaptureMono', 'italic 16px CaptureMono', 'bold italic 16px CaptureMono'].map(font => document.fonts.load(font)));
        await document.fonts.ready;
        if (!document.fonts.check('16px CaptureMono')) throw new Error('Bundled terminal font is not ready');
        const terminal = new Terminal({ cols, rows, fontFamily: 'CaptureMono', fontSize: 16, lineHeight: 1.2,
          letterSpacing: 0, fontWeight: '400', fontWeightBold: '700', theme, cursorBlink: false, scrollback: 0,
          allowProposedApi: true });
        terminal.open(document.getElementById('terminal'));
        await new Promise(resolve => terminal.write(new Uint8Array(bytes), resolve));
        // The write callback drains parsing; the render event drains painting.
        await new Promise(resolve => {
          const listener = terminal.onRender(() => { listener.dispose(); resolve(); });
          terminal.refresh(0, rows - 1);
        });
        await new Promise(requestAnimationFrame);
        const buffer = terminal.buffer.active;
        if (buffer.type !== 'alternate') throw new Error('Capture did not retain the native alt screen');
        const cells = [];
        const text = [];
        for (let y = 0; y < rows; y++) {
          const line = buffer.getLine(y);
          text.push(line.translateToString(true));
          const row = [];
          for (let x = 0; x < cols; x++) {
            const cell = line.getCell(x);
            row.push({ text: cell.getChars(), width: cell.getWidth(), fg: cell.getFgColor(), bg: cell.getBgColor(),
              fg_mode: cell.getFgColorMode(), bg_mode: cell.getBgColorMode(), bold: !!cell.isBold(),
              italic: !!cell.isItalic(), underline: !!cell.isUnderline(), inverse: !!cell.isInverse(), dim: !!cell.isDim() });
          }
          cells.push(row);
        }
        const screen = document.querySelector('.xterm-screen').getBoundingClientRect();
        return { cols, rows, buffer: buffer.type, text, cells, theme: terminal.options.theme,
          metrics: { cell_width_px: screen.width / cols, cell_height_px: screen.height / rows,
            width_px: screen.width, height_px: screen.height, device_pixel_ratio: devicePixelRatio,
            font_family: terminal.options.fontFamily, font_size_px: terminal.options.fontSize,
            line_height: terminal.options.lineHeight, measured: terminal._core._renderService.dimensions.css } };
      }, { cols: entry.cols, rows: entry.rows, theme: themes[entry.theme], bytes: [...ansi] }));
      if (!frame.text.some(line => line.includes('oneline-ccc'))) throw new Error('Empty native cell frame');
      if (privatePattern.test(frame.text.join('\n'))) throw new Error('Private fixture in visible cells; refusing PNG');
      const frameData = json(frame);
      const frameName = `${entry.id}.frame.json`;
      const pngName = `${entry.id}.png`;
      const png = await browserDeadline('native screenshot', () => page.locator('.xterm-screen').screenshot({ animations: 'disabled' }));
      if (errors.length) throw new Error(errors.join('; '));
      const validation = await validateEvidence(entry,frame,png,ansi);
      // Context cleanup is a success gate too. Never give screenshot() a path:
      // a late browser result must not be able to publish evidence on its own.
      await closeContext();
      // Stage on disk, outside the published artifact names and manifest. This
      // also bounds memory for large batches while owned-browser cleanup runs.
      staging ??= await mkdtemp(join(output, '.render-'));
      await writeFile(join(staging, frameName), frameData);
      await writeFile(join(staging, pngName), png);
      staged.push({ entry, validation,
        frame: { path: frameName, sha256: hash(frameData), bytes: Buffer.byteLength(frameData) },
        png: { path: pngName, sha256: hash(png), bytes: png.length },
        terminal: { ...frame.metrics, colors: frame.theme, term: 'xterm-256color', encoding: 'UTF-8' } });
    } catch (error) {
      if (error.code === 'BROWSER_DEADLINE') browserFailure = error;
      entry.status = 'failed';
      entry.failure = String(error.message || error);
      console.error(`FAIL ${entry.id}: ${entry.failure}`);
    } finally {
      try { await closeContext(); }
      catch (error) {
        browserFailure = error;
        entry.status = 'failed';
        if (!entry.failure?.includes(error.message)) entry.failure = entry.failure ? `${entry.failure}; ${error.message}` : error.message;
      }
      await writeFile(manifestPath, json(manifest));
    }
  }
  try {
    try { await finish(); }
    catch (error) {
      // Historical rows and fingerprints remain immutable on cleanup failure.
      // Staged new work has never been published or marked as rendered.
      for (const { entry } of staged) {
        entry.status = 'failed'; entry.failure = error.message;
      }
      await writeFile(manifestPath, json(manifest));
      throw error;
    }
    for (const result of staged) {
      const { entry } = result;
      for (const kind of ['frame', 'png']) {
        await rename(join(staging, result[kind].path), join(output, result[kind].path));
        entry.files[kind] = result[kind];
      }
      entry.evidence_validation = result.validation;
      entry.terminal = result.terminal;
      entry.privacy = 'raw stream, fixture metadata and visible cells checked; PNG rendered only from those cells';
      entry.status = 'rendered';
      await writeFile(manifestPath, json(manifest));
    }
  } finally { if (staging) await rm(staging, { recursive: true, force: true }); }
  // A filesystem-backed contact sheet links original PNGs; it never redraws UI.
  const esc = s => String(s).replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('"', '&quot;');
  const cards = manifest.captures.filter(e => e.status === 'rendered').map(e =>
    `<figure><figcaption>${esc(e.id)} · ${esc(e.visual_review)}</figcaption><a href="${esc(e.files.png.path)}"><img loading="lazy" src="${esc(e.files.png.path)}"></a></figure>`).join('\n');
  await writeFile(join(output, 'index.html'), `<!doctype html><meta charset="utf-8"><title>Actual native PTY captures</title><style>body{background:#161b22;color:#eee;font:14px monospace}figure{margin:24px 0}img{max-width:100%}figcaption{margin:8px 0}</style><h1>Actual native PTY captures</h1><p>Generated evidence. Visual acceptance remains open. Original PNGs below; browser references are separate.</p>${cards}`);
  if (manifest.captures.some(e => e.status === 'failed')) process.exitCode = 1;
}

function previewRequest(url, output, theme = 'dark', screen = 'dashboard') {
  if (!['dark', 'light', 'mono'].includes(theme) || !['dashboard', 'models', 'review', 'jobs'].includes(screen)) throw new Error('Unsupported existing browser reference');
  const parsed = new URL(url);
  if (!['127.0.0.1', 'localhost', '[::1]'].includes(parsed.hostname)) throw new Error('Preview capture requires a local server');
  if (!output.startsWith(join(root, 'artifacts/') ) && !output.startsWith(join(root, '.cache/'))) throw new Error('Preview output must stay in ignored worktree output roots');
  return parsed;
}

async function preview(url, output, browser, theme = 'dark', screen = 'dashboard', finish = async () => {}) {
  // Reference-only entry point. It never passes this HTML through the native path.
  const parsed = previewRequest(url, output, theme, screen);
  await mkdir(output, { recursive: true });
  const context = await browserDeadline('reference context creation', () => browser.newContext({ viewport: { width: 1440, height: 1000 }, deviceScaleFactor: 1, serviceWorkers: 'block' }),
    undefined, late => closeBrowserResource(late, 'late reference context cleanup'));
  let closing;
  const closeContext = () => closing ??= closeBrowserResource(context, 'reference context cleanup');
  try {
    const page = await browserDeadline('reference page creation', () => context.newPage(),
      undefined, late => closeBrowserResource(late, 'late reference page cleanup'));
    page.setDefaultTimeout(10000);
    const fetched = [], fetches = [];
    page.on('response', response => {
      fetches.push((async () => {
        const body = await browserDeadline('reference response body', () => response.body());
        fetched.push({ url: response.url(), status: response.status(), type: response.request().resourceType(),
          content_type: response.headers()['content-type'] || '', bytes: body.length, sha256: hash(body) });
      })().then(() => null, error => error));
    });
    await browserDeadline('reference request routing', () => page.route('**/*', route => new URL(route.request().url()).origin === parsed.origin ? route.continue() : route.abort()));
    await browserDeadline('reference navigation', () => page.goto(url, { waitUntil: 'networkidle' }));
    await browserDeadline('reference screen readiness', () => page.locator('#screen h2').waitFor());
    await browserDeadline('reference theme selection', () => page.locator('#theme').selectOption(theme));
    await browserDeadline('reference screen selection', () => page.locator(`[data-screen="${screen}"]`).click());
    await browserDeadline('reference fonts', () => page.evaluate(() => document.fonts.ready));
    const png = await browserDeadline('reference screenshot', () => page.screenshot({ fullPage: true }));
    // Stop new response events before draining provenance and publishing bytes.
    await closeContext();
    for (let processed=0; processed<fetches.length;) {
      const end=fetches.length;
      const results = await browserDeadline('reference response provenance', () => Promise.all(fetches.slice(processed,end)));
      const failure = results.find(result => result); if (failure) throw failure;
      processed=end;
    }
    if (!fetched.some(resource => resource.type === 'document')) throw new Error('Missing fetched reference document provenance');
    const record = { kind: 'browser-mockup', url, theme, screen, png_sha256: hash(png),
      capture_tool_source_sha: execFileSync('git', ['rev-parse', 'HEAD'], { cwd: root, encoding: 'utf8' }).trim(),
      capture_tool_source_dirty: execFileSync('git', ['status', '--porcelain', '--untracked-files=all'], { cwd: root, encoding: 'utf8' }).trim(),
      capture_tool_render_sha256: hash(await readFile(new URL('./render.mjs', import.meta.url))),
      capture_tool_lifecycle_sha256: hash(await readFile(new URL('./browser-lifecycle.mjs', import.meta.url))),
      capture_tool_browser_job_sha256: hash(await readFile(new URL('./browser-job.py', import.meta.url))),
      served_reference: { revision: null, dirty: null, untracked: null, verification: 'Fetched resource hashes only; server checkout revision and local modifications are unknown',
        resources: fetched.sort((a,b)=>a.url.localeCompare(b.url)) },
      chromium: browser.version(), fonts_ready: true, fixture: 'Existing illustrative browser data; not equivalent to native fakeBackend',
      limitation: 'Capture aid only. Matching fixture/state/revision must be established by the comparison reviewer.' };
    await finish();
    await writeFile(join(output, `${screen}-${theme}.png`), png);
    await writeFile(join(output, 'reference.json'), json(record));
  } finally { await closeContext(); }
}

async function main() {
  const [mode, arg, output, theme, screen] = process.argv.slice(2);
  if (!['native', 'preview'].includes(mode) || !arg || mode === 'preview' && !output) throw new Error('Usage: render.mjs native MANIFEST | preview LOCAL_URL OUTPUT [THEME] [SCREEN]');
  if (mode === 'preview') previewRequest(arg, resolve(output), theme, screen);
  const scratch = await browserCaptureScratch(root);
  // Playwright creates its profile before spawning Chromium, so set the Node
  // process environment too; a launch-only env would still use /tmp for it.
  process.env.TMPDIR = scratch;
  process.env.PLAYWRIGHT_BROWSERS_PATH ??= join(cache, 'browsers');
  const { chromium } = await import(pathToFileURL(join(modules, 'playwright/index.mjs')).href);
  let job, failure;
  try {
    job = await launchOwnedBrowser(chromium, { scratch, command: profile => pinnedBrowserCommand(modules, profile),
      env: { ...process.env, TMPDIR: scratch } });
    if (mode === 'native') await native(resolve(arg), job.browser, job.close);
    else await preview(arg, resolve(output), job.browser, theme, screen, job.close);
  } catch (error) { failure = error; }
  finally {
    if (job) {
      try { await job.close(); }
      catch (error) {
        if (!failure) failure = error;
        else if (!failure.message.includes(error.message)) failure = Object.assign(new Error(`${failure.message}; ${error.message}`), { code: failure.code });
      }
    }
    if (!failure) await rm(scratch, { recursive: true, force: true });
  }
  if (failure) {
    if (failure.code === 'RENDERER_PROVENANCE') throw failure;
    // Launch/connect can fail before the renderer environment can be verified.
    // Record honest failure separately, preserving immutable resume manifests.
    const failureOutput = mode === 'native' ? dirname(resolve(arg)) : resolve(output);
    if (mode === 'preview') await mkdir(failureOutput, { recursive: true });
    await writeFile(join(failureOutput, 'render-failure.json'), json({ status: 'failed', failure: failure.message }));
    throw failure;
  }
}
main().catch(error => { console.error(error.message); process.exitCode = 1; });
