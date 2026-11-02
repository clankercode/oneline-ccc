// Exercise the actual renderer/export functions with faulted page results.
// The page adapter replaces the browser boundary; it does not draw app layouts.
import test from 'node:test';
import assert from 'node:assert/strict';
import * as fs from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { createHash } from 'node:crypto';
import { deflateSync, inflateSync } from 'node:zlib';
import * as childProcess from 'node:child_process';
import { dirname, resolve, join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import vm from 'node:vm';
import * as lifecycle from './browser-lifecycle.mjs';

const file = fileURLToPath(new URL('./render.mjs', import.meta.url));
const manifestPath = resolve(process.env.A2_TEST_MANIFEST || '.cache/tui-visual/fix1-red-fixture/manifest.json');
const original = JSON.parse(await fs.readFile(manifestPath, 'utf8'));
const saved = original.captures[0];
const folder = dirname(manifestPath);
const savedFrame = JSON.parse(await fs.readFile(join(folder, saved.files.frame.path), 'utf8'));
const savedPNG = await fs.readFile(join(folder, saved.files.png.path));
const savedANSI = await fs.readFile(join(folder, saved.files.ansi.path), 'utf8');

function solidPNG(w, h) {
  function crc(data) {
    let n = 0xffffffff;
    for (const byte of data) { n ^= byte; for (let i=0; i<8; i++) n = (n >>> 1) ^ (n & 1 ? 0xedb88320 : 0); }
    return (n ^ 0xffffffff) >>> 0;
  }
  function chunk(name, bytes) {
    const type = Buffer.from(name), size = Buffer.alloc(4), sum = Buffer.alloc(4);
    size.writeUInt32BE(bytes.length); sum.writeUInt32BE(crc(Buffer.concat([type, bytes])));
    return Buffer.concat([size, type, bytes, sum]);
  }
  const ihdr = Buffer.alloc(13); ihdr.writeUInt32BE(w); ihdr.writeUInt32BE(h,4); ihdr[8]=8; ihdr[9]=2;
  const row = Buffer.alloc(1+w*3); for(let x=1;x<row.length;x+=3) row.set([17,24,32],x);
  return Buffer.concat([Buffer.from('89504e470d0a1a0a','hex'), chunk('IHDR',ihdr),chunk('IDAT',deflateSync(Buffer.concat(Array(h).fill(row)))),chunk('IEND',Buffer.alloc(0))]);
}

async function adapter({ frame = structuredClone(savedFrame), png = savedPNG, version = original.renderer.chromium,
                         manifest = structuredClone(original), contextError = false, referenceBody = null, lifecycleSource = null } = {}) {
  manifest.captures = manifest.captures.slice(0, 1);
  manifest.captures[0].status = 'captured-ansi';
  manifest.captures[0].fixture.expected_view ??= { content: savedFrame.text.join('\n'), alt_screen: true };
  const writes = new Map();
  const calls = { contexts: 0, stages: 0 };
  const events = new Map();
  const page = {
    setDefaultTimeout() {}, route: async () => {}, on: (name, fn) => events.set(name,fn),
    setContent: async () => {}, addStyleTag: async () => {}, addScriptTag: async () => {},
    evaluate: async () => frame,
    goto: async () => {
      if (referenceBody) events.get('response')?.({ url: () => 'http://127.0.0.1:38627/tui-preview.html',
        status: () => 200, headers: () => ({'content-type':'text/html'}), request: () => ({resourceType: () => 'document'}),
        body: async () => Buffer.from(referenceBody) });
    },
    screenshot: async ({path}) => { if(path)writes.set(path, png); return png; },
    locator: () => ({ screenshot: async () => png, waitFor: async () => {}, selectOption: async () => {}, click: async () => {} }),
  };
  const browser = { version: () => version, newContext: async () => {
    calls.contexts++;
    if (contextError) throw new Error('injected context failure');
    return {newPage: async () => page, close: async () => {}};
  }};
  let code = await fs.readFile(file,'utf8');
  code = code.replace(/^import .*;\n/gm,'').replaceAll('import.meta.url',JSON.stringify(pathToFileURL(file).href))
    .replace(/main\(\)\.catch\([\s\S]*$/,'')
    .replace("await import('node:child_process')",'childProcess');
  const evidenceFile = join(dirname(file), 'evidence.mjs');
  const evidence = existsSync(evidenceFile) ? await import(pathToFileURL(evidenceFile)) : {};
  const sandbox = { ...evidence, ...lifecycle, childProcess, execFileSync: childProcess.execFileSync,
    readFile: async (path,...args) => {
      const key=resolve(path instanceof URL ? fileURLToPath(path) : path);
      if(key===manifestPath)return JSON.stringify(manifest);
      if(key===join(dirname(file),'browser-lifecycle.mjs') && lifecycleSource!==null)return lifecycleSource;
      if(writes.has(key))return writes.get(key);
      return fs.readFile(path,...args);
    },
    writeFile: async (path,data) => writes.set(resolve(path),data), mkdir: async () => {}, createHash,
    mkdtemp: async prefix => { calls.stages++; return prefix + 'test-only-stage'; },
    rename: async (from,to) => { writes.set(resolve(to),writes.get(resolve(from))); writes.delete(resolve(from)); },
    rm: async path => { for (const key of writes.keys()) if (key.startsWith(path + '/')) writes.delete(key); },
    dirname,resolve,join,fileURLToPath,pathToFileURL,Buffer,URL,setTimeout,clearTimeout,
    process:{env:{...process.env},exitCode:0}, console:{error(){}}, structuredClone };
  vm.createContext(sandbox); vm.runInContext(code+'\nglobalThis.nativeRun=native;globalThis.previewRun=preview;',sandbox);
  return {run: () => sandbox.nativeRun(manifestPath,browser), preview: out => sandbox.previewRun('http://127.0.0.1:38627/tui-preview.html',out,browser),
    writes, calls, manifest, sandbox, latest: () => JSON.parse(writes.get(manifestPath))};
}

test('renderer rejects a valid background-only PNG before rendered status',async () => {
  const a = await adapter({png:solidPNG(savedFrame.metrics.width_px,savedFrame.metrics.height_px)});
  // This test is about evidence gating, not legacy resume compatibility.
  delete a.manifest.renderer;
  await a.run();
  assert.equal(a.latest().captures[0].status,'failed');
  assert.match(a.latest().captures[0].failure,/blank|glyph/i);
});
test('real production evidence remains accepted as a positive control',async () => {
  const a=await adapter();delete a.manifest.renderer;await a.run();
  assert.equal(a.latest().captures[0].status,'rendered');
  assert.match(a.latest().captures[0].evidence_validation.level,/production-view/);
  assert.equal(a.latest().renderer.lifecycle_source_sha256,createHash('sha256').update(await fs.readFile(new URL('./browser-lifecycle.mjs',import.meta.url))).digest('hex'));
});
test('renderer rejects header-only frame against production output',async () => {
  const frame = structuredClone(savedFrame);
  for(let y=1;y<frame.rows;y++){frame.text[y]='';for(const c of frame.cells[y]){c.text='';c.width=1;}}
  const a = await adapter({frame});delete a.manifest.renderer;await a.run();
  assert.equal(a.latest().captures[0].status,'failed');
  assert.match(a.latest().captures[0].failure,/production View|expected.*frame/i);
});
test('renderer rejects wrong requested scene',async () => {
  const a = await adapter();delete a.manifest.renderer;a.manifest.captures[0].scenario='jobs';await a.run();
  assert.equal(a.latest().captures[0].status,'failed');assert.match(a.latest().captures[0].failure,/scene|target/i);
});
test('renderer rejects cell/text disagreement',async () => {
  const frame=structuredClone(savedFrame);for(const row of frame.cells)for(const cell of row)cell.text='';
  const a=await adapter({frame});delete a.manifest.renderer;await a.run();
  assert.equal(a.latest().captures[0].status,'failed');assert.match(a.latest().captures[0].failure,/cell.*text|text.*cell/i);
});
test('renderer checks measured geometry before success',async () => {
  const frame=structuredClone(savedFrame);frame.metrics.width_px++;
  const a=await adapter({frame});delete a.manifest.renderer;await a.run();
  assert.equal(a.latest().captures[0].status,'failed');assert.match(a.latest().captures[0].failure,/geometry/i);
});
test('mixed resume refuses changed renderer without rewriting old provenance',async () => {
  const a=await adapter({version:'different-chromium',contextError:true});
  const old=structuredClone(a.manifest.captures[0]);old.status='rendered';old.id='old-rendered';a.manifest.captures.unshift(old);
  const oldRenderer=structuredClone(a.manifest.renderer);
  await assert.rejects(a.run(),/renderer.*provenance|environment.*mismatch/i);
  assert.equal(a.writes.size,0);
  assert.deepEqual(a.manifest.renderer,oldRenderer);
});
test('mixed resume refuses changed lifecycle source before context creation or artifact mutation',async () => {
  const first=await adapter();delete first.manifest.renderer;await first.run();
  const manifest=first.latest(),old=structuredClone(manifest.captures[0]);old.id='immutable-old-rendered';
  const oldRenderer=structuredClone(manifest.renderer);
  const a=await adapter({manifest,lifecycleSource:Buffer.from('changed lifecycle helper source'),contextError:true});
  a.manifest.captures.unshift(old);
  await assert.rejects(a.run(),/renderer.*provenance|environment.*mismatch/i);
  assert.equal(a.writes.size,0);
  assert.equal(a.calls.contexts,0);
  assert.equal(a.calls.stages,0);
  assert.deepEqual(a.manifest.renderer,oldRenderer);
  assert.deepEqual(a.manifest.captures[0],old);
});
test('reference records served bytes separately from tool revision',async () => {
  const body='<!doctype html><title>served by another checkout</title>';
  const a=await adapter({referenceBody:body});
  const output=resolve(dirname(file),'../../artifacts/test-reference-metadata');await a.preview(output);
  const record=JSON.parse(a.writes.get(join(output,'reference.json')));
  assert.equal(record.source_sha,undefined);
  assert.match(record.capture_tool_source_sha,/^[0-9a-f]{40}$/);
  assert.equal(record.served_reference.revision,null);
  assert.ok(record.served_reference.resources.some(resource=>resource.sha256===createHash('sha256').update(body).digest('hex')));
  assert.equal(record.capture_tool_lifecycle_sha256,createHash('sha256').update(await fs.readFile(new URL('./browser-lifecycle.mjs',import.meta.url))).digest('hex'));
});

// Re-encode genuine screenshot RGB with a chosen alpha. Valid PNG chunks/CRCs,
// real glyph contrast, unchanged geometry: alpha is the only fault.
function rgbaPNG(png, alpha) {
  let width, height, channels; const idat=[];
  for(let p=8;p<png.length;) {
    const n=png.readUInt32BE(p), kind=png.toString('ascii',p+4,p+8), data=png.subarray(p+8,p+8+n);
    if(kind==='IHDR'){width=data.readUInt32BE(0);height=data.readUInt32BE(4);channels=data[9]===2?3:4;}
    if(kind==='IDAT')idat.push(data);p+=12+n;
  }
  const raw=inflateSync(Buffer.concat(idat)), stride=width*channels, rgb=Buffer.alloc(stride*height);
  const paeth=(a,b,c)=>{const p=a+b-c,ds=[Math.abs(p-a),Math.abs(p-b),Math.abs(p-c)];return ds[0]<=ds[1]&&ds[0]<=ds[2]?a:ds[1]<=ds[2]?b:c;};
  for(let y=0;y<height;y++)for(let x=0;x<stride;x++) {
    const i=y*stride+x, a=x>=channels?rgb[i-channels]:0,b=y?rgb[i-stride]:0,c=y&&x>=channels?rgb[i-stride-channels]:0;
    rgb[i]=(raw[y*(stride+1)+1+x]+[0,a,b,Math.floor((a+b)/2),paeth(a,b,c)][raw[y*(stride+1)]])&255;
  }
  const out=Buffer.alloc((width*4+1)*height);
  for(let y=0;y<height;y++)for(let x=0;x<width;x++) {
    const i=(y*width+x)*channels,j=y*(width*4+1)+1+x*4;
    out.set(rgb.subarray(i,i+3),j);out[j+3]=alpha;
  }
  function chunk(kind,data) {
    const name=Buffer.from(kind), n=Buffer.alloc(4),sum=Buffer.alloc(4);n.writeUInt32BE(data.length);
    let crc=0xffffffff;for(const b of Buffer.concat([name,data])){crc^=b;for(let i=0;i<8;i++)crc=(crc>>>1)^(crc&1?0xedb88320:0);}
    sum.writeUInt32BE((crc^0xffffffff)>>>0);return Buffer.concat([n,name,data,sum]);
  }
  const ihdr=Buffer.alloc(13);ihdr.writeUInt32BE(width);ihdr.writeUInt32BE(height,4);ihdr[8]=8;ihdr[9]=6;
  return Buffer.concat([png.subarray(0,8),chunk('IHDR',ihdr),chunk('IDAT',deflateSync(out)),chunk('IEND',Buffer.alloc(0))]);
}
for(const alpha of [0,128])test(`renderer rejects genuine RGB with alpha ${alpha}`,async()=>{
  const a=await adapter({png:rgbaPNG(savedPNG,alpha)});delete a.manifest.renderer;await a.run();
  assert.equal(a.latest().captures[0].status,'failed');
  assert.match(a.latest().captures[0].failure,/alpha|opaque|transparen/i);
  assert.equal([...a.writes.keys()].some(p=>p.endsWith('.png')||p.endsWith('.frame.json')),false);
});
test('genuine RGB re-encoded as opaque RGBA remains accepted',async()=>{
  const a=await adapter({png:rgbaPNG(savedPNG,255)});delete a.manifest.renderer;await a.run();
  assert.equal(a.latest().captures[0].status,'rendered');
});
for(const state of ['unicode','missing','unsupported','drift','empty'])test(`same-screen ${state} relabeling fails independently of echoed labels`,async()=>{
  const a=await adapter();delete a.manifest.renderer;
  a.manifest.captures[0].state=state;a.manifest.captures[0].fixture.state=state;
  await a.run();assert.equal(a.latest().captures[0].status,'failed');
  assert.match(a.latest().captures[0].failure,/state|client|unicode/i);
});

for(const alpha of [0,128,255])test(`standalone verifier checks genuine RGBA alpha ${alpha} with updated hashes`,async()=>{
  const folder=await fs.mkdtemp(join(dirname(file),'../../.cache/tui-visual/tmp/rgba-fault-'));
  try {
    const manifest=structuredClone(original);manifest.captures=[structuredClone(saved)];
    const row=manifest.captures[0],png=rgbaPNG(savedPNG,alpha);
    for(const record of Object.values(row.files))await fs.copyFile(join(dirname(manifestPath),record.path),join(folder,record.path));
    await fs.writeFile(join(folder,row.files.png.path),png);
    row.files.png.sha256=createHash('sha256').update(png).digest('hex');row.files.png.bytes=png.length;
    const path=join(folder,'manifest.json');await fs.writeFile(path,JSON.stringify(manifest));
    const result=childProcess.spawnSync('python3',[join(dirname(file),'verify.py'),path],{encoding:'utf8',env:{...process.env,PYTHONDONTWRITEBYTECODE:'1'}});
    if(alpha===255)assert.equal(result.status,0,result.stderr);
    else {assert.equal(result.status,1);assert.match(result.stderr,/alpha|opaque|transparen/i);assert.doesNotMatch(result.stderr,/hash.*mismatch/i);}
  } finally {await fs.rm(folder,{recursive:true,force:true});}
});

test('Unicode name metadata cannot substitute for visible Unicode on Dashboard',async()=>{
  const a=await adapter();delete a.manifest.renderer;
  const e=a.manifest.captures[0];e.state=e.fixture.state='unicode';
  e.fixture.clients[0].name='Codex 模型 — café · a deliberately long client name';
  await a.run();assert.equal(a.latest().captures[0].status,'failed');
  assert.match(a.latest().captures[0].failure,/state|unicode/i);
});

test('legacy rows without clients still reject a valid transparent PNG',async()=>{
  const a=await adapter({png:rgbaPNG(savedPNG,0)});delete a.manifest.renderer;
  const e=a.manifest.captures[0];e.validation_schema=2;
  delete e.fixture.clients;delete e.fixture.semantic_before;delete e.fixture.semantic_outcomes;
  await a.run();assert.equal(a.latest().captures[0].status,'failed');
  assert.match(a.latest().captures[0].failure,/alpha|opaque|transparen/i);
});
