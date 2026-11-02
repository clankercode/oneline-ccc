import { spawn } from 'node:child_process';
import { readFile, mkdtemp, rm } from 'node:fs/promises';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';

// Bounds promises which Playwright's action timeout does not cover. A timed-out
// operation is never resumed; a late resource is disposed through its own handle.
export const browserStepTimeout = 10000;
export const browserCleanupTimeout = 2000;

export async function browserDeadline(label, operation, milliseconds = browserStepTimeout, disposeLate) {
  let timer, expired = false;
  const result = Promise.resolve().then(operation);
  const observed = result.then(value => {
    if (expired && disposeLate) {
      // Observe both late resolution and rejection, including cleanup rejection.
      void Promise.resolve().then(() => disposeLate(value)).catch(() => {});
    }
    return value;
  });
  try {
    return await Promise.race([observed, new Promise((_, reject) => {
      timer = setTimeout(() => {
        expired = true;
        const error = new Error(`${label} deadline exceeded (${milliseconds}ms)`);
        error.code = 'BROWSER_DEADLINE';
        reject(error);
      }, milliseconds);
    })]);
  } finally { clearTimeout(timer); }
}

export function closeBrowserResource(resource, label = 'late browser resource cleanup') {
  return browserDeadline(label, () => resource.close(), browserCleanupTimeout);
}

// Preserve the pinned launcher's headless executable and Chromium switches.
// Only its transport changes: a fresh loopback port instead of a pipe. Import
// pinned tool data explicitly; refuse upgrades until these defaults are reviewed.
export async function pinnedBrowserCommand(modules, profile) {
  const core = join(modules, 'playwright-core');
  if (JSON.parse(await readFile(join(core, 'package.json'), 'utf8')).version !== '1.56.1') throw new Error('Unreviewed Playwright launch defaults');
  const { registry } = await import(pathToFileURL(join(core, 'lib/server/registry/index.js')).href);
  const { chromiumSwitches } = await import(pathToFileURL(join(core, 'lib/server/chromium/chromiumSwitches.js')).href);
  return { executable: registry.findExecutable('chromium-headless-shell').executablePath('javascript'),
    args: [...chromiumSwitches(false), ...(process.platform === 'darwin' ? ['--enable-unsafe-swiftshader'] : []),
      '--headless', '--hide-scrollbars', '--mute-audio',
      '--blink-settings=primaryHoverType=2,availableHoverTypes=2,primaryPointerType=4,availablePointerTypes=4',
      '--no-sandbox', '--disable-dev-shm-usage', `--user-data-dir=${profile}`,
      '--remote-debugging-address=127.0.0.1', '--remote-debugging-port=0', '--no-startup-window'] };
}

// Own the ChildProcess before waiting for launch or connect. launchServer() in
// pinned Playwright does not return a process handle until launch completes;
// racing that promise alone cannot terminate a browser stalled during startup.
export async function launchOwnedBrowser(chromium, { env, scratch, command }) {
  if (process.platform === 'win32') throw new Error('Native capture requires a POSIX owned browser process group');
  const profile = await mkdtemp(join(scratch, 'browser-profile-'));
  let child;
  try {
    const spec = await command(profile);
    child = spawn(spec.executable, spec.args, { env, detached: true, stdio: ['ignore', 'ignore', 'pipe'] });
  } catch (error) { await rm(profile, { recursive: true, force: true }); throw error; }
  let didExit = false, resolveExit, rejectLaunch, resolveLaunch;
  const exited = new Promise(resolve => { resolveExit = resolve; });
  const launched = new Promise((resolve, reject) => { resolveLaunch = resolve; rejectLaunch = reject; });
  child.once('error', error => { didExit = true; resolveExit(); rejectLaunch(error); });
  child.once('close', () => { didExit = true; resolveExit(); rejectLaunch(new Error('Owned browser exited before launch completed')); });
  let stderr = '';
  child.stderr.on('data', data => {
    stderr = (stderr + data).slice(-8192);
    const match = stderr.match(/DevTools listening on (ws:\/\/[^\s]+)/);
    if (!match) return;
    try {
      const endpoint = new URL(match[1]);
      if (!['127.0.0.1', '[::1]'].includes(endpoint.hostname)) throw new Error('Owned browser advertised a non-local endpoint');
      resolveLaunch(endpoint.href);
    } catch (error) { rejectLaunch(error); }
  });
  const signal = name => {
    // detached:true creates this exact owned process group. Never enumerate or
    // match process names, and never signal it again after observing child exit.
    if (!didExit && child.pid) {
      try { process.kill(-child.pid, name); }
      catch (error) { if (error.code !== 'ESRCH') throw error; }
    }
  };
  let closing, browser;
  const close = () => closing ??= (async () => {
    let failure;
    try {
      if (browser) await closeBrowserResource(browser, 'browser connection cleanup');
      signal('SIGTERM');
      await browserDeadline('browser process exit', () => exited, browserCleanupTimeout);
    } catch (error) {
      failure = error;
      try {
        signal('SIGKILL');
        await browserDeadline('owned browser reap', () => exited, browserCleanupTimeout);
      } catch (killError) { failure = new Error(`${failure.message}; ${killError.message}`); }
    } finally { if (didExit) await rm(profile, { recursive: true, force: true }); }
    if (failure) throw failure;
  })();
  try {
    const endpoint = await browserDeadline('browser launch', () => launched);
    browser = await browserDeadline('browser connect',
      () => chromium.connectOverCDP(endpoint, { timeout: browserStepTimeout }), browserStepTimeout,
      late => closeBrowserResource(late, 'late browser connection cleanup'));
    return { browser, close };
  } catch (error) {
    try { await close(); } catch (cleanupError) { throw new Error(`${error.message}; ${cleanupError.message}`); }
    throw error;
  }
}
