// R1: authentic native bytes paired with contradictory, plausible observations.
import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {dirname,join} from 'node:path';
import {validateEvidence} from './evidence.mjs';
const paths=JSON.parse(process.env.A2B_SCENE_MANIFESTS || '{}');
async function load(path) {
  const m=JSON.parse(await readFile(path,'utf8')),entry=m.captures[0];
  const file=kind=>readFile(join(dirname(path),entry.files[kind].path));
  return {entry,frame:JSON.parse(await file('frame')),png:await file('png'),ansi:await file('ansi')};
}
const check=d=>validateEvidence(d.entry,d.frame,d.png,d.ansi);
const synchronize=d=>{
  d.entry.fixture.semantic_before=structuredClone(d.entry.fixture.semantic_outcomes);
  d.entry.fixture.preparation.after=structuredClone(d.entry.fixture.semantic_outcomes.dashboard);
};
for(const name of ['filter-match','filter-unicode'])for(const key of ['focused_id','selected_id'])test(`${name}: real excluded Pi cannot be ${key}`,{skip:!paths[name]},async()=>{
  const d=await load(paths[name]);
  assert.ok(d.entry.fixture.clients.some(c=>c.id==='pi'));
  d.entry.fixture.semantic_outcomes.dashboard[key]=key==='selected_id'?'pi':'client:pi';synchronize(d);
  await assert.rejects(check(d),/prepared|focus|selected|production|frame/i);
});
for(const [name,key] of [['pending','A2B_BASELINE_MANIFEST'],['filter-match','A2B_BASELINE_MANIFEST'],['filter-unicode','A2B_UNICODE_BASELINE_MANIFEST']])test(`${name}: genuine baseline bytes reject copied prepared observations`,{skip:!paths[name] || !process.env[key]},async()=>{
  const d=await load(process.env[key]),prepared=await load(paths[name]);
  const unchanged=structuredClone(d.entry.fixture.expected_view),frame=structuredClone(d.frame);
  d.entry.variant=d.entry.fixture.variant=prepared.entry.variant;
  for(const field of ['preparation','semantic_before','semantic_outcomes'])d.entry.fixture[field]=structuredClone(prepared.entry.fixture[field]);
  await assert.rejects(check(d),/prepared|production|frame|control/i);
  assert.deepEqual(d.entry.fixture.expected_view,unchanged);assert.deepEqual(d.frame,frame);
});

for(const [name,key] of [['pending','A2B_BASELINE_MANIFEST'],['filter-match','A2B_BASELINE_MANIFEST'],['filter-unicode','A2B_UNICODE_BASELINE_MANIFEST']])test(`${name}: synchronizing copied View still rejects actual baseline controls`,{skip:!paths[name] || !process.env[key]},async()=>{
  const d=await load(process.env[key]),prepared=await load(paths[name]);
  d.entry.variant=d.entry.fixture.variant=prepared.entry.variant;
  for(const field of ['preparation','semantic_before','semantic_outcomes'])d.entry.fixture[field]=structuredClone(prepared.entry.fixture[field]);
  d.entry.fixture.semantic_outcomes.dashboard.production_view=d.entry.fixture.expected_view.content;synchronize(d);
  await assert.rejects(check(d),/frame|production|control/i);
});
for(const name of ['filter-match','filter-unicode']) {
  test(`${name}: filter-only setup preserves every draft`,{skip:!paths[name]},async()=>{
    const d=await load(paths[name]),before=d.entry.fixture.preparation.before.clients;
    before.find(c=>c.id==='pi').draft_sha256='e'.repeat(64);
    await assert.rejects(check(d),/filter.only|draft|transition/i);
  });
  test(`${name}: changing both selected observations to real Pi still fails`,{skip:!paths[name]},async()=>{
    const d=await load(paths[name]);d.entry.fixture.preparation.before.selected_id='pi';
    d.entry.fixture.semantic_outcomes.dashboard.selected_id='pi';synchronize(d);
    await assert.rejects(check(d),/selected|selection|production/i);
  });
  test(`${name}: missing production observations are refused without upgrading old artifacts`,{skip:!paths[name]},async()=>{
    const d=await load(paths[name]);delete d.entry.fixture.semantic_outcomes.dashboard.production_client_controls;synchronize(d);
    await assert.rejects(check(d),/production.*missing|recapture/i);
  });
}

for(const name of ['filter-match','filter-unicode'])test(`${name}: genuine selected detail row has bounded query proof`,{skip:!paths[name]},async()=>{
  const d=await load(paths[name]);
  // Visible selected identity is a separate claim from an active-filter cue.
  const result=await check(d);
  if(d.entry.fixture.semantic_outcomes.dashboard.production_projection)assert.match(result.semantic_level,/rendered filter indicator and query prefix frame-bound/);
  else {assert.match(d.frame.text.join(' '),/Selected: codex \|/);assert.match(result.semantic_level,/query is model-observed only/);}
});

for(const name of ['pending','filter-match','filter-unicode'])test(`${name}: composed body/control identity projection rejects real Pi relabeling`,{skip:!paths[name]},async()=>{
  const d=await load(paths[name]),o=d.entry.fixture.semantic_outcomes.dashboard,p=o.production_projection;
  if(!p)return;
  p.client_projections.find(c=>c.focused).id='client:pi';synchronize(d);
  await assert.rejects(check(d),/matching|identity|projection|production/i);
});
for(const name of ['pending','filter-match','filter-unicode'])test(`${name}: actual body row mapping cannot be reassigned`,{skip:!paths[name]},async()=>{
  const d=await load(paths[name]),p=d.entry.fixture.semantic_outcomes.dashboard.production_projection;
  if(!p)return;
  const c=p.client_projections.find(c=>c.focused);c.body_indices=[p.body_rows.findIndex(row=>row.text!==c.first_row)];synchronize(d);
  await assert.rejects(check(d),/body|mapping|production/i);
});
