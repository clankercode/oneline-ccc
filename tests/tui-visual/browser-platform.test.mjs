import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdir, mkdtemp, writeFile, readFile, rm } from 'node:fs/promises';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import * as lifecycle from './browser-lifecycle.mjs';
import vm from 'node:vm';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../..');

test('deep worktrees use a short shared browser scratch directory', async () => {
  assert.equal(typeof lifecycle.browserCaptureScratch, 'function', 'missing browser socket-path correction');
  const scratch = await lifecycle.browserCaptureScratch(root);
  try {
    assert.ok(Buffer.byteLength(join(scratch, 'org.chromium.Chromium.XXXXXX', 'SingletonSocket')) < 108, 'socket plus NUL must fit sun_path');
    assert.ok(!scratch.includes('/.worktrees/'), 'scratch must not inherit worktree depth');
    assert.match(scratch, /\/\.cache\/browser\/run-/);
  } finally { await rm(scratch, { recursive: true, force: true }); }
});

test('complete Chromium socket byte bound refuses overflow before filesystem writes', async () => {
  const moduleSource = await readFile(new URL('./browser-lifecycle.mjs', import.meta.url), 'utf8');
  const source = moduleSource.slice(moduleSource.indexOf('export async function browserCaptureScratch'), moduleSource.indexOf('// Fixed capture switches')).replace(/^export /gm, '');
  for (const [bytes, unicode] of [[62, false], [63, false], [64, false], [62, true], [63, true]]) {
    const sample = join('/x', '.cache/browser/run-XXXXXX');
    const prefix = unicode ? 'é' : '';
    const checkout = '/' + prefix + 'x'.repeat(bytes - Buffer.byteLength(sample) + 1 - Buffer.byteLength(prefix));
    const scratch = join(checkout, '.cache/browser/run-XXXXXX');
    assert.equal(Buffer.byteLength(scratch), bytes);
    const calls = [];
    const sandbox = { Buffer, dirname, join, execFileSync: () => join(checkout, '.git'),
      mkdir: async (...args) => calls.push(['mkdir', ...args]), mkdtemp: async base => { calls.push(['mkdtemp', base]); return base + 'ABCDEF'; } };
    vm.runInNewContext(source + '\nglobalThis.scratch=browserCaptureScratch;', sandbox);
    if (bytes <= 62) {
      await sandbox.scratch('/ignored-linked-worktree');
      assert.equal(calls.length, 2);
    } else {
      await assert.rejects(sandbox.scratch('/ignored-linked-worktree'), /socket path is too long/);
      assert.deepEqual(calls, [], 'overflow must fail before mkdir or mkdtemp');
    }
  }
});

test('reviewed browser pin uses the public executable path with local capture flags', async () => {
  await mkdir(join(root, '.cache'), { recursive: true });
  const fixture = await mkdtemp(join(root, '.cache/browser-platform-'));
  try {
    for (const name of ['playwright', 'playwright-core']) {
      await mkdir(join(fixture, name));
      await writeFile(join(fixture, name, 'package.json'), JSON.stringify({ version: '1.64.0' }));
    }
    await writeFile(join(fixture, 'playwright/index.mjs'), 'export const chromium={executablePath:()=>"/pinned-fixture/chrome"};');
    const spec = await lifecycle.pinnedBrowserCommand(fixture, '/private/profile');
    assert.equal(spec.executable, '/pinned-fixture/chrome');
    for (const flag of ['--headless', '--no-startup-window', '--remote-debugging-port=0', '--remote-debugging-address=127.0.0.1', '--user-data-dir=/private/profile']) {
      assert.ok(spec.args.includes(flag), `missing ${flag}`);
    }
    await writeFile(join(fixture, 'playwright/package.json'), '{"version":"0.0.0"}');
    await assert.rejects(lifecycle.pinnedBrowserCommand(fixture, '/private/profile'), /Unreviewed Playwright/);
  } finally { await rm(fixture, { recursive: true, force: true }); }
});
