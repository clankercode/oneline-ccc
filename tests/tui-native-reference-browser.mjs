// Fixture-only browser verification. Never starts the native app or contacts an endpoint.
// Run: node tests/tui-native-reference-browser.mjs
// Focused review journeys: --fix1 [--case=F1] [--record=green]
// PLAYWRIGHT_MODULE accepts a module path relative to this file, an absolute path,
// a file URL, or a package name (CI uses ../.cache/browser/node_modules/playwright/index.mjs).
import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { readFile, mkdir, writeFile } from 'node:fs/promises';
import { resolve, extname } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('../', import.meta.url));
const fixRound = process.argv.includes('--fix1');
const record = (process.argv.find(arg=>arg.startsWith('--record='))?.slice(9) || 'green').replace(/[^a-zA-Z0-9-]/g,'');
const caseFilter = process.argv.find(arg=>arg.startsWith('--case='))?.slice(7);
const artifacts = resolve(root, '.cache/artifacts/native-references', ...(fixRound ? ['fix1',record] : []));
await mkdir(artifacts, {recursive:true});
// Playwright's temporary browser profiles and downloads also stay in this worktree.
process.env.TMPDIR = resolve(artifacts,'browser-scratch');
await mkdir(process.env.TMPDIR, {recursive:true});
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE || '/home/xertrov/.agents/skills/headless-browser-screenshots/node_modules/playwright/index.mjs');
const mime = {'.html':'text/html', '.css':'text/css', '.mjs':'text/javascript'};
const server = createServer(async (req,res) => {
  const name = new URL(req.url,'http://localhost').pathname.slice(1);
  // Serve only the companion and approved reference assets; no directory listing.
  if (!/^tui-(native-reference\.(html|css|mjs)|preview\.(html|css|js)|views\.mjs|state\.mjs|commands\.mjs|dialogs\.mjs)$/.test(name)) {res.writeHead(404).end();return;}
  try {res.writeHead(200, {'Content-Type':mime[extname(name)] || 'text/javascript'});res.end(await readFile(resolve(root,'site',name)));}
  catch {res.writeHead(404).end();}
});
await new Promise(resolve => server.listen(0,'127.0.0.1',resolve));
const origin = `http://127.0.0.1:${server.address().port}`;
let browser;
const errors = [], remote = [], captures = [], checks = [];
const screens = ['Gateway','Service','Repair','SSH','Utilities','Settings'];
const themes = ['dark','light','mono'];
const sizes = {wide:{width:1440,height:1000},compact:{width:1024,height:768},narrow:{width:390,height:844}};
try {
  browser = await chromium.launch({headless:true});
  const context = await browser.newContext({deviceScaleFactor:1,reducedMotion:'reduce'});
  await context.route('**/*', route => {
    if (new URL(route.request().url()).origin !== origin) {remote.push(route.request().url());return route.abort();}
    return route.continue();
  });
  const page = await context.newPage();
  page.on('pageerror', error => errors.push(error.message));
  page.on('console', msg => {if (msg.type() === 'error') errors.push(msg.text());});
  if (fixRound) {
    const outcomes = [], measurements = [];
    const visit = async query => {
      await page.setViewportSize({width:390,height:844});
      await page.goto(`${origin}/tui-native-reference.html?${new URLSearchParams({layout:'narrow',...query})}`);
      await page.evaluate(()=>document.fonts.ready);
    };
    const measureFocus = async () => page.evaluate(() => {
      const el=document.activeElement, rect=el.getBoundingClientRect();
      return {label:el.getAttribute('aria-label') || el.textContent.trim(),id:el.id,screen:el.dataset.screen,editor:el.dataset.editor,overlay:el.dataset.overlay,
        top:rect.top,bottom:rect.bottom,left:rect.left,right:rect.right,y:scrollY,width:innerWidth,height:innerHeight};
    });
    const visibleFocus = async () => {
      const focus=await measureFocus();measurements.push(focus);
      assert.ok(focus.top>=4 && focus.bottom<=focus.height-4 && focus.left>=0 && focus.right<=focus.width,`focus outside viewport: ${JSON.stringify(focus)}`);
      return focus;
    };
    const sameAnchor = (before,after) => {assert.ok(Math.abs(before.y-after.y)<2,'return restores visit scrollY');assert.ok(Math.abs(before.top-after.top)<2,'return restores invoking control position');};
    const journey = async (name,run) => {
      if (caseFilter && !name.startsWith(caseFilter)) return;
      const started=Date.now(), result={name,status:'PASS'};measurements.length=0;
      try {await run();} catch(error) {result.status='FAIL';result.error=error.message;process.exitCode=1;}
      result.elapsedMs=Date.now()-started;result.measurements=[...measurements];
      result.query=new URL(page.url()).search;result.viewport=page.viewportSize();
      result.viewportCapture=resolve(artifacts,`${name}-viewport.png`);
      result.consoleCapture=resolve(artifacts,`${name}-console.png`);
      await page.screenshot({path:result.viewportCapture,animations:'disabled'});
      await page.locator('#native-reference').screenshot({path:result.consoleCapture,animations:'disabled'});
      outcomes.push(result);console.log(`${result.status} ${name}${result.error ? ': '+result.error : ''}`);
    };
    await journey('F1-initial-screen-focus',async()=>{
      for (const screen of screens) {await visit({screen});await visibleFocus();}
    });
    await journey('F1-initial-layer-focus',async()=>{
      for (const overlay of ['quit','help','more','palette']) {await visit({screen:'Gateway',overlay});await visibleFocus();}
      await visit({screen:'Gateway',editor:'key'});await visibleFocus();
    });
    await journey('F1-key-editor-return-anchor',async()=>{
      await visit({screen:'Gateway'});
      const control=page.getByRole('button',{name:'Enter hidden API key',exact:true});
      await control.scrollIntoViewIfNeeded();await control.focus();const before=await visibleFocus();
      await page.keyboard.press('Enter');await visibleFocus();await page.keyboard.press('Escape');
      assert.equal(await control.evaluate(el=>el===document.activeElement),true);sameAnchor(before,await visibleFocus());
    });
    await journey('F1-footer-help-return-anchor',async()=>{
      await visit({screen:'Gateway'});const control=page.locator('.key-actions [data-overlay="help"]');
      await control.scrollIntoViewIfNeeded();await control.focus();const before=await visibleFocus();
      await page.keyboard.press('Enter');await visibleFocus();await page.keyboard.press('Escape');
      assert.equal(await control.evaluate(el=>el===document.activeElement),true);sameAnchor(before,await visibleFocus());
    });
    await journey('F1-screen-return-anchor',async()=>{
      await visit({screen:'Settings'});const control=page.locator('.setting[data-screen="Service"]');
      await control.scrollIntoViewIfNeeded();await control.focus();const before=await visibleFocus();
      await page.keyboard.press('Enter');await visibleFocus();await page.getByRole('button',{name:'Back to Settings',exact:true}).focus();await page.keyboard.press('Enter');
      assert.equal(await control.evaluate(el=>el===document.activeElement),true,'return to the invoking Settings row');sameAnchor(before,await visibleFocus());
      assert.equal(await control.evaluate(el=>el.matches(':focus-visible')),true,'keyboard return retains visible focus styling');
    });
    await journey('F1-browser-visit-return-anchor',async()=>{
      await visit({screen:'Settings'});const control=page.locator('.setting[data-screen="Service"]');
      await control.scrollIntoViewIfNeeded();await control.focus();const before=await visibleFocus();
      await page.keyboard.press('Enter');await page.goBack();
      assert.equal(await control.evaluate(el=>el===document.activeElement),true,'browser Back restores this visit');sameAnchor(before,await visibleFocus());
      await page.goForward();assert.equal(await page.locator('#native-reference').getAttribute('data-screen'),'Service');await visibleFocus();
    });
    await journey('F1-layer-return-keeps-appearance',async()=>{
      await visit({screen:'Gateway'});await page.locator('[data-editor="key"]').click();
      await page.locator('#reference-theme').selectOption('light');await page.locator('#reference-layout').selectOption('compact');await page.keyboard.press('Escape');
      assert.equal(await page.locator('html').getAttribute('data-theme'),'light','dismissal retains the chosen theme');
      assert.equal(await page.locator('#native-reference').getAttribute('data-layout'),'compact');await visibleFocus();
    });
    await journey('F2-unavailable-removal-reason',async()=>{
      for (const state of ['unavailable','pending','error']) {
        await visit({screen:'Service',state});
        const control=page.getByRole('button',{name:'Review monitor uninstall',exact:true});
        assert.equal(await control.getAttribute('aria-disabled'),'true',`${state}: removal unavailable`);
        assert.ok(await control.getAttribute('aria-describedby'),`${state}: reason is accessible`);
        await control.focus();await control.press('Enter');
        assert.equal(await page.locator('#native-reference').getAttribute('data-overlay'),'');
        assert.ok((await page.locator('#reference-announcement').innerText()).length>20);
        await page.locator('#reference-nav [data-overlay="more"]').click();await page.locator('.destination-list [data-screen="Service"]').click();
        assert.equal(await page.locator('#native-reference').getAttribute('data-state'),state,'reopening the current destination preserves its represented service state');
        assert.equal(await page.getByRole('button',{name:'Review monitor uninstall',exact:true}).getAttribute('aria-disabled'),'true');
      }
    });
    await journey('F2-direct-service-confirmation-ownership',async()=>{
      for (const state of ['unavailable','pending','error','absent','running','stopped']) {
        await visit({screen:'Service',state});
        const backend=await page.locator('#reference-body dt').filter({hasText:/^Backend$/}).evaluate(el=>el.nextElementSibling.textContent);
        assert.equal(backend,state==='unavailable' ? 'Unavailable' : 'systemd user');
        for (const operation of ['install','uninstall']) {
          await visit({screen:'Service',state,overlay:`service:${operation}`});
          const facts=await page.locator('#reference-body dl').evaluate(dl=>Object.fromEntries([...dl.querySelectorAll('dt')].map(el=>[el.textContent,el.nextElementSibling.textContent])));
          assert.equal(facts.Backend,backend,`${state}: confirmation retains actual backend`);
          assert.ok(facts.Ownership,`${state}: ownership is explicit`);
          const enabled=operation==='install' ? state==='absent' : ['running','stopped'].includes(state);
          assert.equal(await page.locator('[data-action="confirm"]').getAttribute('aria-disabled')==='true',!enabled,`${state}: ${operation} authority`);
          if (operation==='uninstall' && enabled) assert.equal(facts.Ownership,'Owned by this account');
          if (['unavailable','pending','error'].includes(state)) assert.doesNotMatch(facts.Ownership,/^Owned/);
          await visibleFocus();
          if (!enabled) {await page.locator('[data-action="confirm"]').focus();await page.keyboard.press('Enter');assert.doesNotMatch(await page.locator('#reference-announcement').innerText(),/confirmation selected/);}
        }
      }
    });
    await journey('F3-named-utility-availability',async()=>{
      const research=await readFile(resolve(root,'docs/research/tui-stack.md'),'utf8');
      for (const identity of ['herdrdev/herdr','tmux/tmux','pingdotgg/t3code','getpaseo/paseo']) assert.ok(research.includes(identity),'utility identity has an existing research source');
      await visit({screen:'Utilities'});
      const body=await page.locator('#reference-body').innerText();
      for (const name of ['herdr','tmux','T3 Code','Paseo','Orca']) assert.ok(body.includes(name),`useful named utility: ${name}`);
      const rows=await page.locator('.utility-table tbody tr').allTextContents();
      assert.match(rows.find(text=>text.includes('Orca')),/Project not identified; unavailable/);
      for (const name of ['herdr','T3 Code','Paseo']) assert.match(rows.find(text=>text.includes(name)),/unverified/i,`${name}: intended match remains unverified`);
      assert.doesNotMatch(body,/Before an installer can be offered|Exact project and trusted source|Destination and privileges|No reviewed utility installer/);
      assert.match(body,/does not install, update or remove utilities/);
      assert.match(body,/own instructions/);
      assert.equal(await page.locator('#reference-body a[href^="http"]').count(),0,'no invented verified external links');
      const disabled=page.locator('#reference-body button[aria-disabled="true"]');
      await disabled.focus();await page.keyboard.press('Enter');assert.match(await page.locator('#reference-announcement').innerText(),/does not install, update or remove utilities/);
    });
    const openEditor = async query => {await page.locator('.key-actions [data-overlay="palette"]').click();await page.locator('#palette-search').fill(query);await page.keyboard.press('Enter');};
    await journey('F4-budget-compact-draft-consistency',async()=>{
      await visit({screen:'Gateway',editor:'budget'});
      await page.locator('#editor-value').fill('1000');await page.keyboard.press('Enter');
      await openEditor('compact');assert.equal(await page.locator('#editor-value').inputValue(),'900','native 1000-token example derives 900 compact tokens');
      await page.keyboard.press('Escape');await openEditor('context budget');
      await page.locator('#editor-value').fill('19');await page.keyboard.press('Enter');
      await page.locator('#reference-layer').selectOption('overlay:discard');assert.match(await page.locator('[data-confirmation="discard"]').innerText(),/19 → previous value/);
      await page.keyboard.press('Escape');await openEditor('compact');assert.equal(await page.locator('#editor-value').inputValue(),'17','integer truncation matches native 19-token example');
      await page.locator('#editor-value').fill('15');await page.keyboard.press('Enter');
      await openEditor('compact');assert.equal(await page.locator('#editor-value').inputValue(),'15');
      await page.locator('#editor-value').fill('0');await page.keyboard.press('Enter');assert.equal(await page.locator('#editor-value').getAttribute('aria-invalid'),'true');
      await page.keyboard.press('Escape');await openEditor('compact');assert.equal(await page.locator('#editor-value').inputValue(),'15','invalid compact edit preserves retained draft');
      await page.keyboard.press('Escape');await openEditor('context budget');await page.locator('#editor-value').fill('2000');await page.keyboard.press('Escape');
      await openEditor('compact');assert.equal(await page.locator('#editor-value').inputValue(),'15','cancelled budget edit does not recalculate compact');
    });
    await journey('F4-credential-reference-draft-consistency',async()=>{
      await visit({screen:'Gateway',editor:'credential'});
      await page.locator('#editor-value').fill('OTHER_DUMMY_KEY');await page.keyboard.press('Enter');
      assert.equal(await page.locator('#reference-body dt').filter({hasText:/^Env reference$/}).evaluate(el=>el.nextElementSibling.textContent),'OTHER_DUMMY_KEY');
      await page.getByRole('button',{name:'Edit environment reference',exact:true}).click();assert.equal(await page.locator('#editor-value').inputValue(),'OTHER_DUMMY_KEY');
      await page.locator('#editor-value').fill('CANCELLED_DUMMY_KEY');await page.keyboard.press('Escape');
      assert.equal(await page.locator('#reference-body dt').filter({hasText:/^Env reference$/}).evaluate(el=>el.nextElementSibling.textContent),'OTHER_DUMMY_KEY');
    });
    await journey('F5-wrapped-palette-row-geometry',async()=>{
      await visit({screen:'Gateway',overlay:'palette'});await page.setViewportSize({width:320,height:844});await page.evaluate(()=>document.fonts.ready);
      await page.locator('#palette-search').fill('credential');
      const assertRows = async () => {
        const rows=await page.locator('.palette-option').evaluateAll(elements=>elements.map(el=>{
          const box=el.getBoundingClientRect(),hint=el.querySelector('small').getBoundingClientRect();
          return {text:el.textContent,height:el.clientHeight,scrollHeight:el.scrollHeight,top:box.top,bottom:box.bottom,hintBottom:hint.bottom};
        }));
        measurements.push({paletteRows:rows});
        rows.forEach((row,index)=>{assert.ok(row.scrollHeight<=row.height+1,`${row.text}: row must contain wrapped content`);assert.ok(row.hintBottom<=row.bottom-4,`${row.text}: hint stays within hit area`);if(index)assert.ok(row.top>=rows[index-1].bottom,'rows do not overlap');});
      };
      await assertRows();await page.locator('#palette-search').fill('');await assertRows();
      for(let i=0;i<12;i++)await page.keyboard.press('ArrowDown');
      const selected=await page.locator('.palette-option[aria-selected="true"]').evaluate(el=>{
        const box=el.getBoundingClientRect(),viewport=el.parentElement.getBoundingClientRect();return {top:box.top,bottom:box.bottom,start:viewport.top,end:viewport.bottom};
      });
      assert.ok(selected.top>=selected.start-1 && selected.bottom<=selected.end+1,'selected result scrolls fully into the results window');
      await visibleFocus();assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
    });
    await journey('F0-fixture-boundary',async()=>{
      const source=await readFile(resolve(root,'site/tui-native-reference.mjs'),'utf8');
      assert.doesNotMatch(source,/\b(fetch|WebSocket)\s*\(|\b(localStorage|sessionStorage)\s*\./,'no endpoint access or persistent credential/draft storage');
      assert.match(await readFile(resolve(root,'site/tui-native-reference.html'),'utf8'),/connect-src 'none'/);
      await visit({screen:'Gateway',editor:'credential'});
      const initial=await page.locator('#editor-value').inputValue();
      const reference=()=>page.locator('#reference-body dt').filter({hasText:/^Env reference$/}).evaluate(el=>el.nextElementSibling.textContent);
      await page.locator('#editor-value').fill('PAGE_ONLY_DUMMY_REF');await page.keyboard.press('Enter');
      assert.equal(await reference(),'PAGE_ONLY_DUMMY_REF','the reference demonstrates a page-memory edit');
      assert.deepEqual(await page.evaluate(()=>({local:Object.keys(localStorage),session:Object.keys(sessionStorage)})),{local:[],session:[]},'the edit does not populate browser storage');
      await page.reload();await page.evaluate(()=>document.fonts.ready);
      assert.equal(await reference(),initial,'reload returns to the fixture instead of retaining the credential reference');
      await visibleFocus();
    });
    assert.ok(outcomes.length,'the requested case filter must select a consumer journey');
    assert.deepEqual(remote,[]);assert.deepEqual(errors,[]);
    const summary={mode:'fix1',record,passed:outcomes.filter(x=>x.status==='PASS').length,failed:outcomes.filter(x=>x.status==='FAIL').length,pageErrors:errors,externalRequests:remote,outcomes};
    await writeFile(resolve(artifacts,'results.json'),JSON.stringify(summary,null,2)+'\n');
    console.log(JSON.stringify({...summary,outcomes:undefined},null,2));
  } else {
  async function visit(query, capture = true) {
    const layout = query.layout || 'wide';
    await page.setViewportSize(sizes[layout]);
    const url = `${origin}/tui-native-reference.html?${new URLSearchParams(query)}`;
    await page.goto(url);await page.evaluate(()=>document.fonts.ready);
    await page.locator('#reference-title').waitFor();
    const name = [query.screen,query.theme || 'dark',layout,query.state,query.overlay && `overlay-${query.overlay}`,query.editor && `editor-${query.editor}`,query.variant,query.focus && `focus-${query.focus}`].filter(Boolean).join('-').replaceAll(':','-');
    const geometry = await page.evaluate(() => {
      const root=document.querySelector('#native-reference');
      return {pageOverflow:document.documentElement.scrollWidth>innerWidth,consoleOverflow:root.scrollWidth>root.clientWidth,
        tiny:[...root.querySelectorAll('h2,h3,p,small,button,input,dt,dd,td,kbd')].filter(el=>parseFloat(getComputedStyle(el).fontSize)<13).length,
        collapsed:[...root.querySelectorAll('caption,p,dd,small')].filter(el=>el.getClientRects().length && el.textContent.trim().length>12 && el.getBoundingClientRect().width<80).map(el=>el.textContent.trim()),
        focus:root.contains(document.activeElement)};
    });
    assert.equal(geometry.pageOverflow,false,`${name}: page overflow`);
    assert.equal(geometry.consoleOverflow,false,`${name}: console overflow`);
    assert.equal(geometry.tiny,0,`${name}: smaller-than-cell text`);
    assert.deepEqual(geometry.collapsed,[],`${name}: collapsed text column`);
    assert.equal(geometry.focus,true,`${name}: initial keyboard focus`);
    assert.equal(await page.locator('#reference-nav [aria-current="page"]').count(),1);
    if (capture) {
      const path=resolve(artifacts,`${name}.png`);
      await page.locator('#native-reference').screenshot({path,animations:'disabled'});
      captures.push({name,path,query,url:new URL(page.url()).search,title:await page.title(),viewport:sizes[layout]});
    }
    checks.push(name);
  }
  for (const screen of screens) for (const theme of themes) for (const layout of ['wide','narrow']) await visit({screen,theme,layout});
  for (const screen of screens) await visit({screen,theme:'dark',layout:'compact'});
  for (const overlay of ['quit','discard','ssh','service:install','service:uninstall','help','palette','more'])
    for (const theme of themes) for (const layout of ['wide','narrow']) {
      await visit({screen:overlay.startsWith('service:')?'Service':overlay==='ssh'?'SSH':overlay==='more'?'Settings':'Gateway',theme,layout,overlay});
      if (!['help','palette','more'].includes(overlay)) assert.equal(await page.locator('[data-close][data-initial-focus]').evaluate(el=>el===document.activeElement),true);
    }
  for (const editor of ['filter','model','budget','compact','endpoint','credential','key'])
    for (const theme of themes) for (const layout of ['wide','narrow']) await visit({screen:'Gateway',theme,layout,editor});
  const additional = {
    Gateway:['missing','stored','error'],Service:['running','stopped','unavailable','pending','error'],
    Repair:['empty','preparing','error','returned'],SSH:['unavailable','preparing','error','returned']
  };
  for (const [screen,states] of Object.entries(additional)) for (const state of states) await visit({screen,state,theme:'light',layout:'narrow'});
  for (const [editor,variant] of [['budget','invalid'],['compact','invalid'],['endpoint','invalid'],['model','long'],['key','vault'],['model','cursor-middle'],['filter','empty']]) await visit({screen:'Gateway',theme:'mono',layout:'narrow',editor,variant});
  await visit({screen:'Gateway',theme:'dark',layout:'narrow',overlay:'quit',variant:'dirty-busy'});
  for (const theme of themes) await visit({screen:'Utilities',theme,layout:'narrow',focus:'0'});

  // Exercise consumer paths rather than assertions that merely repeat markup.
  await visit({screen:'Gateway'},false);
  await page.getByRole('button',{name:'Edit local endpoint',exact:true}).click();
  await page.locator('#editor-value').fill('gateway-a.invalid');await page.locator('#editor-value').press('Enter');
  assert.match(await page.locator('#editor-error').innerText(),/absolute HTTP/);
  await page.locator('#editor-value').fill('https://gateway-a.invalid/v2');await page.locator('#editor-value').press('Enter');
  assert.equal(await page.locator('#native-reference').getAttribute('data-editor'),'');
  await page.getByRole('button',{name:'Edit local endpoint',exact:true}).click();
  assert.equal(await page.locator('#editor-value').inputValue(),'https://gateway-a.invalid/v2');
  await page.locator('#editor-value').fill('https://gateway-a.invalid/unsaved');await page.keyboard.press('Escape');
  await page.getByRole('button',{name:'Edit local endpoint',exact:true}).click();
  assert.equal(await page.locator('#editor-value').inputValue(),'https://gateway-a.invalid/v2');await page.keyboard.press('Escape');
  assert.equal(await page.getByRole('button',{name:'Edit local endpoint',exact:true}).evaluate(el=>el===document.activeElement),true);
  await page.keyboard.press('Control+p');await page.locator('#palette-search').fill('repair');await page.keyboard.press('Enter');
  assert.equal(await page.locator('#native-reference').getAttribute('data-screen'),'Repair');
  await page.keyboard.press('Control+p');await page.locator('#palette-search').fill('zzzz-no-match');
  assert.match(await page.locator('#palette-count').innerText(),/No matching/);await page.keyboard.press('Escape');
  await page.keyboard.press('?');assert.equal(await page.locator('#native-reference').getAttribute('data-overlay'),'help');await page.keyboard.press('Escape');
  await page.locator('#reference-screen').selectOption('Settings');await page.getByRole('switch',{name:/ASCII/}).click();
  assert.equal(await page.getByRole('switch',{name:/ASCII/}).getAttribute('aria-checked'),'true');
  await page.getByRole('switch',{name:/No color/}).focus();await page.keyboard.press('?');await page.keyboard.press('Escape');
  assert.equal(await page.getByRole('switch',{name:/No color/}).evaluate(el=>el===document.activeElement),true);
  await page.getByRole('button',{name:/Background monitor Optional/}).click();
  await page.getByRole('button',{name:'Review monitor install',exact:true}).click();
  await page.keyboard.press('ArrowRight');assert.match(await page.locator(':focus').innerText(),/Install monitor/);
  await page.keyboard.press('Enter');assert.match(await page.locator('#reference-announcement').innerText(),/No native action ran/);await page.keyboard.press('Escape');
  await page.locator('#reference-screen').selectOption('Utilities');
  await page.getByRole('button',{name:'Manage utilities',exact:true}).focus();
  await page.keyboard.press('Enter');
  assert.match(await page.locator('#reference-announcement').innerText(),/does not install, update or remove utilities/);
  const stable=page.url();await page.reload();assert.equal(page.url(),stable);
  await page.locator('#reference-screen').selectOption('SSH');await page.goBack();
  assert.equal(await page.locator('#native-reference').getAttribute('data-screen'),'Utilities');
  await visit({screen:'Gateway',editor:'key'},false);
  await page.locator('#key-input').press('Enter');assert.equal(await page.getByRole('button',{name:'Save key',exact:true}).evaluate(el=>el===document.activeElement),true);
  assert.equal(await page.locator('#key-input').inputValue(),'[hidden]');
  await page.keyboard.press('Escape');
  await page.getByRole('button',{name:'More',exact:true}).click();
  assert.equal(await page.locator('.destination-list button').count(),6);
  await page.keyboard.press('ArrowDown');await page.keyboard.press('Enter');
  assert.equal(await page.locator('#native-reference').getAttribute('data-screen'),'Service');
  await page.getByRole('button',{name:'More',exact:true}).click();await page.keyboard.press('Escape');
  assert.equal(await page.getByRole('button',{name:'More',exact:true}).evaluate(el=>el===document.activeElement),true);
  assert.equal(await page.evaluate(()=>localStorage.length+sessionStorage.length),0);
  assert.deepEqual(remote,[],'no external requests');assert.deepEqual(errors,[],'no console or page errors');
  const summary={captures:captures.length,renderChecks:checks.length,interactionJourneys:11,pageErrors:errors,externalRequests:remote,captureRoot:artifacts,checks,artifacts:captures};
  await writeFile(resolve(artifacts,'verification.json'),JSON.stringify(summary,null,2)+'\n');
  console.log(JSON.stringify({...summary,checks:undefined,artifacts:undefined},null,2));
  }
} finally {await browser?.close();await new Promise(resolve=>server.close(resolve));}
