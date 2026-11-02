// Compatibility is about recorded observations, never reconstructed fixtures.
import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { validateEvidence } from './evidence.mjs';

async function load(path) {
  const bytes=await readFile(path),manifest=JSON.parse(bytes),entry=manifest.captures[0];
  const artifact=kind=>readFile(join(dirname(path),entry.files[kind].path));
  return {entry,frame:JSON.parse(await artifact('frame')),png:await artifact('png'),ansi:await artifact('ansi'),bytes,path};
}
const current=await load(process.env.A2_TEST_MANIFEST);
const clone=d=>({...d,entry:structuredClone(d.entry)});
const check=d=>validateEvidence(d.entry,d.frame,d.png,d.ansi);

test('genuine current evidence retains strict semantic proof',async()=>{
  assert.equal(current.entry.validation_schema,3);
  assert.equal((await check(current)).semantic_level,'observed-model-and-production-view');
});
for(const schema of [1,2])test(`legacy schema ${schema} permits unavailable client observations with weaker proof`,async()=>{
  const d=clone(current);d.entry.validation_schema=schema;
  delete d.entry.fixture.clients;delete d.entry.fixture.semantic_before;delete d.entry.fixture.semantic_outcomes;
  assert.match((await check(d)).semantic_level,/legacy.*not recorded/);
});
for(const state of ['baseline','empty'])test(`current ${state} scene requires recorded clients`,async()=>{
  const d=clone(current);d.entry.state=d.entry.fixture.state=state;delete d.entry.fixture.clients;
  await assert.rejects(check(d),/client.*(missing|required|not reached)|missing.*client/i);
});
test('current schema still requires semantic observations',async()=>{
  const d=clone(current);delete d.entry.fixture.semantic_outcomes;
  await assert.rejects(check(d),/missing semantic outcome/i);
});
for(const schema of [1,2,3])for(const clients of [{},'',0,false,'clients',42,true])test(`schema ${schema} rejects malformed empty clients ${JSON.stringify(clients)}`,async()=>{
  const d=clone(current);d.entry.validation_schema=schema;
  d.entry.state=d.entry.fixture.state='empty';d.entry.fixture.clients=clients;
  await assert.rejects(check(d),/client.*(shape|array|type)/i);
});
for(const clients of [[],null])test(`empty client observations accept ${JSON.stringify(clients)}`,async()=>{
  const d=clone(current);d.entry.state=d.entry.fixture.state='empty';d.entry.fixture.clients=clients;
  assert.equal((await check(d)).semantic_level,'observed-model-and-production-view');
});
for(const schema of [1,2,3])test(`recorded client observations are checked in schema ${schema}`,async()=>{
  const d=clone(current);d.entry.validation_schema=schema;
  d.entry.state=d.entry.fixture.state='missing';
  await assert.rejects(check(d),/client state missing/i);
});
test('unchanged original legacy row without clients remains verifiable',{
  skip:!process.env.A2_LEGACY_MANIFEST,
},async()=>{
  const d=await load(process.env.A2_LEGACY_MANIFEST);
  assert.equal(Object.hasOwn(d.entry.fixture,'clients'),false);
  assert.ok(!d.entry.validation_schema || d.entry.validation_schema<3);
  const result=await check(d);
  assert.match(result.level,/legacy-ansi-only/);
  assert.match(result.semantic_level,/legacy.*not recorded/);
  assert.deepEqual(await readFile(d.path),d.bytes,'historical manifest bytes unchanged');
});
