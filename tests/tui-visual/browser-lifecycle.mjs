import { spawn } from 'node:child_process';
import { readFile, mkdtemp, rm } from 'node:fs/promises';
import { join } from 'node:path';
import { pathToFileURL, fileURLToPath } from 'node:url';

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

// The Linux subreaper stays alive as the group's identity anchor until every
// descendant has been reaped. A browser leader's exit or inherited-FD closure
// cannot complete cleanup. Node sends control through this owned pipe, never a
// numeric group signal that could race a recycled PGID after supervisor exit.
export async function launchOwnedBrowser(chromium, { env, scratch, command }) {
  if (process.platform !== 'linux') throw new Error('Native capture requires a Linux browser job subreaper');
  const profile = await mkdtemp(join(scratch, 'browser-profile-'));
  let child;
  try {
    const spec = await command(profile);
    child = spawn('python3', [fileURLToPath(new URL('./browser-job.py', import.meta.url)), JSON.stringify([spec.executable, ...spec.args])],
      { env, detached: true, stdio: ['pipe', 'pipe', 'pipe'] });
  } catch (error) { await rm(profile, { recursive: true, force: true }); throw error; }
  let didExit = false, drained = false, supervisorFailure, resolveExit, rejectLaunch, resolveLaunch;
  const exited = new Promise(resolve => { resolveExit = resolve; });
  const launched = new Promise((resolve, reject) => { resolveLaunch = resolve; rejectLaunch = reject; });
  child.once('error', error => { supervisorFailure = error; didExit = true; resolveExit(); rejectLaunch(error); });
  child.once('close', code => {
    didExit = true;
    if (code !== 0 || !drained) supervisorFailure ??= new Error('Owned browser supervisor exited without complete subtree reap');
    resolveExit(); rejectLaunch(supervisorFailure ?? new Error('Owned browser exited before launch completed'));
  });
  // The job may finish between requesting closure and writing its control pipe.
  child.stdin.on('error', error => { if (error.code !== 'EPIPE') supervisorFailure ??= error; });
  let status = '';
  child.stdout.on('data', data => {
    status += data;
    while (status.includes('\n')) {
      const end = status.indexOf('\n'), line = status.slice(0, end); status = status.slice(end + 1);
      try {
        const message = JSON.parse(line);
        if (message.event === 'drained') drained = true;
        else if (message.event === 'error') { supervisorFailure = new Error(message.message); rejectLaunch(supervisorFailure); }
        else if (message.event === 'leader-exit') rejectLaunch(new Error('Owned browser leader exited before launch completed'));
      } catch (error) { supervisorFailure = error; rejectLaunch(error); }
    }
  });
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
  const request = mode => {
    if (!didExit) child.stdin.write(mode + '\n');
  };
  const completed = async () => {
    await exited;
    if (supervisorFailure) throw supervisorFailure;
    if (!drained) throw new Error('Owned browser subtree completion was not acknowledged');
  };
  let closing, browser;
  const close = () => closing ??= (async () => {
    let failure;
    try {
      if (browser) await closeBrowserResource(browser, 'browser connection cleanup');
      request('terminate');
      await browserDeadline('browser process exit', completed, browserCleanupTimeout);
    } catch (error) {
      failure = error;
      try {
        request('kill');
        await browserDeadline('owned browser reap', completed, browserCleanupTimeout);
      } catch (killError) { failure = new Error(`${failure.message}; ${killError.message}`); }
    } finally { if (didExit && drained && !supervisorFailure) await rm(profile, { recursive: true, force: true }); }
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
