// Companion to the approved four-screen preview. All values below are fixed fixtures.
// No fetch, native bridge, storage API, credential input, or background timers.
const $ = selector => document.querySelector(selector);
const e = value => String(value).replace(/[&<>"']/g, c => ({'&':'&amp;', '<':'&lt;', '>':'&gt;', '"':'&quot;', "'":'&#39;'})[c]);
const screens = ['Gateway', 'Service', 'Repair', 'SSH', 'Utilities', 'Settings'];
const states = {
  Gateway: ['session', 'missing', 'stored', 'error'],
  Service: ['absent', 'running', 'stopped', 'unavailable', 'pending', 'error'],
  Repair: ['ready', 'empty', 'preparing', 'error', 'returned'],
  SSH: ['ready', 'unavailable', 'preparing', 'error', 'returned'],
  Utilities: ['unavailable'], Settings: ['default']
};
const overlays = ['quit', 'discard', 'ssh', 'service:install', 'service:uninstall', 'help', 'palette', 'more'];
const overlayTitles = {quit:'Leave console', discard:'Discard Codex draft', ssh:'SSH handoff', 'service:install':'Install monitor', 'service:uninstall':'Uninstall monitor', help:'Keyboard help', palette:'Commands', more:'More destinations'};
const editors = {
  filter: { title: 'Filter clients', label: 'Client name', value: 'Codex', hint: 'Enter applies the filter. Esc keeps the previous filter.', note: 'Filtering keeps the current screen and selected client.', action: 'Apply filter' },
  model: { title: 'Exact model ID', label: 'Model ID', value: 'gpt-6.1-sol', hint: 'Preserve the exact ID. It will be validated when you review the plan.', note: 'A model name alone does not verify protocol support, capacity or tiers.', action: 'Keep in draft' },
  budget: { title: 'Custom context budget', label: 'Budget in tokens', value: '353000', hint: 'Enter a positive whole number. Saving recalculates the compact budget.', note: 'This is a chosen budget, not verified model capacity.', action: 'Keep in draft' },
  compact: { title: 'Compact token budget', label: 'Compact tokens', value: '272000', hint: 'Enter a positive whole number. Keep this choice separate from context capacity.', note: 'The draft is validated before any client configuration changes.', action: 'Keep in draft' },
  endpoint: { title: 'Local gateway endpoint', label: 'Endpoint URL', value: 'https://gateway-a.invalid/v1', hint: 'Use an absolute HTTP or HTTPS URL without a username, password, query or fragment.', note: 'Local-only value. Public views and repair briefs use the alias Gateway A.', action: 'Keep in draft' },
  credential: { title: 'Credential environment reference', label: 'Environment variable name', value: 'GATEWAY_A_API_KEY', hint: 'Enter the variable name, not its API key value. The profile is validated at review.', note: 'Editing this reference does not reveal or change a credential.', action: 'Keep in draft' },
  key: { title: 'Hidden API key', action: 'Save key' }
};
const choose = (value, values, fallback = values[0]) => values.includes(value) ? value : fallback;
let current, layerReturn = null, screenReturns = [], rendering = false, paletteIndex = 0;
const draftValues = {};
const display = { ascii: false, noColor: false, motion: false };
function fromURL() {
  const q = new URLSearchParams(location.search);
  const screen = choose(q.get('screen'), screens);
  return { screen, theme: choose(q.get('theme'), ['dark','light','mono']), layout: choose(q.get('layout'), ['wide','compact','narrow']),
    state: choose(q.get('state'), states[screen]), overlay: choose(q.get('overlay'), overlays, ''),
    editor: choose(q.get('editor'), Object.keys(editors), ''), variant: q.get('variant') || '', focus: q.get('focus') || '' };
}
const announce = text => { $('#reference-announcement').textContent = text; };
const action = (label, attrs = '', secondary = false) => `<button type="button" class="action${secondary ? ' secondary' : ''}" ${attrs}>${e(label)}</button>`;
const unavailable = (label, reason, id) => action(label, `aria-disabled="true" aria-describedby="${id}" data-reason="${e(reason)}"`, true);
const notice = (heading, text, tone = '') => `<div class="notice ${tone}"><strong>${e(heading)}</strong><p>${e(text)}</p></div>`;
const detail = rows => `<dl>${rows.map(([label, value, tone = '']) => `<dt>${e(label)}</dt><dd class="${tone}">${e(value)}</dd>`).join('')}</dl>`;
const title = (heading, subtitle, meta = '') => `<div class="screen-title"><div><h2 id="reference-title" tabindex="-1">${e(heading)}</h2><p>${e(subtitle)}</p></div>${meta ? `<span>${e(meta)}</span>` : ''}</div>`;
const eyebrow = text => `<p class="section-label">${e(text)}</p>`;
const back = () => action(`Back to ${screenReturns.at(-1)?.view.screen || 'Settings'}`, 'data-back', true);
const reason = (id, text) => `<p class="disabled-reason" id="${id}">${e(text)}</p>`;

function captureAnchor() {
  const el = document.activeElement;
  const selector = el.id ? `#${CSS.escape(el.id)}` : el.dataset.focusId !== undefined ? `#reference-body [data-focus-id="${el.dataset.focusId}"]` : el.dataset.overlay ? `#native-reference [data-overlay="${el.dataset.overlay}"]` : '';
  return {selector, y:scrollY, top:el.getBoundingClientRect().top};
}
function rememberVisit() {
  if (!rendering) history.replaceState({anchor:captureAnchor(),layerReturn,screenReturns},'',location.href);
}
function keepVisible(el) {
  const rect=el.getBoundingClientRect();
  if (rect.top<8 || rect.bottom>innerHeight-8 || rect.left<0 || rect.right>innerWidth) el.scrollIntoView({block:'nearest',inline:'nearest'});
}

function gateway() {
  const descriptions = {
    session: ['SESSION ONLY', 'Provided for this session', 'Session ends when you quit. A stored credential is a separate choice.'],
    missing: ['NEEDS KEY', 'No credential available', 'Enter a key for this session, or return without changing credentials.'],
    stored: ['STORED REFERENCE', 'Vault reference present; not unlocked', 'Unlocking the native credential store is required before client launch.'],
    error: ['UNAVAILABLE', 'Credential store could not be opened', 'Keep existing credentials unchanged. Use a session key or inspect the native vault.']
  };
  const [status, value, guidance] = descriptions[current.state];
  return title('Gateway & credentials', 'Keep the route, credential and verification state separate.', 'Selected: Gateway A') +
    `<div class="workarea"><div>${eyebrow('Connection / local value')}${detail([
      ['Alias', 'Gateway A'], ['Endpoint', draftValues.endpoint ?? 'https://gateway-a.invalid/v1'], ['Route', 'Not checked', 'warn'], ['Authentication', 'Not tested', 'warn'], ['Inference', 'Not tested', 'muted']
    ])}${notice('Unverified connection', 'An endpoint and a key do not establish model availability or a successful request.')}
    <div class="action-row">${action('Edit local endpoint', 'data-editor="endpoint"')}</div></div>
    <div class="detail">${eyebrow('Credential / value never shown')}<h3 class="${current.state === 'error' ? 'bad' : 'warn'}">${status}</h3>${detail([
      ['Key', current.state === 'missing' ? 'Not provided' : '[hidden]'], ['Availability', value], ['Default', 'Session only'], ['Env reference', draftValues.credential ?? editors.credential.value]
    ])}${notice('Choose where the key lives', guidance, 'info')}
    <div class="action-row">${action('Enter hidden API key', 'data-editor="key"')}${action('Edit environment reference', 'data-editor="credential"', true)}</div></div></div>
    <div class="action-row">${back()}</div>`;
}
function serviceFacts() {
  const state=current.screen === 'Service' ? current.state : 'absent';
  return {
    backend:state === 'unavailable' ? 'Unavailable' : 'systemd user',
    ownership:({absent:'No monitor registration',running:'Owned by this account',stopped:'Owned by this account',unavailable:'No supported backend',pending:'Not verified / request pending',error:'Not verified / reconcile first'})[state],
    canInstall:state === 'absent',canRemove:['running','stopped'].includes(state),
    reason:({absent:'No monitor registration is present.',unavailable:'No supported user-session backend is available. Use foreground inspection.',pending:'Wait for a final native observation before changing the monitor.',error:'Reconcile native status and ownership before changing the monitor.'})[state]
  };
}
function service() {
  const observations = {
    absent: ['NOT INSTALLED', 'No monitor registration', 'Foreground inspection remains available. The monitor is optional.'],
    running: ['RUNNING', 'Last observation: 12 seconds ago', 'Observation is read only. It does not validate gateway authentication.'],
    stopped: ['STOPPED', 'Registered; no current observation', 'Start is an explicit action. Client configuration remains separate.'],
    unavailable: ['UNAVAILABLE', 'No supported user-session backend', 'Use foreground inspection. Do not substitute a system-wide service.'],
    pending: ['REQUEST PENDING', 'Waiting for native status', 'The requested state has not been observed. Wait for a final outcome.'],
    error: ['OUTCOME UNKNOWN', 'Native status could not be reconciled', 'Inspect the native user service before retrying. No automatic retry.']
  };
  const [status, observed, guidance] = observations[current.state];
  const facts=serviceFacts();
  const blocked = ['absent','unavailable','pending','error'].includes(current.state);
  const why = current.state === 'absent' ? 'Install the optional monitor before using lifecycle controls.' : blocked ? guidance : 'Inspect native status before requesting a lifecycle change.';
  return title('Background monitor', 'Optional local observation. Setup works without it.', 'Current user only') +
    `<div class="status-band"><strong class="${current.state === 'running' ? 'good' : 'warn'}">${status}</strong><span>${observed}</span></div>
    <div class="workarea"><div>${eyebrow('Backend / ownership')}${detail([
      ['Backend', facts.backend], ['Ownership', facts.ownership], ['Scope', 'This account / user session'], ['Permission', 'No elevation'], ['Purpose', 'Observe local status only']
    ])}<div class="action-row">${action('Inspect status', 'data-action="inspect-service"')}${facts.canInstall ? action('Review monitor install', 'data-overlay="service:install"', true) : facts.canRemove ? action('Review monitor uninstall', 'data-overlay="service:uninstall"', true) : unavailable('Review monitor uninstall',facts.reason,'service-removal-reason')}</div>${!facts.canInstall && !facts.canRemove ? reason('service-removal-reason',facts.reason) : ''}</div>
    <div class="detail">${eyebrow('Observation boundaries')}${detail([
      ['Client writes', 'Off'], ['Auto updates', 'Off'], ['Auto repair', 'Off'], ['Model calls', 'None']
    ])}${notice(status === 'OUTCOME UNKNOWN' ? 'Reconcile before retrying' : 'Optional by design', guidance, current.state === 'error' ? 'error' : 'info')}
    <div class="action-row">${['Start','Stop','Restart'].map(label => {
      const disabled = blocked || (label === 'Start' && current.state === 'running') || (label !== 'Start' && current.state === 'stopped');
      const disabledWhy = current.state === 'running' && label === 'Start' ? 'The monitor is already running. Use Stop or Restart.' : current.state === 'stopped' && label !== 'Start' ? 'The monitor is stopped. Use Start.' : why;
      const id = `service-${label.toLowerCase()}-reason`;
      return disabled ? unavailable(label, disabledWhy, id) + `<span class="sr-only" id="${id}">${e(disabledWhy)}</span>` : action(label, `data-action="service-request"`, true);
    }).join('')}</div>${reason('service-reason', why)}</div></div><div class="action-row">${back()}</div>`;
}
const repairBrief = `MANUAL REPAIR / CODEX\n\nObserved\n  Configuration schema: needs inspection\n  Gateway: Gateway A\n  Credential: omitted\n  Endpoint and local paths: omitted\n\nRequest\n  Explain the configuration mismatch.\n  Propose the smallest change and validation.\n  Do not request secrets or contact the gateway.\n\nBoundary\n  No agent execution or production writes.\n  Review any suggestion before ordinary Apply.`;
function repair() {
  const ready = ['ready','returned'].includes(current.state);
  const why = 'Prepare a redacted brief before exporting to terminal scrollback.';
  return title('Manual repair brief', 'Read the exact handoff before you share it.', 'Selected: Codex') +
    `<div class="workarea"><div>${eyebrow('Scope / operator led')}${detail([
      ['Target', 'Codex'], ['Mode', 'Manual only'], ['Evidence', 'Redacted diagnostics'], ['Execution', 'No agent is started'], ['Config writes', 'None']
    ])}${notice('Review before sharing', 'Endpoints, key values and local paths are omitted. Add only the non-secret context your helper needs.', 'info')}
    <div class="action-row">${action(ready ? 'Prepare again' : 'Prepare redacted brief', 'data-action="prepare-brief"')}${ready ? action('Export to terminal scrollback', 'data-action="export-brief"', true) : unavailable('Export brief', why, 'repair-reason')}</div>${!ready ? reason('repair-reason', why) : ''}</div>
    <div class="detail">${eyebrow('Exact brief / redacted')}${ready ? `<pre class="brief" aria-label="Redacted manual repair brief">${e(repairBrief)}</pre>` : `<div class="empty-state"><strong class="${current.state === 'error' ? 'bad' : 'warn'}">${current.state === 'preparing' ? 'Preparing redacted diagnostics' : current.state === 'error' ? 'Could not prepare the brief' : 'No brief prepared'}</strong><p class="muted">${current.state === 'preparing' ? 'Wait for preparation before export. No agent is running.' : current.state === 'error' ? 'Retry preparation. Keep private diagnostics local until they can be safely redacted.' : 'Prepare the manual brief to inspect its exact contents here.'}</p></div>`}
    ${current.state === 'returned' ? notice('Returned from terminal', 'Export outcome is unavailable. Inspect terminal scrollback before attempting another export.') : ''}</div></div><div class="action-row">${back()}</div>`;
}
function ssh() {
  const blocked = current.state === 'unavailable' || current.state === 'preparing';
  const why = current.state === 'preparing' ? 'Wait for command preparation. Terminal ownership has not changed.' : 'The native backend has no reviewed SSH helper for this platform.';
  const status = {ready:'EXPLICIT HANDOFF', unavailable:'HELPER UNAVAILABLE', preparing:'PREPARING COMMAND', error:'HANDOFF FAILED', returned:'TERMINAL RETURNED'}[current.state];
  return title('SSH helper handoff', 'Give a separate command the terminal, only when you choose.', 'No server enabled') +
    `<div class="status-band"><strong class="${current.state === 'error' ? 'bad' : 'warn'}">${status}</strong><span>SSH setup is a separate decision</span></div>
    <div class="workarea"><div>${eyebrow('Terminal ownership')}<ol class="steps"><li><div><strong>Review the handoff</strong><small>Confirm before this console yields the terminal.</small></div></li><li><div><strong>Use the native helper</strong><small>The helper owns its prompts and any administrator review.</small></div></li><li><div><strong>Return to this console</strong><small>Inspect the exit result before deciding what to do next.</small></div></li></ol></div>
    <div class="detail">${eyebrow('Scope / access')}${detail([
      ['Current action', 'Handoff only'], ['Server state', 'Not inspected'], ['Firewall', 'Not inspected'], ['Remote login', 'Not verified']
    ])}${notice(current.state === 'error' ? 'Terminal handoff failed' : 'Review remains with you', current.state === 'error' ? 'The helper did not take terminal ownership. Inspect availability and try again explicitly.' : current.state === 'returned' ? 'Terminal control is back. A helper exit alone does not verify SSH access.' : 'Opening the helper does not enable a server, alter a firewall or verify remote access.', current.state === 'error' ? 'error' : 'info')}
    <div class="action-row">${blocked ? unavailable('Hand off to SSH command', why, 'ssh-reason') : action('Review SSH handoff', 'data-overlay="ssh"')}</div>${blocked ? reason('ssh-reason', why) : ''}</div></div><div class="action-row">${back()}</div>`;
}
function utilities() {
  const why = 'This build does not install, update or remove utilities.';
  // Identity descriptions come from docs/research/tui-stack.md; no installed state is inferred.
  const tools = [
    ['herdr','Agent terminal runtime / background server','Candidate: herdrdev/herdr / intended match unverified'],
    ['tmux','Terminal multiplexer','tmux/tmux / project identified'],
    ['T3 Code','Agent control','Candidate: pingdotgg/t3code / intended match unverified'],
    ['Paseo','Agent development environment','Candidate: getpaseo/paseo / intended match unverified'],
    ['Orca','Project not identified; unavailable','']
  ];
  return title('Utilities', 'Recognize the project before choosing its setup instructions.', 'Read only') +
    `<div class="status-band" id="utility-reason"><strong class="warn">READ ONLY</strong><span>${why}</span></div>
    <div class="table-wrap"><table class="utility-table"><caption>Utility identities / installation state is not inspected</caption><thead><tr><th scope="col">Utility</th><th scope="col">About the project</th></tr></thead><tbody>${tools.map(([name,about,identity])=>`<tr><td>${e(name)}</td><td data-label="About">${e(about)}${identity ? `<small>${e(identity)}</small>` : ''}</td></tr>`).join('')}</tbody></table></div>
    ${notice('Use the project you already know', 'Follow its own instructions. If a name is unclear, confirm the exact project first.', 'info')}
    <div class="action-row">${unavailable('Manage utilities',why,'utility-reason')}${back()}</div>`;
}
function settings() {
  const setting = (label, value, attrs, hint = '') => `<button type="button" class="setting" ${attrs}><strong>${e(label)}</strong><span class="setting-value">${e(value)}</span>${hint ? `<small>${e(hint)}</small>` : ''}</button>`;
  return title('Settings & navigation', 'Local display choices, grouped by what they affect.', 'Native configuration: Codex only') +
    `<div class="workarea"><div class="setting-group">${eyebrow('Display / this session')}
    ${setting('Appearance', current.theme, 'data-action="cycle-theme"', 'Dark / light / monochrome')}
    ${setting('ASCII decoration', display.ascii ? '[x] On' : '[ ] Off', `role="switch" aria-checked="${display.ascii}" data-toggle="ascii"`)}
    ${setting('No color', display.noColor ? '[x] On' : '[ ] Off', `role="switch" aria-checked="${display.noColor}" data-toggle="noColor"`, 'Terminal output preference; monochrome is a separate palette.')}
    ${setting('Reduced motion', display.motion ? '[x] On' : '[ ] Off', `role="switch" aria-checked="${display.motion}" data-toggle="motion"`, 'Keep progress readable without animation.')}
    <p class="selected-note">Display preferences do not change client configuration.</p></div>
    <div class="detail"><div class="setting-group">${eyebrow('Credentials')}${setting('Enter hidden API key', 'Session default', 'data-editor="key"', 'Vault storage is a separate, explicit choice.')}${setting('Gateway & credentials', 'Open', 'data-screen="Gateway"')}</div>
    <div class="setting-group">${eyebrow('Tools & observation')}${setting('Background monitor', 'Optional', 'data-screen="Service"', 'Read-only observation in the user session.')}${setting('Utilities', 'Unavailable', 'data-screen="Utilities"')}${setting('Manual repair', 'Open', 'data-screen="Repair"')}${setting('SSH helper', 'Review handoff', 'data-screen="SSH"')}</div></div></div>`;
}
const views = { Gateway: gateway, Service: service, Repair: repair, SSH: ssh, Utilities: utilities, Settings: settings };

function confirmation() {
  const busy = current.variant === 'busy' || current.variant === 'dirty-busy';
  const facts=serviceFacts();
  const serviceOperation=current.overlay.startsWith('service:');
  const blocked=serviceOperation && !(current.overlay === 'service:install' ? facts.canInstall : facts.canRemove);
  const content = {
    quit: ['Leave this console?', busy ? 'A job is still running. Request cancellation and wait for its final outcome.' : 'Codex has an unapplied draft. Leaving drops this draft.', [['Draft',current.variant === 'busy' ? 'None' : 'Codex / not applied'], ['Running job',busy ? 'Configuration inspection' : 'None'], ['Client files','No draft applied']], busy ? 'Cancellation is a request' : 'Keep the draft until you decide', busy ? 'The console stays open until the job reaches a safe final outcome.' : 'Return to keep editing. Session-only keys are dropped when the console closes.', busy ? 'Request cancel; stay until outcome' : 'Leave without applying'],
    discard: ['Discard Codex draft?', 'Only the selected client’s draft is affected.', [['Selected client','Codex'], ['Context budget',`${draftValues.budget ?? editors.budget.value} → previous value`], ['Other drafts','Kept'], ['Prepared plan','Cleared']], 'Applied configuration stays', 'Discard returns this client to its previous choices. It does not edit installed client files.', 'Discard Codex draft'],
    ssh: ['Hand terminal control to SSH?', 'The backend helper will own the terminal until it exits.', [['Action','Explicit command handoff'], ['Server state','Not inspected'], ['Administrator','Helper requests separately']], 'Handoff is not enablement', 'Review any server, account or firewall change in the helper. Returning does not prove remote login works.', 'Hand off terminal'],
    'service:install': ['Install the background monitor?', 'Review the optional observer separately from client setup.', [['Backend',facts.backend], ['Ownership',facts.ownership], ['Scope','This account / at sign-in'], ['Permission','No elevation'], ['Capability','Read-only local observation']], 'Configuration and credentials are separate', 'No client updates, automatic repair or model calls. Native inspection must confirm the installation outcome.', 'Install monitor'],
    'service:uninstall': ['Uninstall the background monitor?', facts.canRemove ? 'Remove the owned user-session observer registration.' : 'Removal is unavailable until backend status and ownership allow it.', [['Backend',facts.backend], ['Ownership',facts.ownership], ['Scope','This account only'], ['Client files','Kept'], ['Credentials','Kept']], 'Keep unrelated state', 'A foreign or uncertain registration must be reconciled before removal. Native status must confirm the result.', 'Uninstall monitor']
  }[current.overlay];
  const [heading, subtitle, rows, noteHeading, noteText, label] = content;
  const disabledWhy=facts.reason || 'A monitor registration already exists. Inspect its native status before changing it.';
  return `<div class="layer-body" data-confirmation="${e(current.overlay)}">${eyebrow('Confirmation / explicit choice')}${title(heading, subtitle)}${detail(rows)}${notice(noteHeading, noteText)}<div class="action-row">${action('Back; keep working', 'data-close data-initial-focus', true)}${action(label, `data-action="confirm"${blocked ? ` aria-disabled="true" aria-describedby="confirmation-reason" data-reason="${e(disabledWhy)}"` : ''}`)}</div>${blocked ? reason('confirmation-reason',disabledWhy) : ''}</div>`;
}
function editor() {
  const field = current.editor, def = editors[field];
  if (field === 'key') return `<div class="layer-body" data-field="key">${eyebrow('Gateway A / credential entry')}${title(def.title, 'The value stays hidden. There is no reveal or copy action.')}
    <form class="editor-form" id="editor-form"><label for="key-input">API key</label><input class="hidden-field" id="key-input" type="text" value="[hidden]" readonly autocomplete="off" aria-describedby="key-hint" data-initial-focus>
    <p class="editor-hint" id="key-hint">Session only by default. Enter moves to Save; Back drops the unsaved key.</p>
    <label class="check-label"><input type="checkbox" id="persist-key" ${current.variant === 'vault' ? 'checked' : ''}> Request persistent vault storage</label>
    ${detail([['Session','Held until the console closes'], ['Vault','Availability not checked']])}${notice('Choose storage explicitly', 'A storage request is not a saved credential. Keep the previous reference if the native vault refuses it.', 'info')}
    <div class="action-row">${unavailable('Save key', 'Credential saving requires an available native credential backend.', 'key-reason')}${action('Back; drop unsaved key', 'data-close', true)}</div>${reason('key-reason', 'Credential saving requires an available native credential backend.')}</form></div>`;
  let value = draftValues[field] ?? def.value;
  if (current.variant === 'empty') value = '';
  if (current.variant === 'invalid' && ['budget','compact'].includes(field)) value = '0';
  if (current.variant === 'invalid' && field === 'endpoint') value = 'gateway-a.invalid';
  if (current.variant === 'long') value = field === 'model' ? 'exact-model-id-研究-é-long-context-development-2026-11-03' : field === 'endpoint' ? 'https://gateway-a.invalid/local-routing/long-endpoint/v1' : value;
  return `<div class="layer-body" data-field="${field}">${eyebrow(`${current.screen} / field editor`)}${title(def.title, 'Esc keeps the previous draft. Enter keeps this field in the draft.')}
    <form class="editor-form" id="editor-form" novalidate><label for="editor-value">${e(def.label)}</label><input id="editor-value" type="text" value="${e(value)}" ${['budget','compact'].includes(field) ? 'inputmode="numeric"' : ''} autocomplete="off" spellcheck="false" aria-describedby="editor-hint editor-error" data-initial-focus>
    <p class="editor-hint" id="editor-hint">${e(def.hint)}</p><p class="editor-error" id="editor-error" role="status"></p>
    ${notice(field === 'endpoint' ? 'Local-only endpoint' : 'Draft only', def.note, 'info')}
    <div class="action-row"><button class="action" type="submit">${e(def.action)}</button>${action('Back; keep previous', 'data-close', true)}</div></form></div>`;
}
function help() {
  return `<div class="layer-body">${eyebrow('Help / keyboard')}${title('A place for every key', 'Focus marks the next action. Selected labels keep your place.')}
    ${detail([['Tab / Shift+Tab','Move between visible controls'], ['Enter / Space','Activate a focused button or choice'], ['Left / Right','Move between section tabs or confirmation actions'], ['Esc','Close this layer; retain the previous draft'], ['Ctrl+P','Search secondary screens and editor references'], ['?','Open help outside text entry'], ['/','Open the client filter editor'], ['q','Open the leave confirmation outside text entry']])}
    ${notice('Text entry keeps its keys', 'Typing, paste and cursor movement stay inside an editor. No shortcut starts a native operation.', 'info')}
    <div class="action-row">${action('Back to screen', 'data-close data-initial-focus', true)}</div></div>`;
}
function more() {
  const hints = {Gateway:'Endpoint and hidden credentials', Service:'Optional read-only background monitor', Repair:'Redacted manual repair brief', SSH:'Explicit terminal helper handoff', Utilities:'Install and update availability', Settings:'Display, credentials and navigation'};
  return `<div class="layer-body" data-navigation="more">${eyebrow('Navigation / secondary destinations')}${title('More destinations', 'Move focus to a destination, then Enter to open it.', `Selected: ${current.screen}`)}
    <div class="destination-list" aria-label="More destinations">${screens.map((screen,i) => `<button type="button" class="destination-row${screen === current.screen ? ' selected' : ''}" data-screen="${screen}" ${screen === current.screen ? 'aria-current="page"' : ''} ${i === 0 ? 'data-initial-focus' : ''}><strong>${e(screen)}</strong><span>${screen === current.screen ? 'Selected' : 'Open'}</span><small>${hints[screen]}</small></button>`).join('')}</div>
    <p class="selected-note">Up/Down moves within the menu. Enter opens; Esc returns to the same screen.</p><div class="action-row">${action('Back to screen', 'data-close', true)}</div></div>`;
}
function palette() {
  return `<div class="layer-body">${eyebrow('Commands / navigation')}${title('Go to a screen or field', 'Search by name. Enter opens the selected result.')}<label for="palette-search" class="sr-only">Search commands</label><input class="palette-search" id="palette-search" role="combobox" aria-expanded="true" aria-controls="palette-results" aria-autocomplete="list" autocomplete="off" placeholder="Try gateway, repair or budget" data-initial-focus><p id="palette-count" class="selected-note" role="status"></p><div id="palette-results" class="palette-results" role="listbox" aria-label="Matching commands"></div><div class="action-row">${action('Back to screen', 'data-close', true)}</div></div>`;
}
function updatePalette() {
  const commands = [...screens.map(screen => ({label:screen, attrs:`data-screen="${screen}"`, hint:'Open screen'})), ...Object.entries(editors).map(([field, def]) => ({label:def.title, attrs:`data-editor="${field}"`, hint:'Open field editor'}))];
  const query = $('#palette-search').value.toLowerCase();
  const matches = commands.filter(c => c.label.toLowerCase().includes(query));
  paletteIndex = Math.max(0, Math.min(paletteIndex, matches.length - 1));
  $('#palette-results').innerHTML = matches.map((c, i) => `<button type="button" role="option" id="palette-option-${i}" class="palette-option" tabindex="-1" aria-selected="${i === paletteIndex}" ${c.attrs}>${e(c.label)}<small>${c.hint}</small></button>`).join('');
  $('#palette-count').textContent = matches.length ? `${matches.length} results · Up/Down choose · Enter open` : 'No matching commands. Try “Gateway” or “budget”.';
  if (matches.length) {
    $('#palette-search').setAttribute('aria-activedescendant', `palette-option-${paletteIndex}`);
    $(`#palette-option-${paletteIndex}`).scrollIntoView({block:'nearest'});
  } else $('#palette-search').removeAttribute('aria-activedescendant');
}
function render({push = false, anchor} = {}) {
  rendering = true;
  // Editors take precedence; a URL can never stack two layers.
  if (current.editor) current.overlay = '';
  document.documentElement.dataset.theme = current.theme;
  const root = $('#native-reference');
  Object.assign(root.dataset, {layout:current.layout, screen:current.screen, state:current.state, overlay:current.overlay, editor:current.editor});
  $('#reference-screen').innerHTML = screens.map(s => `<option>${s}</option>`).join('');
  $('#reference-state').innerHTML = states[current.screen].map(s => `<option>${s}</option>`).join('');
  $('#reference-layer').innerHTML = `<option value="">Screen body</option><optgroup label="Confirmations & navigation">${overlays.map(s => `<option value="overlay:${s}">${overlayTitles[s]}</option>`).join('')}</optgroup><optgroup label="Field editors">${Object.entries(editors).map(([id, def]) => `<option value="editor:${id}">${def.title}</option>`).join('')}</optgroup>`;
  for (const key of ['screen','theme','layout','state']) $(`#reference-${key}`).value = current[key];
  $('#reference-layer').value = current.editor ? `editor:${current.editor}` : current.overlay ? `overlay:${current.overlay}` : '';
  $('#reference-nav').innerHTML = ['Dashboard','Models','Review','Jobs'].map((s,i) => `<a href="./tui-preview.html#console" title="Open the approved primary preview; choose ${s} there"><span aria-hidden="true">0${i+1}</span> ${s}</a>`).join('') + '<button type="button" data-overlay="more" aria-current="page" aria-haspopup="true" aria-expanded="'+String(current.overlay === 'more')+'"><span aria-hidden="true">05</span> More</button>';
  const layerName = current.editor ? editors[current.editor].title : overlayTitles[current.overlay];
  $('#breadcrumb').innerHTML = `${e(current.screen)}${layerName ? ` / <strong>${e(layerName)}</strong>` : ' / <strong>Overview</strong>'}`;
  $('#context-state').textContent = current.editor ? 'Previous draft retained until save' : current.overlay ? 'Back keeps working' : ({Gateway:'Local connection', Service:'Optional observation', Repair:'Manual operation', SSH:'Explicit handoff', Utilities:'Read only', Settings:'This session'})[current.screen];
  $('#reference-body').innerHTML = current.editor ? editor() : current.overlay === 'help' ? help() : current.overlay === 'palette' ? palette() : current.overlay === 'more' ? more() : current.overlay ? confirmation() : views[current.screen]();
  $('#reference-body').setAttribute('aria-labelledby', 'reference-title');
  $('#reference-status').textContent = current.overlay === 'quit' && ['busy','dirty-busy'].includes(current.variant) ? '1 running job / wait for outcome' : current.screen === 'Service' && current.state === 'pending' ? 'Awaiting native observation' : current.screen === 'Repair' && current.state === 'preparing' ? 'Preparing manual brief' : current.screen === 'SSH' && current.state === 'preparing' ? 'Preparing SSH command' : 'No active jobs';
  $('#key-hints').innerHTML = current.editor === 'key' ? '<kbd>Enter</kbd> next · <kbd>Tab</kbd> move · <kbd>Esc</kbd> drop' : current.editor ? '<kbd>Enter</kbd> keep field · <kbd>Esc</kbd> back' : ['palette','more'].includes(current.overlay) ? '<kbd>Up/Down</kbd> choose · <kbd>Esc</kbd> back' : current.overlay ? '<kbd>Tab</kbd> choose · <kbd>Esc</kbd> back' : '<kbd>Tab</kbd> move · <kbd>Enter</kbd> open';
  const q = new URLSearchParams({screen:current.screen, theme:current.theme, layout:current.layout, state:current.state});
  for (const key of ['overlay','editor','variant','focus']) if (current[key]) q.set(key,current[key]);
  const url = `${location.pathname}?${q}`;
  const navigation={layerReturn,screenReturns};
  if (push) history.pushState(navigation,'',url); else history.replaceState(navigation,'',url);
  document.title = `oneline-ccc · ${current.screen}${layerName ? ` / ${layerName}` : ''} · ${current.state} · ${current.theme} ${current.layout} reference`;
  if (current.overlay === 'palette') updatePalette();
  if (current.editor && current.editor !== 'key') {
    const input = $('#editor-value');
    if (current.variant === 'invalid') validateEditor(false);
    const pos = current.variant === 'cursor-start' ? 0 : current.variant === 'cursor-middle' ? Math.floor(input.value.length/2) : input.value.length;
    input.setSelectionRange(pos,pos);
  }
  const selector = current.focus ? `[data-focus-id="${CSS.escape(current.focus)}"]` : '';
  $('#reference-body').querySelectorAll('button,input').forEach((el,i) => { el.dataset.focusId = String(i); });
  root.querySelectorAll('button').forEach(button => {
    if (button.querySelector('.focus-marker')) return;
    const marker=document.createElement('span'); marker.className='focus-marker'; marker.setAttribute('aria-hidden','true'); marker.textContent='>';
    button.prepend(marker);
  });
  const restored = anchor?.selector && $(anchor.selector);
  const target = restored || (selector && $(selector)) || $('#reference-body [data-initial-focus]') || $('#reference-body button:not([aria-disabled="true"])') || $('#reference-title');
  if (restored) {
    target.focus({preventScroll:true});
    window.scrollTo(0,anchor.y);
    window.scrollBy(0,target.getBoundingClientRect().top-anchor.top);
  } else target.focus();
  keepVisible(target);
  rendering = false;
  rememberVisit();
}
function change(patch, options = {}) {
  rememberVisit();
  if ('layerReturn' in options) layerReturn=options.layerReturn;
  if ('screenReturns' in options) screenReturns=options.screenReturns;
  current = {...current,...patch}; render({push:true,anchor:options.anchor});
}
function closeLayer() {
  const previous=layerReturn;
  change(previous ? {...previous.view,theme:current.theme,layout:current.layout} : {overlay:'',editor:'',variant:'',focus:''},{anchor:previous?.anchor,layerReturn:null});
}
function goToScreen(screen) {
  const previous=layerReturn || {view:{...current},anchor:captureAnchor()};
  if (screen === previous.view.screen) {
    change({...previous.view,theme:current.theme,layout:current.layout},{layerReturn:null,anchor:previous.anchor});
    return;
  }
  change({screen,state:states[screen][0],overlay:'',editor:'',variant:'',focus:''},{screenReturns:[...screenReturns,previous],layerReturn:null});
}
function openLayer(kind, name) {
  const previous=layerReturn || {view:{...current},anchor:captureAnchor()};
  change({overlay:'',editor:'',variant:'',focus:'',[kind]:name},{layerReturn:previous});
}
function validateEditor(save) {
  const field = current.editor, input = $('#editor-value');
  if (!input) return false;
  const value = input.value;
  let error = '';
  if (['budget','compact'].includes(field) && (!/^[1-9]\d*$/.test(value) || !Number.isSafeInteger(Number(value)))) error = 'Enter a positive whole number of tokens. The previous draft is retained.';
  if (field === 'model' && !value.trim()) error = 'Enter the exact model ID, or go Back to keep the previous draft.';
  if (field === 'credential' && !/^[A-Za-z_][A-Za-z0-9_]*$/.test(value)) error = 'Use an environment variable name such as GATEWAY_A_API_KEY, not a key value.';
  if (field === 'endpoint') {
    try { const url = new URL(value); if (!['http:','https:'].includes(url.protocol) || url.username || url.password || url.search || url.hash) throw new Error(); }
    catch { error = 'Use an absolute HTTP or HTTPS endpoint without credentials, query or fragment.'; }
  }
  $('#editor-error').textContent = error;
  input.setAttribute('aria-invalid', String(!!error));
  if (error) { input.focus({preventScroll:true}); return false; }
  if (save) {
    draftValues[field] = value;
    if (field === 'budget') {
      const n=Number(value);
      draftValues.compact=String(Math.floor(n/10)*9 + Math.floor(n%10*9/10));
    }
    closeLayer(); announce(`Design reference: ${editors[field].title} kept in page memory only. No native configuration changed.`);
  }
  return true;
}
$('#reference-controls').addEventListener('submit', event => event.preventDefault());
$('#reference-controls').addEventListener('change', event => {
  const name = event.target.name;
  if (name === 'layer') {
    const value = event.target.value;
    if (value.startsWith('overlay:')) openLayer('overlay',value.slice(8));
    else if (value.startsWith('editor:')) openLayer('editor',value.slice(7));
    else closeLayer();
  } else if (name === 'screen') goToScreen(event.target.value);
  else change({[name]:event.target.value,focus:''});
});
$('#native-reference').addEventListener('click', event => {
  const button = event.target.closest('button');
  if (!button) return;
  if (button.getAttribute('aria-disabled') === 'true') { announce(button.dataset.reason); return; }
  if (button.hasAttribute('data-back')) {
    const previous=screenReturns.at(-1);
    if (previous) change({...previous.view,theme:current.theme,layout:current.layout},{anchor:previous.anchor,screenReturns:screenReturns.slice(0,-1),layerReturn:null});
    else goToScreen('Settings');
    return;
  }
  if (button.dataset.screen) { goToScreen(button.dataset.screen); return; }
  if (button.dataset.overlay) { openLayer('overlay',button.dataset.overlay); return; }
  if (button.dataset.editor) { openLayer('editor',button.dataset.editor); return; }
  if (button.hasAttribute('data-close')) { closeLayer(); return; }
  if (button.dataset.toggle) { display[button.dataset.toggle] = !display[button.dataset.toggle]; const id=button.dataset.focusId; current.focus=id; render(); return; }
  switch (button.dataset.action) {
    case 'cycle-theme': change({theme:['dark','light','mono'][(['dark','light','mono'].indexOf(current.theme)+1)%3],focus:button.dataset.focusId}); break;
    case 'prepare-brief': change({state:'ready'}); announce('Design reference: displaying the fixed redacted brief. No diagnostics were read.'); break;
    case 'export-brief': announce('Design reference: export requires the native terminal. Nothing was written or copied.'); break;
    case 'inspect-service': announce('Design reference: displaying a fixed observation. No native status was queried.'); break;
    case 'service-request': announce('Design reference: lifecycle action requires the native backend. No service request was sent.'); break;
    case 'confirm': announce(`Design reference: ${current.overlay} confirmation selected. No native action ran; Back returns to the screen.`); break;
  }
});
$('#native-reference').addEventListener('submit', event => { event.preventDefault(); if (current.editor !== 'key') validateEditor(true); });
$('#native-reference').addEventListener('input', event => { if (event.target.id === 'palette-search') { paletteIndex=0; updatePalette(); } });
$('#native-reference').addEventListener('keydown', event => {
  if (event.isComposing || event.keyCode === 229) return;
  if (event.key === 'Escape' && (current.overlay || current.editor)) { event.preventDefault(); closeLayer(); return; }
  if (event.target.matches('input,textarea,select')) {
    if (event.target.id === 'key-input' && event.key === 'Enter') { event.preventDefault(); $('#native-reference [aria-disabled="true"]').focus(); }
    if (event.target.id === 'palette-search') {
      if (['ArrowDown','ArrowUp'].includes(event.key)) { event.preventDefault(); paletteIndex += event.key === 'ArrowDown' ? 1 : -1; updatePalette(); }
      if (event.key === 'Enter') { event.preventDefault(); $('#palette-results [aria-selected="true"]')?.click(); }
    }
    return;
  }
  if (current.overlay === 'more' && ['ArrowUp','ArrowDown'].includes(event.key)) {
    event.preventDefault();
    const buttons=[...document.querySelectorAll('.destination-list button')];
    const index=buttons.indexOf(document.activeElement);
    buttons[Math.max(0,Math.min(buttons.length-1,index+(event.key === 'ArrowDown' ? 1 : -1)))]?.focus();
    return;
  }
  if (['ArrowLeft','ArrowRight'].includes(event.key)) {
    const container = event.target.closest('#reference-nav, [data-confirmation] .action-row');
    if (container) { event.preventDefault(); const buttons=[...container.querySelectorAll('button,a')]; const i=buttons.indexOf(document.activeElement); buttons[(i+(event.key === 'ArrowRight' ? 1 : buttons.length-1))%buttons.length]?.focus(); }
    return;
  }
  if (current.overlay || current.editor) return;
  if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'p') { event.preventDefault(); openLayer('overlay','palette'); }
  if (event.key === '?') { event.preventDefault(); openLayer('overlay','help'); }
  if (event.key === '/') { event.preventDefault(); openLayer('editor','filter'); }
  if (event.key === 'q') { event.preventDefault(); openLayer('overlay','quit'); }
});
window.addEventListener('popstate', event => {
  current=fromURL();layerReturn=event.state?.layerReturn || null;screenReturns=event.state?.screenReturns || [];
  render({anchor:event.state?.anchor});
});
history.scrollRestoration='manual';
window.addEventListener('scroll',rememberVisit,{passive:true});
document.addEventListener('focusin',rememberVisit);
current=fromURL();
render();
