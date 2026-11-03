import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdir, mkdtemp, writeFile, rm } from 'node:fs/promises';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import * as lifecycle from './browser-lifecycle.mjs';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../..');

test('deep worktrees use a short shared browser scratch directory', async () => {
  assert.equal(typeof lifecycle.browserCaptureScratch, 'function', 'missing browser socket-path correction');
  const scratch = await lifecycle.browserCaptureScratch(root);
  try {
    assert.ok(Buffer.byteLength(scratch) <= 64, 'Chromium adds a Unix socket suffix to TMPDIR');
    assert.ok(!scratch.includes('/.worktrees/'), 'scratch must not inherit worktree depth');
    assert.match(scratch, /\/\.cache\/browser\/run-/);
  } finally { await rm(scratch, { recursive: true, force: true }); }
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
