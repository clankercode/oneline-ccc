// Lifecycle fault tests use the real renderer with an in-memory browser boundary.
// No browser, native fixture, Go process, network, or screenshot is started here.
import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { dirname, resolve, join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import vm from 'node:vm';
import { EventEmitter } from 'node:events';
import { spawn as spawnChild } from 'node:child_process';

const file = fileURLToPath(new URL('./render.mjs', import.meta.url));
const source = await readFile(file, 'utf8');
const hash = data => createHash('sha256').update(data).digest('hex');
const deferred = () => {
  let resolve, reject;
  const promise = new Promise((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
};

async function adapter(stall, options = {}) {
  const timers = new Map(); let nextTimer = 0;
  const writes = new Map(), calls = [], removals = [];
  const manifestPath = resolve(dirname(file), '../../.cache/a18/manifest.json');
  const ansi = Buffer.from('\u001b[?1049honeline-ccc');
  const manifest = structuredClone(options.manifest ?? { kind: 'native-pty', captures: [{ id: 'test-native', theme: 'dark', cols: 1, rows: 1,
    status: 'captured-ansi', files: { ansi: { path: 'native.ansi', sha256: hash(ansi) } } }] });
  const held = deferred();
  const call = (name, value) => { calls.push(name); return name === stall ? held.promise : Promise.resolve(value); };
  const events = new Map();
  const page = {
    setDefaultTimeout() {}, on: (name, handler) => events.set(name, handler), close: () => call('page.close'),
    route: () => call('page.route'), setContent: () => call('page.setContent'),
    addStyleTag: () => call('page.addStyleTag'), addScriptTag: () => call('page.addScriptTag'),
    evaluate: () => call('page.evaluate', { text: ['oneline-ccc'], metrics: {}, theme: {} }),
    screenshot: () => call('page.screenshot', Buffer.from('test-only-png')),
    goto: () => {
      events.get('response')?.({ url: () => 'http://127.0.0.1:12345/reference', status: () => 200,
        request: () => ({ resourceType: () => 'document' }), headers: () => ({}),
        body: () => call('response.body', Buffer.from('test-only-document')) });
      return call('page.goto');
    },
    locator: () => ({ screenshot: () => call('page.screenshot', Buffer.from('test-only-png')),
      waitFor: () => call('page.waitFor'), selectOption: () => call('page.selectOption'), click: () => call('page.click') }),
  };
  const context = { newPage: () => call('context.newPage', page), close: () => call('context.close') };
  const browser = { version: () => 'fake-lifecycle-browser', newContext: () => call('browser.newContext', context) };
  const lifecycleURL = new URL('./browser-lifecycle.mjs', import.meta.url);
  const sandbox = { createHash, dirname, resolve, join, fileURLToPath, pathToFileURL, Buffer, URL,
    readFile: async path => {
      const key = path instanceof URL ? fileURLToPath(path) : path;
      if (key === manifestPath) return JSON.stringify(manifest);
      if (String(key).endsWith('.ansi')) return ansi;
      if (String(key).endsWith('browser-lifecycle.mjs') && options.lifecycleSource) return options.lifecycleSource;
      if (String(key).endsWith('browser-job.py') && options.browserJobSource) return options.browserJobSource;
      return Buffer.from('test-only-pinned-asset');
    },
    writeFile: async (path, data) => writes.set(path, data), mkdir: async () => {},
    mkdtemp: async prefix => prefix + 'test-only-stage',
    rename: async (from, to) => { assert.ok(writes.has(from)); writes.set(to, writes.get(from)); writes.delete(from); },
    rm: async path => { removals.push(path); for (const key of writes.keys()) if (key.startsWith(path + '/')) writes.delete(key); },
    validateEvidence: async () => ({ level: 'test-only-boundary' }),
    execFileSync: (_command, args) => args[0] === 'rev-parse' ? '0'.repeat(40) : '',
    setTimeout: (fn, milliseconds) => {
      assert.ok(Number.isFinite(milliseconds) && milliseconds > 0 && milliseconds <= 10000, 'deadline must be finite and within its budget');
      timers.set(++nextTimer, fn); return nextTimer;
    }, clearTimeout: id => timers.delete(id),
    process: { env: {}, exitCode: 0 }, console: { error() {} },
  };
  // The lifecycle helper accepts the same fake clock as the renderer when loaded
  // into this sandbox, so deadline tests never depend on scheduler speed.
  const lifecycleSource = (await readFile(lifecycleURL, 'utf8')).replace(/^import .*;\n/gm, '').replace(/^export /gm, '')
    .replaceAll('import.meta.url', JSON.stringify(lifecycleURL.href));
  const code = source.replace(/^import .*;\n/gm, '').replaceAll('import.meta.url', JSON.stringify(pathToFileURL(file).href))
    .replace(/const \{ chromium \} = await import\([^;]+;/, 'const { chromium } = globalThis.testPlaywright;')
    .replace('profile => pinnedBrowserCommand(modules, profile)', 'globalThis.testCommand')
    .replace(/main\(\)\.catch\([\s\S]*$/, '');
  vm.createContext(sandbox);
  vm.runInContext(lifecycleSource + '\n' + code + '\nglobalThis.nativeRun=native;globalThis.previewRun=preview;globalThis.launchRun=launchOwnedBrowser;globalThis.mainRun=main;', sandbox);
  let settled = false, failure;
  const run = finish => sandbox.nativeRun(manifestPath, browser, finish).then(() => { settled = true; }, error => { settled = true; failure = error; });
  const flush = () => new Promise(setImmediate);
  const finish = async () => {
    await flush();
    for (let i = 0; !settled && i < 20; i++) {
      assert.ok(timers.size, `renderer stalled at ${stall} without a finite deadline`);
      const [id, fn] = timers.entries().next().value; timers.delete(id); fn(); await flush();
    }
    assert.ok(settled, 'renderer did not settle within its finite lifecycle deadlines');
    if (failure) throw failure;
  };
  const drive = async promise => {
    let done = false, result, error;
    void promise.then(value => { done = true; result = value; }, reason => { done = true; error = reason; });
    await flush();
    for (let i = 0; !done && i < 30; i++) {
      assert.ok(timers.size, 'lifecycle promise stalled without a finite deadline');
      const [id, fn] = timers.entries().next().value; timers.delete(id); fn(); await flush();
    }
    assert.ok(done, 'lifecycle did not settle');
    if (error) throw error;
    return result;
  };
  const tick = async () => {
    assert.ok(timers.size, 'expected an active lifecycle deadline');
    const [id, fn] = timers.entries().next().value; timers.delete(id); fn(); await flush();
  };
  return { run, finish, flush, drive, tick, held, writes, calls, removals, context, page, browser, manifest, manifestPath,
    preview: () => sandbox.previewRun('http://127.0.0.1:12345/reference', join(dirname(manifestPath), 'reference'), browser),
    latest: () => JSON.parse(writes.get(manifestPath)), sandbox };
}

test('successful artifacts are published only after the owned browser finishes cleanup', async () => {
  const a = await adapter();
  void a.run(async () => {
    assert.equal([...a.writes.keys()].some(path => dirname(path) === dirname(a.manifestPath) && /\.(png|frame\.json)$/.test(path)), false);
    assert.equal(a.latest().captures[0].status, 'captured-ansi');
  });
  await a.finish();
  assert.equal(a.latest().captures[0].status, 'rendered');
});

test('final browser cleanup failure preserves historical rows and publishes no new artifacts', async () => {
  const prior = await adapter(); void prior.run(); await prior.finish();
  const manifest = prior.latest(), history = structuredClone(manifest.captures[0]);
  manifest.captures.push({ ...structuredClone(history), id: 'new-capture', status: 'captured-ansi',
    files: { ansi: history.files.ansi } });
  const a = await adapter(undefined, { manifest });
  void a.run(async () => { throw new Error('browser server cleanup deadline exceeded'); });
  await assert.rejects(a.finish(), /browser server cleanup deadline/);
  assert.deepEqual(a.latest().captures[0], history);
  assert.deepEqual(a.latest().renderer, manifest.renderer);
  assert.equal(a.latest().captures[1].status, 'failed');
  assert.equal([...a.writes.keys()].some(path => /\.(png|frame\.json)$/.test(path)), false);
});

test('a lifecycle timeout fails remaining pending rows without reusing the unhealthy browser', async () => {
  const a = await adapter('context.newPage');
  a.manifest.captures.push({ ...structuredClone(a.manifest.captures[0]), id: 'next-capture' });
  void a.run(); await a.finish();
  assert.deepEqual(a.latest().captures.map(entry => entry.status), ['failed', 'failed']);
  assert.equal(a.calls.filter(call => call === 'browser.newContext').length, 1);
});

function fakeChromium(stall, { noExit = false, survivingDescendant = false } = {}) {
  const held = deferred(), calls = [];
  let descendantAlive = survivingDescendant;
  const child = new EventEmitter(); child.exitCode = null; child.signalCode = null; child.pid = 123456;
  child.stderr = new EventEmitter(); child.stdout = new EventEmitter(); child.stdin = new EventEmitter();
  const status = message => child.stdout.emit('data', JSON.stringify(message) + '\n');
  const exit = () => { status({ event: 'drained' }); child.exitCode = 0; child.emit('close', 0); };
  const browser = { close: async () => { calls.push('browser.close'); if (stall === 'browser.close') return held.promise; } };
  const chromium = {
    connectOverCDP: async (endpoint, options) => { calls.push(['connect', endpoint, options]); return stall === 'connect' ? held.promise : browser; },
  };
  const spawn = (executable, args, options) => {
    calls.push(['spawn', executable, args, options]);
    if (stall !== 'launch') queueMicrotask(() => child.stderr.emit('data', 'DevTools listening on ws://127.0.0.1:12345/test-owned\n'));
    return child;
  };
  child.stdin.write = data => {
    const signal = data === 'kill\n' ? 'SIGKILL' : 'SIGTERM'; calls.push(signal);
    assert.equal(child.exitCode, null, 'control must not target a finished/recycled supervisor');
    if (signal === 'SIGKILL') descendantAlive = false;
    if (survivingDescendant && signal === 'SIGTERM') status({ event: 'leader-exit', returncode: 0 });
    else if (!noExit && (signal === 'SIGKILL' || stall !== 'process-exit')) exit();
  };
  const configure = a => {
    a.sandbox.spawn = spawn; a.sandbox.process.platform = 'linux';
    a.sandbox.process.kill = () => assert.fail('Node must never signal a numeric/recycled PGID');
    a.sandbox.testPlaywright = { chromium };
    a.sandbox.testCommand = async () => ({ executable: '/test-only-chromium', args: [] });
  };
  const launch = a => {
    configure(a);
    return a.sandbox.launchRun(chromium, { env: {}, scratch: '/test-only-scratch', command: a.sandbox.testCommand });
  };
  return { chromium, browser, child, calls, held, launch, configure, status, descendantAlive: () => descendantAlive };
}

test('leader exit with a stalled descendant cannot complete cleanup or publish evidence', async () => {
  const a = await adapter(), fake = fakeChromium(undefined, { survivingDescendant: true });
  const job = await a.drive(fake.launch(a));
  void a.run(job.close); await a.flush();
  assert.equal(a.latest().captures[0].status, 'captured-ansi', 'leader exit was incorrectly accepted as complete owned-job cleanup');
  await assert.rejects(a.finish(), /process exit deadline/);
  assert.equal(fake.descendantAlive(), false);
  assert.equal(a.latest().captures[0].status, 'failed');
  assert.equal([...a.writes.keys()].some(path => /\.(png|frame\.json)$/.test(path)), false);
});

test('leader exit before cleanup retains the identity anchor until descendant reap', async () => {
  const a = await adapter(), fake = fakeChromium(undefined, { survivingDescendant: true });
  const job = await a.drive(fake.launch(a));
  fake.status({ event: 'leader-exit', returncode: 0 }); await a.flush();
  assert.equal(fake.child.exitCode, null, 'the supervisor must outlive the command leader');
  await assert.rejects(a.drive(job.close()), /process exit deadline/);
  assert.equal(fake.descendantAlive(), false);
  assert.deepEqual(fake.calls.filter(call => typeof call === 'string'), ['browser.close', 'SIGTERM', 'SIGKILL']);
  await assert.rejects(a.drive(job.close()), /process exit deadline/);
  assert.equal(fake.calls.filter(call => call === 'SIGKILL').length, 1, 'no recycled PGID or repeated control after completion');
});

test('supervisor close without subtree completion cannot remove the profile or publish', async () => {
  const a = await adapter(), fake = fakeChromium();
  const job = await a.drive(fake.launch(a));
  fake.child.exitCode = 0; fake.child.emit('close', 0);
  void a.run(job.close);
  await assert.rejects(a.finish(), /without complete subtree reap/);
  assert.equal(a.latest().captures[0].status, 'failed');
  assert.equal(a.removals.some(path => path.includes('browser-profile-')), false);
  assert.equal(fake.calls.includes('SIGTERM'), false);
  assert.equal([...a.writes.keys()].some(path => /\.(png|frame\.json)$/.test(path)), false);
});

for (const stall of ['browser.close', 'process-exit']) {
  test(`${stall} timeout kills and reaps only the owned browser process group`, async () => {
    const a = await adapter(), fake = fakeChromium(stall);
    const job = await a.drive(fake.launch(a));
    await assert.rejects(a.drive(job.close()), /cleanup deadline|process exit deadline/);
    assert.equal(fake.calls.filter(call => call === 'SIGKILL').length, 1);
    assert.equal(fake.child.exitCode, 0);
    await assert.rejects(a.drive(job.close()), /cleanup deadline|process exit deadline/);
    assert.equal(fake.calls.filter(call => call === 'SIGKILL').length, 1);
  });
}

test('browser connect timeout reaps the owned process and closes a late connection', async () => {
  const a = await adapter(), fake = fakeChromium('connect');
  await assert.rejects(a.drive(fake.launch(a)), /connect deadline/);
  assert.equal(fake.child.exitCode, 0);
  fake.held.resolve(fake.browser); await a.flush();
  assert.ok(fake.calls.includes('browser.close'));
});

test('browser launch timeout kills and reaps even when launch never returns', async () => {
  const a = await adapter(), fake = fakeChromium('launch');
  await assert.rejects(a.drive(fake.launch(a)), /launch deadline/);
  assert.equal(fake.child.exitCode, 0);
  const options = fake.calls.find(call => call[0] === 'spawn')[3];
  assert.equal(options.detached, true);
  assert.ok(fake.calls.includes('SIGTERM'));
  fake.child.stderr.emit('data', 'DevTools listening on ws://127.0.0.1:12345/too-late\n'); await a.flush();
  assert.equal(fake.calls.filter(call => call[0] === 'connect').length, 0);
});

test('successful browser close is insufficient without actual owned process exit', async () => {
  const a = await adapter(), fake = fakeChromium(undefined, { noExit: true });
  const job = await a.drive(fake.launch(a));
  await assert.rejects(a.drive(job.close()), /process exit deadline.*reap deadline/);
  assert.ok(fake.calls.includes('SIGKILL'));
});

test('a native page stall and a cleanup stall compose into failed evidence and owned process reap', async () => {
  const a = await adapter('context.newPage'), fake = fakeChromium('browser.close');
  Object.assign(fake.browser, a.browser);
  const job = await a.drive(fake.launch(a));
  void a.run(job.close);
  await assert.rejects(a.finish(), /browser connection cleanup deadline/);
  assert.equal(a.latest().captures[0].status, 'failed');
  assert.match(a.latest().captures[0].failure, /page creation deadline/);
  assert.equal(fake.child.exitCode, 0);
  assert.ok(fake.calls.includes('SIGKILL'));
  assert.equal([...a.writes.keys()].some(path => /\.(png|frame\.json)$/.test(path)), false);
});

async function osDeadline(promise) {
  let timer;
  try { return await Promise.race([promise, new Promise((_, reject) => { timer = setTimeout(() => reject(new Error('tiny process barrier deadline')), 3000); })]); }
  finally { clearTimeout(timer); }
}

for (const variant of ['same group', 'escaped session', 'graceful positive control']) {
  const escape = variant === 'escaped session', stubborn = variant !== 'graceful positive control';
  test(`owned cleanup actually reaps leader-exit descendant (${variant}) and preserves unrelated child`, { timeout: 10000 }, async () => {
    const a = await adapter(), controls = [], statuses = [];
    let child, childClosed, unrelated, unrelatedClosed, descendantPID, leaderPID;
    const leaderExited = deferred();
    try {
      unrelated = spawnChild(process.execPath, ['-e', 'setInterval(() => {}, 1000)'], { stdio: 'ignore' });
      unrelatedClosed = new Promise(resolve => unrelated.once('close', resolve));
      a.sandbox.spawn = (executable, args, options) => {
        child = spawnChild(executable, args, options);
        childClosed = new Promise(resolve => child.once('close', resolve));
        let output = '', errors = '';
        child.stdout.on('data', data => {
          output += data;
          while (output.includes('\n')) {
            const end = output.indexOf('\n'), message = JSON.parse(output.slice(0, end)); output = output.slice(end + 1);
            statuses.push(message);
            if (message.event === 'started') leaderPID = message.pid;
            if (message.event === 'leader-exit') leaderExited.resolve();
          }
        });
        child.stderr.on('data', data => { errors += data; const match = errors.match(/DESCENDANT_PID=(\d+)/); if (match) descendantPID = Number(match[1]); });
        const write = child.stdin.write.bind(child.stdin);
        child.stdin.write = data => { controls.push(data); return write(data); };
        return child;
      };
      a.sandbox.process.platform = process.platform;
      a.sandbox.process.kill = () => assert.fail('no numeric PGID signal from Node');
      const fixture = `import os,signal,sys
reader,writer=os.pipe()
descendant=os.fork()
if descendant == 0:
    os.close(reader)
    ${escape ? 'os.setsid()' : 'pass'}
    signal.signal(signal.SIGTERM,${stubborn ? 'signal.SIG_IGN' : 'signal.SIG_DFL'})
    os.write(writer,b'ready');os.close(writer)
    for fd in (0,1,2): os.close(fd)
    while True: signal.pause()
os.close(writer);os.read(reader,5);os.close(reader)
sys.stderr.write(f'DESCENDANT_PID={descendant}\\nDevTools listening on ws://127.0.0.1:12345/test-only\\n');sys.stderr.flush()
while True: signal.pause()
`;
      const job = await osDeadline(a.sandbox.launchRun({ connectOverCDP: async () => ({ close: async () => {} }) },
        { env: { ...process.env, PYTHONDONTWRITEBYTECODE: '1' }, scratch: '/test-only-scratch', command: async () => ({ executable: 'python3', args: ['-u', '-c', fixture] }) }));
      void a.run(job.close);
      await osDeadline(leaderExited.promise); await a.flush();
      assert.ok(descendantPID && leaderPID);
      if (stubborn) {
        assert.equal(child.exitCode, null, 'identity anchor must remain alive after leader exit');
        assert.equal(statuses.some(status => status.event === 'drained'), false);
        assert.equal(a.latest().captures[0].status, 'captured-ansi');
        await a.tick(); // Deterministic graceful timeout requests pidfd kill/reap.
      }
      await osDeadline(childClosed);
      if (stubborn) {
        await assert.rejects(a.finish(), /process exit deadline/);
        assert.equal(a.latest().captures[0].status, 'failed');
        assert.equal([...a.writes.keys()].some(path => /\.(png|frame\.json)$/.test(path)), false);
      } else {
        await a.finish();
        assert.equal(a.latest().captures[0].status, 'rendered');
      }
      assert.equal(child.exitCode, 0);
      assert.ok(statuses.some(status => status.event === 'reaped' && status.pid === leaderPID));
      assert.ok(statuses.some(status => status.event === 'reaped' && status.pid === descendantPID && status.returncode === (stubborn ? -9 : -15)));
      assert.equal(statuses.at(-1).event, 'drained');
      assert.deepEqual(controls, stubborn ? ['terminate\n', 'kill\n'] : ['terminate\n']);
      assert.equal(unrelated.exitCode, null); assert.equal(unrelated.signalCode, null);
      if (stubborn) await assert.rejects(a.drive(job.close()), /process exit deadline/);
      else await a.drive(job.close());
      assert.equal(controls.length, stubborn ? 2 : 1, 'completed/recycled anchor cannot receive another control');
    } finally {
      // Closing this owned pipe requests subtree kill while keeping its live
      // subreaper/identity anchor available until all descendants are reaped.
      if (child && child.exitCode === null && child.signalCode === null) child.stdin.end();
      if (childClosed) await osDeadline(childClosed);
      if (unrelated) { unrelated.kill('SIGTERM'); await osDeadline(unrelatedClosed); }
    }
  });
}

for (const stall of ['launch', 'connect']) {
  test(`CLI ${stall} timeout records separate failure evidence without rewriting a resume manifest`, async () => {
    const a = await adapter(), fake = fakeChromium(stall); fake.configure(a);
    a.sandbox.process.argv = ['node', file, 'native', a.manifestPath];
    await assert.rejects(a.drive(a.sandbox.mainRun()), /deadline/);
    assert.equal(a.writes.has(a.manifestPath), false);
    const failure = JSON.parse(a.writes.get(join(dirname(a.manifestPath), 'render-failure.json')));
    assert.equal(failure.status, 'failed'); assert.match(failure.failure, new RegExp(stall + ' deadline'));
    assert.equal(fake.child.exitCode, 0);
    assert.equal([...a.writes.keys()].some(path => /\.(png|frame\.json)$/.test(path)), false);
  });
}

test('CLI refuses non-local reference URL/output before spawning or writing failure evidence', async () => {
  for (const [url, output] of [['https://example.invalid/', dirname(file)], ['http://127.0.0.1:12345/', dirname(file)]]) {
    const a = await adapter(), fake = fakeChromium(); fake.configure(a);
    a.sandbox.process.argv = ['node', file, 'preview', url, output];
    await assert.rejects(a.drive(a.sandbox.mainRun()), /local server|ignored worktree output/);
    assert.equal(fake.calls.length, 0); assert.equal(a.writes.size, 0);
  }
});

for (const key of ['lifecycleSource','browserJobSource'])test(`CLI ${key} provenance mismatch preserves all output, including failure evidence`, async () => {
  const first = await adapter(); void first.run(); await first.finish();
  const manifest = first.latest();
  manifest.captures.push({ ...structuredClone(manifest.captures[0]), id: 'pending-new', status: 'captured-ansi' });
  const a = await adapter(undefined, { manifest, [key]: Buffer.from('different capture helper source') });
  const fake = fakeChromium(); Object.assign(fake.browser, a.browser); fake.configure(a);
  a.sandbox.process.argv = ['node', file, 'native', a.manifestPath];
  await assert.rejects(a.drive(a.sandbox.mainRun()), /provenance\/environment mismatch/);
  assert.equal(a.writes.size, 0);
  assert.equal(a.calls.includes('browser.newContext'), false);
  assert.equal(fake.child.exitCode, 0);
});

for (const stall of ['browser.newContext', 'context.newPage', 'page.route', 'page.goto', 'page.waitFor',
  'page.selectOption', 'page.click', 'page.evaluate', 'page.screenshot', 'response.body', 'context.close']) {
  test(`reference ${stall} is bounded and cannot publish PNG/provenance after failure`, async () => {
    const a = await adapter(stall);
    await assert.rejects(a.drive(a.preview()), /deadline/);
    assert.equal([...a.writes.keys()].some(path => path.endsWith('.png') || path.endsWith('reference.json')), false);
  });
}

test('reference positive control still records served document hashes after cleanup', async () => {
  const a = await adapter(); await a.drive(a.preview());
  const record = JSON.parse([...a.writes].find(([path]) => path.endsWith('reference.json'))[1]);
  assert.equal(record.kind, 'browser-mockup');
  assert.equal(record.served_reference.resources[0].sha256, hash(Buffer.from('test-only-document')));
  assert.ok(a.calls.includes('context.close'));
});

for (const stall of ['browser.newContext', 'context.newPage', 'page.route', 'page.setContent',
  'page.addStyleTag', 'page.addScriptTag', 'page.evaluate', 'page.screenshot', 'context.close']) {
  test(`native ${stall} deadline records failure before publishing evidence`, async () => {
    const a = await adapter(stall); void a.run(); await a.finish();
    const entry = a.latest().captures[0];
    assert.equal(entry.status, 'failed');
    assert.match(entry.failure, /deadline|timed out/i);
    assert.equal(a.sandbox.process.exitCode, 1);
    assert.equal([...a.writes.keys()].some(path => /\.(png|frame\.json)$/.test(path)), false);
    assert.equal(entry.files.ansi.sha256, a.manifest.captures[0].files.ansi.sha256);
    assert.equal(entry.files.png, undefined);
    if (stall !== 'browser.newContext') assert.ok(a.calls.includes('context.close'));
  });
}

for (const stall of ['browser.newContext', 'context.newPage']) {
  test(`late ${stall} result is closed and cannot publish after timeout`, async () => {
    const a = await adapter(stall); void a.run(); await a.finish();
    const before = a.writes.get(a.manifestPath);
    a.held.resolve(stall === 'browser.newContext' ? a.context : a.page); await a.flush();
    assert.ok(a.calls.includes(stall === 'browser.newContext' ? 'context.close' : 'page.close'));
    assert.equal(a.writes.get(a.manifestPath), before);
    assert.equal(a.latest().captures[0].status, 'failed');
  });
}
