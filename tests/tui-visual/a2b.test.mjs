// Positive controls are fresh production View/PTY/PNG artifacts, never UI mocks.
import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {dirname,join} from 'node:path';
import {validateEvidence} from './evidence.mjs';
const paths=JSON.parse(process.env.A2B_SCENE_MANIFESTS || '{}'),data={};
for(const [name,path] of Object.entries(paths)) {
  const m=JSON.parse(await readFile(path,'utf8'));
  for(const entry of m.captures) {
    const artifact=kind=>readFile(join(dirname(path),entry.files[kind].path));
    data[`${name}-${entry.cols}`]={entry,frame:JSON.parse(await artifact('frame')),png:await artifact('png'),ansi:await artifact('ansi')};
  }
}
const clone=d=>({...d,entry:structuredClone(d.entry)});
const check=d=>validateEvidence(d.entry,d.frame,d.png,d.ansi);
for(const [name,d] of Object.entries(data)) {
  test(`genuine ${name} has observed semantic proof`,async()=>{
    const level=(await check(d)).semantic_level;
    if(name.startsWith('more-'))assert.equal(level,'observed-model-and-production-view');
    else { assert.match(level,/observed-prepared-model-and-frame-controls/);assert.match(level,/setup not PTY/);if(name.startsWith('filter-'))assert.match(level,/query is model-observed only; rendered cue unproven/); }
  });
  if(name.startsWith('more-')) {
    test(`${name} rejects first/last relabeling`,async()=>{
      const bad=clone(d);bad.entry.variant=bad.entry.fixture.variant=d.entry.variant==='baseline'?'last-item':'baseline';
      await assert.rejects(check(bad),/first|last/i);
    });
    test(`${name} rejects truncated destination list used to relabel a boundary`,async()=>{
      const bad=clone(d),more=bad.entry.fixture.semantic_outcomes.more;
      more.destinations=more.destinations.filter(row=>row.id===more.focused_id);
      bad.entry.variant=bad.entry.fixture.variant=d.entry.variant==='baseline'?'last-item':'baseline';
      bad.entry.fixture.semantic_before=structuredClone(bad.entry.fixture.semantic_outcomes);
      await assert.rejects(check(bad),/destination|row|boundary/i);
    });
    for(const key of ['focused_id','enter_screen','focused_label'])test(`${name} rejects false ${key} observation`,async()=>{
      const bad=clone(d);bad.entry.fixture.semantic_outcomes.more[key]='Wrong destination';
      bad.entry.fixture.semantic_before=structuredClone(bad.entry.fixture.semantic_outcomes);
      await assert.rejects(check(bad),/More|destination|focused/i);
    });
  } else {
    for(const key of ['focused_id','selected_id','filter'])test(`${name} rejects changed ${key}`,async()=>{
      const bad=clone(d);bad.entry.fixture.semantic_outcomes.dashboard[key]='incorrect';
      bad.entry.fixture.semantic_before=structuredClone(bad.entry.fixture.semantic_outcomes);
      bad.entry.fixture.preparation.after=structuredClone(bad.entry.fixture.semantic_outcomes.dashboard);
      await assert.rejects(check(bad),/prepared|pending|filter|focus|matching/i);
    });
    test(`${name} rejects false PTY setup provenance`,async()=>{
      const bad=clone(d);bad.entry.fixture.preparation.transport='PTY';
      await assert.rejects(check(bad),/prepar|provenance|PTY/i);
    });
    test(`${name} rejects relabeled baseline with no prepared transition`,async()=>{
      const bad=clone(d);bad.entry.fixture.preparation.before=structuredClone(bad.entry.fixture.preparation.after);
      await assert.rejects(check(bad),/prepar|transition|before/i);
    });
    if(name.startsWith('pending'))test(`${name} rejects fabricated pending draft`,async()=>{
      const bad=clone(d),o=bad.entry.fixture.semantic_outcomes.dashboard;
      const c=o.clients.find(c=>c.id==='codex');c.dirty=false;c.draft_sha256=c.baseline_sha256;
      bad.entry.fixture.semantic_before=structuredClone(bad.entry.fixture.semantic_outcomes);
      bad.entry.fixture.preparation.after=structuredClone(o);
      await assert.rejects(check(bad),/draft|pending/i);
    });
    else test(`${name} rejects false matching observations`,async()=>{
      const bad=clone(d);bad.entry.fixture.semantic_outcomes.dashboard.clients[0].matching=true;
      bad.entry.fixture.semantic_outcomes.filter.matching_clients=999;
      bad.entry.fixture.semantic_before=structuredClone(bad.entry.fixture.semantic_outcomes);
      bad.entry.fixture.preparation.after=structuredClone(bad.entry.fixture.semantic_outcomes.dashboard);
      await assert.rejects(check(bad),/match|filter/i);
    });
  }
}
test('A2b semantic fixtures supplied',{skip:!Object.keys(paths).length},()=>assert.ok(Object.keys(data).length));

for(const [variant,key] of [['pending-focus','A2B_BASELINE_MANIFEST'],['filter-match','A2B_BASELINE_MANIFEST'],['filter-unicode','A2B_UNICODE_BASELINE_MANIFEST']]) {
  test(`actual baseline cannot be relabeled ${variant}`,{skip:!process.env[key]},async()=>{
    const path=process.env[key],m=JSON.parse(await readFile(path,'utf8')),entry=m.captures[0];
    const artifact=kind=>readFile(join(dirname(path),entry.files[kind].path));
    const d={entry,frame:JSON.parse(await artifact('frame')),png:await artifact('png'),ansi:await artifact('ansi')};
    assert.equal((await check(d)).semantic_level,'observed-model-and-production-view');
    entry.variant=entry.fixture.variant=variant;
    await assert.rejects(check(d),/prepared.*(provenance|transition|missing)/i);
  });
}
