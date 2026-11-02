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
    if(variant.startsWith('inline-'))test(`${variant}/${entry.cols} matching false identity and focus still reject real row`,async()=>{
      const bad=clone(),o=bad.entry.fixture.semantic_outcomes.models,field=variant.slice(7);
      o[field]={model:'glm-5.3',budget:272000,tier:'Fast'}[field];o.focused_id=field+':'+o[field];
      bad.entry.fixture.semantic_before=structuredClone(bad.entry.fixture.semantic_outcomes);
      await assert.rejects(check(bad),/Models|focus|identity/i);
    });
    if(variant.startsWith('inline-'))test(`${variant}/${entry.cols} synchronized real nonfocused production row rejects (R1)`,async()=>{
      const bad=clone(),o=bad.entry.fixture.semantic_outcomes.models,field=variant.slice(7);
      const alternative={model:'glm-5.3',budget:272000,tier:'Fast'}[field];
      const row=d.frame.text.find(line=>line.includes('( ) '+{model:'Model ',budget:'Budget ',tier:'Default tier '}[field]+alternative));
      assert.ok(row,'real nonfocused production row exists');
      o[field]=alternative;o.focused_id=field+':'+alternative;o.focused_row=row;
      bad.entry.fixture.semantic_before=structuredClone(bad.entry.fixture.semantic_outcomes);
      const view=structuredClone(bad.entry.fixture.expected_view);
      await assert.rejects(check(bad),/Models.*focus|focused.*row|focus.*identity/i);
      assert.deepEqual(bad.entry.fixture.expected_view,view);assert.equal(bad.frame,d.frame);assert.equal(bad.ansi,d.ansi);assert.equal(bad.png,d.png);
    });
    if(variant.startsWith('inline-'))test(`${variant}/${entry.cols} supplied focus marker cannot be removed`,async()=>{
      const bad=clone();bad.entry.fixture.semantic_outcomes.models.focused_row=bad.entry.fixture.semantic_outcomes.models.focused_row.replace('> ','  ');
      bad.entry.fixture.semantic_before=structuredClone(bad.entry.fixture.semantic_outcomes);
      await assert.rejects(check(bad),/Models.*focus|marker/i);
    });
    if(variant.startsWith('inline-'))test(`${variant}/${entry.cols} synchronized forged focus marker on an ordinary row rejects`,async()=>{
      const bad=clone(),o=bad.entry.fixture.semantic_outcomes.models,field=variant.slice(7),alternative={model:'glm-5.3',budget:272000,tier:'Fast'}[field];
      const ordinary=d.frame.text.find(line=>line.includes('( ) '+{model:'Model ',budget:'Budget ',tier:'Default tier '}[field]+alternative));
      assert.ok(ordinary);
      const forged=ordinary.trim().replace(/^\( /,'> ( ');
      o[field]=alternative;o.focused_id=field+':'+alternative;o.focused_row=forged;
      o.body_rows=o.body_rows.map(row=>row.trim().startsWith('> ')?row.replace('> ','  '):row.trim()===ordinary.trim()?forged:row);
      bad.entry.fixture.semantic_before=structuredClone(bad.entry.fixture.semantic_outcomes);
      await assert.rejects(check(bad),/Models.*focus|marker/i);
    });
    if(variant.startsWith('inline-') && paths['advanced-collapsed'])test(`${variant}/${entry.cols} genuine collapsed-form bytes reject borrowed inline observations`,async()=>{
      const path=paths['advanced-collapsed'],manifest=JSON.parse(await readFile(path,'utf8'));
      const baseline=structuredClone(manifest.captures.find(e=>e.cols===entry.cols));assert.ok(baseline);
      const file=kind=>readFile(join(dirname(path),baseline.files[kind].path));
      const view=structuredClone(baseline.fixture.expected_view);
      baseline.variant=baseline.fixture.variant=variant;
      baseline.fixture.semantic_outcomes=structuredClone(entry.fixture.semantic_outcomes);
      baseline.fixture.semantic_before=structuredClone(entry.fixture.semantic_before);
      const native={entry:baseline,frame:JSON.parse(await file('frame')),png:await file('png'),ansi:await file('ansi')};
      await assert.rejects(check(native),/Models|focus|production|frame/i);
      assert.deepEqual(baseline.fixture.expected_view,view);
    });
    if(variant.startsWith('advanced-'))test(`${variant}/${entry.cols} relabeling disclosure rejects actual marker`,async()=>{
      const bad=clone();bad.entry.variant=bad.entry.fixture.variant=variant==='advanced-expanded'?'advanced-collapsed':'advanced-expanded';
      bad.entry.fixture.semantic_outcomes.models.advanced=!bad.entry.fixture.semantic_outcomes.models.advanced;
      bad.entry.fixture.semantic_before=structuredClone(bad.entry.fixture.semantic_outcomes);
      await assert.rejects(check(bad),/disclosure|marker/i);
    });
  }
}
