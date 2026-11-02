// Shared capture/verification gates. Terminal parsing, never app-layout HTML.
import { inflateSync } from 'node:zlib';
import { createRequire } from 'node:module';
import { readFile } from 'node:fs/promises';
import { dirname, resolve, join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../..');
const require = createRequire(import.meta.url);
const { Terminal } = require(join(root, '.cache/tui-visual/tools/node_modules/@xterm/xterm/lib/xterm.js'));
const themes = { dark: '#111820', light: '#f5f6f8', mono: '#111111' };
const screens = Object.fromEntries(['Dashboard','Models','Review','Jobs','Gateway','Service','Repair','SSH','Utilities','Settings'].map(s => [s.toLowerCase(),s]));
const modals = {help:['Dashboard','help'],palette:['Dashboard','palette'],discard:['Models','discard'],quit:['Models','quit'],
  'ssh-confirm':['SSH','ssh'],'monitor-install':['Service','service:install'],'monitor-uninstall':['Service','service:uninstall'],more:['Dashboard','more']};

function scene(entry) {
  const f = entry.fixture;
  if (!f) throw new Error('Missing fixture scene');
  let expected;
  if (screens[entry.scenario]) expected = [screens[entry.scenario],'',''];
  else if (modals[entry.scenario]) expected = [...modals[entry.scenario],''];
  else if (entry.scenario.startsWith('edit-')) {
    const field = entry.scenario.slice(5).replace('compaction','compact');
    expected = [field === 'filter' ? 'Dashboard' : field === 'key' ? 'Gateway' : 'Models','',field];
  } else throw new Error('Unknown requested scene');
  if (JSON.stringify([f.screen,f.modal,f.editor]) !== JSON.stringify(expected)) throw new Error('Requested scene/target does not match fixture state');
  for (const key of ['scenario','state','variant','theme','cols','rows']) {
    if (key in f && f[key] !== entry[key]) throw new Error(`Requested scene ${key} does not match fixture`);
  }
}

// State predicates use actual serialized clients, never the echoed request.
function fixtureState(entry,frame) {
  const clients=entry.fixture.clients || [], c=clients[0];
  const fail=()=>{throw new Error(`Fixture client state ${entry.state} not reached`);};
  if(entry.state==='empty'){if(clients.length)fail();return;}
  if(!c)fail();
  switch(entry.state) {
    case 'baseline':
      if(!c.installed || !c.supported || c.status!=='CONFIGURED' || c.name!=='Codex')fail();break;
    case 'unicode':
      if(!c.installed || !c.supported || c.name!=='Codex 模型 — café · a deliberately long client name')fail();
      if(['dashboard','models'].includes(entry.scenario) && entry.variant==='baseline' && !frame.text.join('\n').includes('模型'))fail();break;
    case 'missing':if(c.installed || c.status!=='NOT INSTALLED')fail();break;
    case 'unsupported':if(c.supported || !c.reason)fail();break;
    case 'drift':if(c.status!=='DRIFT')fail();break;
    case 'parity': {
      const names=['Codex','Claude Code','OpenCode','Pi','Grok Build','Hermes'];
      if(clients.length!==6 || clients.some((c,i)=>c.name!==names[i] || c.supported!==(i===0) || c.installed!==(c.id!=='pi')))fail();break;
    }
    default:fail();
  }
}

function semanticOutcomes(entry,frame) {
  const legacy=entry.validation_schema<3 || !entry.validation_schema;
  const hasClients=Object.hasOwn(entry.fixture,'clients');
  // Old rows did not record clients. Absence is unavailable historical proof,
  // not an empty snapshot. Recorded observations still have to satisfy state.
  if(hasClients)fixtureState(entry,frame);
  else if(!legacy)throw new Error('Missing required fixture client observations');
  const o=entry.fixture.semantic_outcomes;
  if(legacy)return 'legacy; distinguishing outcomes not recorded'+(hasClients?'':'; client observations unavailable');
  if(!o || !entry.fixture.semantic_before)throw new Error('Missing semantic outcome observations');
  if(JSON.stringify(o)!==JSON.stringify(entry.fixture.semantic_before))throw new Error('Semantic outcome changed after actual PTY key delivery');
  const text=frame.text.join('\n');
  if(entry.scenario==='help' || entry.scenario==='review') {
    const match=/Page\s*(\d+)\s*of\s*(\d+)/.exec(text),page=o.page;
    if(!page || !Number.isInteger(page.number) || !Number.isInteger(page.total) || page.number<1 || page.number>page.total
      || !match || +match[1]!==page.number || +match[2]!==page.total)throw new Error('Semantic page mismatch with production View/frame marker');
    if(['last-page','acknowledged'].includes(entry.variant) && page.number!==page.total)throw new Error('Last page outcome not reached');
    if(entry.scenario==='review' && entry.variant==='acknowledged' && (o.review_acknowledged!==true || !text.includes('Complete review acknowledged')))throw new Error('Review acknowledgment outcome missing');
  }
  switch(entry.variant) {
    case 'baseline':break;
    case 'last-page':if(!['help','review'].includes(entry.scenario))throw new Error('Invalid page variant');break;
    case 'acknowledged':if(entry.scenario!=='review')throw new Error('Invalid acknowledgment variant');break;
    case 'no-results': {
      if(entry.scenario==='dashboard') {
        const f=o.filter, clients=entry.fixture.clients || [];
        if(!f?.query || f.matching_clients!==0 || clients.some(c=>(c.id+' '+c.name).toLowerCase().includes(f.query.toLowerCase()))
          || !f.clear_control_label || !text.includes(f.clear_control_label))throw new Error('Filter no-results semantic outcome missing');
      } else if(entry.scenario==='palette') {
        const p=o.palette;
        if(!p?.query || p.matching_commands!==0 || !text.includes(p.query) || !text.includes('No commands match.'))throw new Error('Palette no-results semantic outcome missing');
      } else throw new Error('Invalid no-results variant');
      break;
    }
    case 'invalid': {
      const e=o.editor,v=o.validation,fields={'edit-budget':'budget','edit-compaction':'compact','edit-endpoint':'endpoint'};
      if(!fields[entry.scenario] || e?.field!==fields[entry.scenario] || e.value!=='invalid-input' || v?.value!==e.value || v.field!==e.field
        || entry.fixture.editor!==e.field || v.editor_retained!==true || !text.includes('invalid-input'))throw new Error('Invalid editor value/retention outcome missing');
      if(!v.status || o.status!==v.status || !frame.text.some(line=>line===v.status.slice(0,entry.cols)))throw new Error('Validation refusal status does not match production View');
      if(!/^[0-9a-f]{64}$/.test(v.before_draft_sha256) || v.before_draft_sha256!==v.after_draft_sha256)throw new Error('Validation refusal changed draft');
      break;
    }
    default:throw new Error('Unsupported semantic variant on this baseline');
  }
  return 'observed-model-and-production-view';
}

export async function terminalFrame(bytes, cols, rows) {
  const t = new Terminal({cols,rows,allowProposedApi:true,scrollback:0});
  try {
    await new Promise(resolve => t.write(bytes,resolve));
    const text=[], cells=[];
    for(let y=0;y<rows;y++) {
      const line=t.buffer.active.getLine(y);text.push(line.translateToString(true));const row=[];
      for(let x=0;x<cols;x++) {
        const c=line.getCell(x);
        row.push({text:c.getChars(),width:c.getWidth(),fg:c.getFgColor(),bg:c.getBgColor(),fg_mode:c.getFgColorMode(),bg_mode:c.getBgColorMode(),
          bold:!!c.isBold(),italic:!!c.isItalic(),underline:!!c.isUnderline(),inverse:!!c.isInverse(),dim:!!c.isDim()});
      }
      cells.push(row);
    }
    return {text,cells,buffer:t.buffer.active.type};
  } finally { t.dispose(); }
}

function decodePNG(png, width, height) {
  if (!png.subarray(0,8).equals(Buffer.from('89504e470d0a1a0a','hex'))) throw new Error('Not a PNG');
  const idat=[];let channels, actualWidth,actualHeight;
  for(let p=8;p+12<=png.length;) {
    const n=png.readUInt32BE(p), kind=png.toString('ascii',p+4,p+8), end=p+12+n;
    if(end>png.length)throw new Error('Truncated PNG');
    const data=png.subarray(p+8,p+8+n);
    if(kind==='IHDR') {
      actualWidth=data.readUInt32BE(0);actualHeight=data.readUInt32BE(4);
      channels=data[9]===2?3:data[9]===6?4:0;
      if(data[8]!==8 || !channels || data[10] || data[11] || data[12])throw new Error('Unsupported PNG encoding');
    }
    if(kind==='tRNS')throw new Error('Native PNG must be opaque; transparency chunk refused');
    if(kind==='IDAT')idat.push(data);
    p=end;
    if(kind==='IEND')break;
  }
  if(actualWidth!==width || actualHeight!==height || width<1 || height<1 || width*height>10_000_000)throw new Error('PNG geometry mismatch');
  const stride=width*channels, raw=inflateSync(Buffer.concat(idat),{maxOutputLength:(stride+1)*height});
  if(raw.length!==(stride+1)*height)throw new Error('PNG decoded length mismatch');
  const pixels=Buffer.alloc(stride*height);
  const paeth=(a,b,c)=>{const p=a+b-c,pa=Math.abs(p-a),pb=Math.abs(p-b),pc=Math.abs(p-c);return pa<=pb&&pa<=pc?a:pb<=pc?b:c;};
  for(let y=0;y<height;y++) {
    const filter=raw[y*(stride+1)];if(filter>4)throw new Error('Unknown PNG filter');
    for(let x=0;x<stride;x++) {
      const offset=y*stride+x, left=x>=channels?pixels[offset-channels]:0,up=y?pixels[offset-stride]:0,diagonal=y&&x>=channels?pixels[offset-stride-channels]:0;
      const predict=[0,left,up,Math.floor((left+up)/2),paeth(left,up,diagonal)][filter];
      pixels[offset]=(raw[y*(stride+1)+1+x]+predict)&255;
    }
  }
  return {pixels,channels,width,height};
}

function glyphSanity(png, frame) {
  const m=frame.metrics, decoded=decodePNG(png,m.width_px,m.height_px), {pixels,channels,width}=decoded;
  if(channels===4)for(let i=3;i<pixels.length;i+=4) {
    if(pixels[i]!==255)throw new Error('Native PNG must be opaque; non-opaque alpha refused');
  }
  const cw=m.cell_width_px,ch=m.cell_height_px;
  let expected=0,painted=0;
  // Text-bearing cells must contain contrasting pixels, not merely a colored
  // panel background. This is a paint/export sanity check, not OCR or a grade.
  for(let y=0;y<frame.rows;y++)for(let x=0;x<frame.cols;x++) {
    const cell=frame.cells[y][x];if(!cell.width || !/\S/u.test(cell.text))continue;
    expected++;
    const x0=Math.floor(x*cw),y0=Math.floor(y*ch),x1=Math.min(width,Math.floor((x+cell.width)*cw)),y1=Math.min(decoded.height,Math.floor((y+1)*ch));
    const first=(y0*width+x0)*channels;let contrast=false;
    for(let py=y0;py<y1&&!contrast;py++)for(let px=x0;px<x1;px++) {
      const i=(py*width+px)*channels;
      if(pixels[i]!==pixels[first] || pixels[i+1]!==pixels[first+1] || pixels[i+2]!==pixels[first+2]){contrast=true;break;}
    }
    if(contrast)painted++;
  }
  if(!expected || painted/expected<.9)throw new Error(`Blank/missing glyph pixels: ${painted}/${expected} text cells painted`);
  return {expected_text_cells:expected,contrasting_text_cells:painted,minimum_fraction:.9};
}

export async function validateEvidence(entry, frame, png, ansi) {
  scene(entry);
  const m=frame.metrics;
  if(frame.cols!==entry.cols || frame.rows!==entry.rows || frame.cells.length!==entry.rows || frame.text.length!==entry.rows || frame.cells.some(r=>r.length!==entry.cols)
    || !m || m.width_px!==entry.cols*m.cell_width_px || m.height_px!==entry.rows*m.cell_height_px || m.device_pixel_ratio!==1)throw new Error('Terminal geometry mismatch');
  const fromCells=frame.cells.map(row=>row.filter(c=>c.width!==0).map(c=>c.text||' ').join('').replace(/ +$/,''));
  if(JSON.stringify(fromCells)!==JSON.stringify(frame.text.map(line=>line.replace(/ +$/,''))))throw new Error('cell/text disagreement');
  if(!frame.text.some(s=>s.includes('oneline-ccc')))throw new Error('Empty native cell frame');
  if(frame.theme.background.toLowerCase()!==themes[entry.theme])throw new Error('Requested scene theme does not match frame');
  const view=entry.fixture.expected_view;
  if(entry.validation_schema>=2 && !view)throw new Error('Missing production View reference');
  if(view) {
    if(!view.alt_screen)throw new Error('Expected production View must use alternate screen');
    const text=view.content.replace(/(?<!\r)\n/g,'\r\n');
    const expected=await terminalFrame(`\x1b[?1049h\x1b[H${text}`,entry.cols,entry.rows);
    // The production renderer can erase trailing padding instead of printing
    // space glyphs; compare semantic line text with that padding normalized.
    const lines=text=>text.map(line=>line.replace(/ +$/,''));
    if(JSON.stringify(lines(expected.text))!==JSON.stringify(lines(frame.text)))throw new Error('Captured frame does not match expected production View');
  }
  if(ansi) {
    const replay=await terminalFrame(new Uint8Array(ansi),entry.cols,entry.rows);
    if(replay.buffer!=='alternate' || JSON.stringify(replay.cells)!==JSON.stringify(frame.cells) || JSON.stringify(replay.text)!==JSON.stringify(frame.text))throw new Error('ANSI replay/cell frame disagreement');
  }
  const semantic_level=semanticOutcomes(entry,frame);
  return {semantic_level,level:view?'production-view-and-ansi':'legacy-ansi-only; original production View unavailable',...glyphSanity(png,frame)};
}

async function main() {
  const path=resolve(process.argv[2]), manifest=JSON.parse(await readFile(path,'utf8'));
  const counts={};
  for(const entry of manifest.captures) {
    if(entry.status!=='rendered')continue;
    const frame=JSON.parse(await readFile(join(dirname(path),entry.files.frame.path),'utf8'));
    const result=await validateEvidence(entry,frame,await readFile(join(dirname(path),entry.files.png.path)),await readFile(join(dirname(path),entry.files.ansi.path)));
    const level=`${result.level}; semantics=${result.semantic_level}`;
    counts[level]=(counts[level]||0)+1;
  }
  console.log(JSON.stringify(counts));
}
if(process.argv[1] && pathToFileURL(resolve(process.argv[1])).href===import.meta.url)main().catch(e=>{console.error(e.message);process.exitCode=1;});
