import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {join,dirname} from 'node:path';
import {validateEvidence} from './evidence.mjs';
const paths=JSON.parse(process.env.A2C_MODELS_MANIFESTS || '{}');
for(const [variant,path] of Object.entries(paths)) {
  const m=JSON.parse(await readFile(path,'utf8'));
  for(const entry of m.captures.filter(e=>e.status==='rendered')) {
    const file=kind=>readFile(join(dirname(path),entry.files[kind].path));
    const d={entry,frame:JSON.parse(await file('frame')),png:await file('png'),ansi:await file('ansi')};
    const check=d=>validateEvidence(d.entry,d.frame,d.png,d.ansi),clone=()=>({...d,entry:structuredClone(entry)});
    test(`genuine ${variant}/${entry.cols}`,async()=>assert.match((await check(d)).semantic_level,/observed-Models-disclosure/));
    for(const key of ['selected_id','focused_id','draft_sha256'])test(`${variant}/${entry.cols} false ${key} rejects`,async()=>{
      const bad=clone();bad.entry.fixture.semantic_outcomes.models[key]=key==='draft_sha256'?'f'.repeat(64):key==='focused_id' && variant.startsWith('inline-')?{'inline-model':'model:glm-5.3','inline-budget':'budget:272000','inline-tier':'tier:Fast'}[variant]:'pi';
      bad.entry.fixture.semantic_before=structuredClone(bad.entry.fixture.semantic_outcomes);
      await assert.rejects(check(bad),/Models|draft|focus|identity/i);
    });
    if(variant.startsWith('advanced-'))test(`${variant}/${entry.cols} relabeling disclosure rejects actual marker`,async()=>{
      const bad=clone();bad.entry.variant=bad.entry.fixture.variant=variant==='advanced-expanded'?'advanced-collapsed':'advanced-expanded';
      bad.entry.fixture.semantic_outcomes.models.advanced=!bad.entry.fixture.semantic_outcomes.models.advanced;
      bad.entry.fixture.semantic_before=structuredClone(bad.entry.fixture.semantic_outcomes);
      await assert.rejects(check(bad),/disclosure|marker/i);
    });
  }
}
