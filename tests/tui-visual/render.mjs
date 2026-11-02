// Development-only renderer: xterm consumes native ANSI; HTML contains no app UI.
import { readFile, writeFile, mkdir } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { dirname, resolve, join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { execFileSync } from 'node:child_process';
import { validateEvidence } from './evidence.mjs';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../..');
const cache = join(root, '.cache/tui-visual');
const modules = join(cache, 'tools/node_modules');
const hash = data => createHash('sha256').update(data).digest('hex');
const json = value => JSON.stringify(value, null, 2) + '\n';
async function bounded(promise, milliseconds = 10000) {
  let timer;
  try {
    return await Promise.race([promise, new Promise((_, reject) => {
      timer = setTimeout(() => reject(new Error('Terminal font/paint deadline exceeded')), milliseconds);
    })]);
  } finally { clearTimeout(timer); }
}
const privatePattern = /sk-capture-DUMMY-secret-never-publish|capture\.invalid|https?:\/\//i;
const themes = {
  dark: { background: '#111820', foreground: '#E6EAF0' },
  light: { background: '#F5F6F8', foreground: '#17212B' },
  mono: { background: '#111111', foreground: '#eeeeee' },
};

async function native(manifestPath, browser) {
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
  const environment = { kind: 'xterm.js DOM renderer', xterm: '5.5.0', playwright: '1.56.1',
    chromium: browser.version(), font: 'JetBrains Mono Latin 5.2.8', font_assets: fontAssets,
    xterm_js_sha256: hash(js), xterm_css_sha256: hash(css), font_size: 16, line_height: 1.2, device_scale_factor: 1,
    unicode_fallback: 'Non-Latin glyphs use Chromium host fallback; compare only identical font/browser environments',
    render_source_sha256: hash(await readFile(new URL('./render.mjs', import.meta.url))),
    evidence_source_sha256: hash(await readFile(new URL('./evidence.mjs', import.meta.url))) };
  // Freeze the environment before any resumed writes, including failure writes.
  // Legacy manifests lacking the complete fingerprint require a fresh run.
  const stable = value => JSON.stringify(value, (_key,v) => v && !Array.isArray(v) && typeof v === 'object'
    ? Object.fromEntries(Object.keys(v).sort().map(k => [k,v[k]])) : v);
  if (manifest.renderer && stable(manifest.renderer) !== stable(environment)) throw new Error('Renderer provenance/environment mismatch; resume refused without modifying manifest');
  if (manifest.captures.some(e => e.status === 'rendered') && !manifest.renderer) throw new Error('Missing renderer provenance for existing PNGs; resume refused');
  manifest.renderer ??= environment;
  for (const entry of manifest.captures) {
    if (entry.status !== 'captured-ansi') continue;
    let context;
    try {
      const ansi = await readFile(join(output, entry.files.ansi.path));
      if (hash(ansi) !== entry.files.ansi.sha256) throw new Error('ANSI hash changed before render');
      if (privatePattern.test(ansi.toString())) throw new Error('Private fixture value in ANSI');
      context = await browser.newContext({ viewport: { width: 2300, height: 1500 }, deviceScaleFactor: 1,
        serviceWorkers: 'block', locale: 'en-US', timezoneId: 'UTC', colorScheme: entry.theme === 'light' ? 'light' : 'dark' });
      const page = await context.newPage();
      page.setDefaultTimeout(10000);
      await page.route('**/*', route => route.abort('blockedbyclient'));
      const errors = [];
      page.on('pageerror', error => errors.push(error.message));
      await page.setContent('<!doctype html><meta charset="utf-8"><div id="terminal"></div>');
      await page.addStyleTag({ content: css + fontCSS + '\nhtml,body{margin:0;padding:0}#terminal{width:max-content}.xterm-viewport{overflow:hidden!important}' });
      await page.addScriptTag({ content: js });
      const frame = await bounded(page.evaluate(async ({ cols, rows, theme, bytes }) => {
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
      const png = await page.locator('.xterm-screen').screenshot({ animations: 'disabled' });
      if (errors.length) throw new Error(errors.join('; '));
      entry.evidence_validation = await validateEvidence(entry,frame,png,ansi);
      // Publish evidence only after content, geometry and paint gates succeed.
      await writeFile(join(output, frameName), frameData);
      await writeFile(join(output, pngName), png);
      entry.files.frame = { path: frameName, sha256: hash(frameData), bytes: Buffer.byteLength(frameData) };
      entry.files.png = { path: pngName, sha256: hash(png), bytes: png.length };
      entry.terminal = { ...frame.metrics, colors: frame.theme, term: 'xterm-256color', encoding: 'UTF-8' };
      entry.privacy = 'raw stream, fixture metadata and visible cells checked; PNG rendered only from those cells';
      entry.status = 'rendered';
    } catch (error) {
      entry.status = 'failed';
      entry.failure = String(error.message || error);
      console.error(`FAIL ${entry.id}: ${entry.failure}`);
    } finally {
      if (context) await context.close();
      await writeFile(manifestPath, json(manifest));
    }
  }
  // A filesystem-backed contact sheet links original PNGs; it never redraws UI.
  const esc = s => String(s).replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('"', '&quot;');
  const cards = manifest.captures.filter(e => e.status === 'rendered').map(e =>
    `<figure><figcaption>${esc(e.id)} · ${esc(e.visual_review)}</figcaption><a href="${esc(e.files.png.path)}"><img loading="lazy" src="${esc(e.files.png.path)}"></a></figure>`).join('\n');
  await writeFile(join(output, 'index.html'), `<!doctype html><meta charset="utf-8"><title>Actual native PTY captures</title><style>body{background:#161b22;color:#eee;font:14px monospace}figure{margin:24px 0}img{max-width:100%}figcaption{margin:8px 0}</style><h1>Actual native PTY captures</h1><p>Generated evidence. Visual acceptance remains open. Original PNGs below; browser references are separate.</p>${cards}`);
  if (manifest.captures.some(e => e.status === 'failed')) process.exitCode = 1;
}

async function preview(url, output, browser, theme = 'dark', screen = 'dashboard') {
  // Reference-only entry point. It never passes this HTML through the native path.
  if (!['dark', 'light', 'mono'].includes(theme) || !['dashboard', 'models', 'review', 'jobs'].includes(screen)) throw new Error('Unsupported existing browser reference');
  const parsed = new URL(url);
  if (!['127.0.0.1', 'localhost', '[::1]'].includes(parsed.hostname)) throw new Error('Preview capture requires a local server');
  if (!output.startsWith(join(root, 'artifacts/') ) && !output.startsWith(join(root, '.cache/'))) throw new Error('Preview output must stay in ignored worktree output roots');
  await mkdir(output, { recursive: true });
  const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }, deviceScaleFactor: 1, serviceWorkers: 'block' });
  try {
    const page = await context.newPage();
    const fetched = [], fetches = [];
    page.on('response', response => {
      fetches.push((async () => {
        const body = await response.body();
        fetched.push({ url: response.url(), status: response.status(), type: response.request().resourceType(),
          content_type: response.headers()['content-type'] || '', bytes: body.length, sha256: hash(body) });
      })());
    });
    await page.route('**/*', route => new URL(route.request().url()).origin === parsed.origin ? route.continue() : route.abort());
    await page.goto(url, { waitUntil: 'networkidle' });
    await page.locator('#screen h2').waitFor();
    await page.locator('#theme').selectOption(theme);
    await page.locator(`[data-screen="${screen}"]`).click();
    await bounded(page.evaluate(() => document.fonts.ready));
    await page.screenshot({ path: join(output, `${screen}-${theme}.png`), fullPage: true });
    const png = await readFile(join(output, `${screen}-${theme}.png`));
    for (let processed=0; processed<fetches.length;) { const end=fetches.length; await Promise.all(fetches.slice(processed,end)); processed=end; }
    if (!fetched.some(resource => resource.type === 'document')) throw new Error('Missing fetched reference document provenance');
    await writeFile(join(output, 'reference.json'), json({ kind: 'browser-mockup', url, theme, screen, png_sha256: hash(png),
      capture_tool_source_sha: execFileSync('git', ['rev-parse', 'HEAD'], { cwd: root, encoding: 'utf8' }).trim(),
      capture_tool_source_dirty: execFileSync('git', ['status', '--porcelain', '--untracked-files=all'], { cwd: root, encoding: 'utf8' }).trim(),
      capture_tool_render_sha256: hash(await readFile(new URL('./render.mjs', import.meta.url))),
      served_reference: { revision: null, dirty: null, untracked: null, verification: 'Fetched resource hashes only; server checkout revision and local modifications are unknown',
        resources: fetched.sort((a,b)=>a.url.localeCompare(b.url)) },
      chromium: browser.version(), fonts_ready: true, fixture: 'Existing illustrative browser data; not equivalent to native fakeBackend',
      limitation: 'Capture aid only. Matching fixture/state/revision must be established by the comparison reviewer.' }));
  } finally { await context.close(); }
}

async function main() {
  await mkdir(join(cache, 'tmp'), { recursive: true });
  // Playwright creates its profile before spawning Chromium, so set the Node
  // process environment too; a launch-only env would still use /tmp for it.
  process.env.TMPDIR = join(cache, 'tmp');
  process.env.PLAYWRIGHT_BROWSERS_PATH ??= join(cache, 'browsers');
  const { chromium } = await import(pathToFileURL(join(modules, 'playwright/index.mjs')).href);
  const browser = await chromium.launch({ headless: true, args: ['--disable-dev-shm-usage'],
    env: { ...process.env, TMPDIR: join(cache, 'tmp') } });
  try {
    const [mode, arg, output, theme, screen] = process.argv.slice(2);
    if (mode === 'native') await native(resolve(arg), browser);
    else if (mode === 'preview') await preview(arg, resolve(output), browser, theme, screen);
    else throw new Error('Usage: render.mjs native MANIFEST | preview LOCAL_URL OUTPUT [THEME] [SCREEN]');
  } finally { await browser.close(); }
}
main().catch(error => { console.error(error.message); process.exitCode = 1; });
