// Every positive uses a real fixture/production View/PTY/PNG from the runner.
import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { dirname,join } from 'node:path';
import { validateEvidence } from './evidence.mjs';

const paths=JSON.parse(process.env.A2_SCENE_MANIFESTS || '{}');
const data={};
for(const [name,path] of Object.entries(paths)) {
  const manifest=JSON.parse(await readFile(path,'utf8')),entry=manifest.captures[0],folder=dirname(path);
  data[name]={entry,frame:JSON.parse(await readFile(join(folder,entry.files.frame.path),'utf8')),
    png:await readFile(join(folder,entry.files.png.path)),ansi:await readFile(join(folder,entry.files.ansi.path))};
}
const clone=d=>({...d,entry:structuredClone(d.entry),frame:structuredClone(d.frame)});
const check=d=>validateEvidence(d.entry,d.frame,d.png,d.ansi);
const focused=(name,fn)=>test(name,{skip:!Object.keys(paths).length},fn);
for(const name of ['unicode','help-first','help-last','budget-invalid','compact-invalid','endpoint-invalid','filter-empty','palette-empty','review-last','review-ack']) {
  focused(`genuine ${name} positive`,async()=>{
    const result=await check(data[name]);assert.match(result.level,/production-view/);
    assert.equal(result.semantic_level,'observed-model-and-production-view');
  });
}
focused('first-page Help relabeled last-page fails on the same modal',async()=>{
  const d=clone(data['help-first']);d.entry.variant=d.entry.fixture.variant='last-page';
  await assert.rejects(check(d),/last.*page|final.*page/i);
});
focused('Help metadata must agree with actual production page marker',async()=>{
  const d=clone(data['help-first']);d.entry.variant=d.entry.fixture.variant='last-page';
  d.entry.fixture.semantic_outcomes.page={number:6,total:6};
  d.entry.fixture.semantic_before.page={number:6,total:6};
  await assert.rejects(check(d),/page.*(marker|View|frame)|page.*mismatch/i);
});
for(const name of ['budget-invalid','compact-invalid','endpoint-invalid']) {
  focused(`${name} body text cannot impersonate status`,async()=>{
    const d=clone(data[name]);
    const borrowed=d.frame.text.find((line,index)=>index>2 && line.trim());
    assert.ok(borrowed,'genuine scene must contain a body row');
    d.entry.fixture.semantic_outcomes.status=borrowed.trim();
    d.entry.fixture.semantic_outcomes.validation.status=borrowed.trim();
    d.entry.fixture.semantic_before=structuredClone(d.entry.fixture.semantic_outcomes);
    await assert.rejects(check(d),/refusal|status/i);
  });
  focused(`${name} refusal must match View, not just metadata`,async()=>{
    const d=clone(data[name]);d.entry.fixture.semantic_outcomes.status='Some other refusal';
    d.entry.fixture.semantic_outcomes.validation.status='Some other refusal';
    d.entry.fixture.semantic_before=structuredClone(d.entry.fixture.semantic_outcomes);
    await assert.rejects(check(d),/refusal|status/i);
  });
  focused(`${name} retains the actual invalid value`,async()=>{
    const d=clone(data[name]);d.entry.fixture.semantic_outcomes.editor.value='500000';
    d.entry.fixture.semantic_outcomes.validation.value='500000';
    d.entry.fixture.semantic_before=structuredClone(d.entry.fixture.semantic_outcomes);
    await assert.rejects(check(d),/invalid.*value|value.*invalid/i);
  });
  focused(`${name} draft mutation is rejected`,async()=>{
    const d=clone(data[name]);d.entry.fixture.semantic_outcomes.validation.after_draft_sha256='f'.repeat(64);
    d.entry.fixture.semantic_before=structuredClone(d.entry.fixture.semantic_outcomes);
    await assert.rejects(check(d),/draft/i);
  });
}
focused('new schema requires semantic observations',async()=>{
  const d=clone(data['help-last']);delete d.entry.fixture.semantic_outcomes;
  await assert.rejects(check(d),/semantic.*outcome|semantic.*observation/i);
});

focused('post-key page drift is rejected even when both pages exist',async()=>{
  const d=clone(data['help-last']);d.entry.fixture.semantic_outcomes.page.number--;
  await assert.rejects(check(d),/outcome changed.*PTY/i);
});

for(const name of ['filter-empty','palette-empty'])focused(`${name} false zero count fails independently`,async()=>{
  const d=clone(data[name]);
  const prop=name==='filter-empty'?'filter':'palette';
  const count=name==='filter-empty'?'matching_clients':'matching_commands';
  d.entry.fixture.semantic_outcomes[prop][count]=1;
  d.entry.fixture.semantic_before=structuredClone(d.entry.fixture.semantic_outcomes);
  await assert.rejects(check(d),/no-results/i);
});
focused('review acknowledgment must actually be complete',async()=>{
  const d=clone(data['review-last']);d.entry.variant=d.entry.fixture.variant='acknowledged';
  await assert.rejects(check(d),/acknowledgment/i);
});

focused('historical semantic proof stays explicitly weaker',async()=>{
  const d=clone(data['help-last']);d.entry.validation_schema=2;
  delete d.entry.fixture.semantic_before;delete d.entry.fixture.semantic_outcomes;
  assert.match((await check(d)).semantic_level,/legacy.*not recorded/);
});
