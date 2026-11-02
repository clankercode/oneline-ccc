'use strict';

const clients = [
  { name: 'Codex', state: 'CONFIGURED', tone: 'good', model: 'gpt-6.1-sol', budget: '500000', tier: 'Ultrafast · fixture attestation', source: 'Local configuration', note: 'Configuration found. Authentication and inference are unverified.', action: 'Review configuration' },
  { name: 'Claude Code', state: 'NEEDS SETUP', tone: 'muted', model: 'Not selected', budget: 'Not selected', tier: 'Not selected', source: 'No managed profile', note: 'Choose a gateway and model before reviewing configuration.', action: 'Explore model choices' },
  { name: 'OpenCode', state: 'DRIFT', tone: 'warn', model: 'glm-5.3', budget: '353000', tier: 'Default', source: 'Native and saved differ', note: 'The context budget changed outside this manager. Review both values before writing.', action: 'Review changes' },
  { name: 'Pi', state: 'NOT INSTALLED', tone: 'muted', model: 'Not selected', budget: 'Not selected', tier: 'Not selected', source: 'Illustrative detection', note: 'Installation support needs its own verified client adapter.', action: 'Explore model choices' },
  { name: 'Grok Build', state: 'UNVERIFIED', tone: 'warn', model: 'gpt-6.1-sol', budget: '272000', tier: 'Unverified', source: 'Adapter not verified', note: 'This client remains read-only until its schema and version are verified.', action: 'Explore model choices' },
  { name: 'Hermes', state: 'CUSTOM', tone: 'muted', model: 'Custom configuration', budget: 'Unknown', tier: 'Unknown', source: 'Operator-owned configuration', note: 'Preserve custom settings. Inspect compatibility before any managed changes.', action: 'Explore model choices' }
];
const screen = document.getElementById('screen');
const consoleView = document.getElementById('console');
const announcement = document.getElementById('announcement');
let activeScreen = 'dashboard';
let selectedClient = 0;

const title = (heading, subtitle, meta) => `<div class="screen-title"><div><h2>${heading}</h2><p>${subtitle}</p></div><span>${meta}</span></div>`;
const action = (label, destination, secondary = false) => `<button type="button" class="action${secondary ? ' secondary' : ''}" data-open="${destination}">${label}</button>`;

function dashboard() {
  const c = clients[selectedClient];
  return title('Your clients', 'One gateway. Each client keeps its own configuration.', 'Snapshot: illustrative') + `
  <div class="workarea"><div><p class="section-label">CLIENT / CONFIGURATION</p><div class="client-list" aria-label="Select a client">
  ${clients.map((client, index) => `<button type="button" class="client" data-client="${index}" aria-pressed="${index === selectedClient}"><span class="client-name">${client.name}<small>${client.model}</small></span><span class="state ${client.tone}">${client.state}</span></button>`).join('')}
  </div><p class="selected-note">Selected: ${c.name} · ${selectedClient + 1} of ${clients.length}</p><div class="narrow-controls">${action('Previous client', 'previous', true)}${action('Next client', 'next', true)}</div></div>
  <div class="detail"><p class="section-label">SELECTED CLIENT</p><h3>${c.name} <span class="${c.tone}">/ ${c.state}</span></h3><dl><dt>Gateway</dt><dd>Gateway A</dd><dt>Model</dt><dd>${c.model}</dd><dt>Budget</dt><dd>${c.budget} ${/^\d/.test(c.budget) ? 'tokens' : ''}</dd><dt>Tier</dt><dd>${c.tier}</dd><dt>Source</dt><dd>${c.source}</dd><dt>Auth / API</dt><dd class="muted">Unverified / not tested</dd></dl><div class="notice"><strong>Before you continue</strong><p>${c.note}</p></div><div class="action-row">${action(c.action, c.state === 'DRIFT' ? 'review' : 'models')}</div></div></div>
  <div class="action-row"><span class="muted">Monitor: not installed · optional</span><span class="warn">1 configuration drift</span></div>`;
}

function models() {
  return title('Models & budgets', 'Keep model identity, context budget and request tier separate.', 'Gateway A / illustrative inventory') + `
  <div class="table-wrap"><table><thead><tr><th>MODEL ID</th><th>CHOSEN BUDGET</th><th>REQUEST TIER</th><th>CAPACITY EVIDENCE</th></tr></thead><tbody>
  <tr><td>gpt-6.1-sol</td><td data-label="Budget">500000</td><td data-label="Tier">Ultrafast*</td><td data-label="Capacity" class="warn">Not verified</td></tr>
  <tr><td>glm-5.3</td><td data-label="Budget">353000</td><td data-label="Tier">Default</td><td data-label="Capacity" class="warn">Not verified</td></tr>
  <tr><td>glm-5.3-flash</td><td data-label="Budget">272000</td><td data-label="Tier">Default</td><td data-label="Capacity" class="warn">Not verified</td></tr>
  </tbody></table></div><p class="selected-note">* Tier shown from illustrative operator attestation, not an entitlement check.</p>
  <div class="notice"><strong>Budgets are your limits, not provider capacities.</strong><p>A shared budget must fit every linked model. Model visibility alone does not establish context capacity, protocol support or tier eligibility.</p></div>
  <div class="action-row">${action('Review budget change', 'review')}${action('Back to clients', 'dashboard', true)}</div>`;
}

function review() {
  return title('Review changes', 'OpenCode / context budget', 'Draft / illustrative change') + `
  <div class="workarea"><div><p class="section-label">PROPOSED CHANGE</p><pre class="diff"><span class="muted">  context budget</span>
<span class="remove">- native setting    353000</span>
<span class="add">+ saved preference  500000</span></pre><dl><dt>Unrelated</dt><dd>Settings preserved</dd><dt>Credential</dt><dd>Reference only; value hidden</dd><dt>Privilege</dt><dd>Current user</dd><dt>Backup</dt><dd>Required before replacement</dd></dl></div>
  <div class="detail"><p class="section-label">BEFORE APPLY</p><h3 class="warn">Capacity is unverified</h3><p>Review the model's applicable capacity before accepting the larger budget. Keep the current value until that evidence is available.</p><div class="notice"><strong>Design preview only</strong><p>Apply is unavailable. No native settings, credentials or services are read or changed by this page.</p></div><div class="action-row">${action('Back to clients', 'dashboard', true)}<button class="action" type="button" disabled aria-describedby="apply-reason">Apply changes</button></div><p id="apply-reason" class="selected-note">Requires verified capacity and a native configuration adapter.</p></div></div>`;
}

function jobs() {
  return title('Jobs & receipts', 'Stages stay readable when work stops or needs attention.', 'Illustrative history / no running jobs') + `
  <div class="job"><span class="number">01</span><div><strong>Inspect configuration</strong><small>Read supported fields · retain source and observation time</small></div><strong class="state good">COMPLETED</strong></div>
  <div class="job"><span class="number">02</span><div><strong>Compare native and saved values</strong><small>OpenCode budget differs · review before any write</small></div><strong class="state warn">ATTENTION</strong></div>
  <div class="job"><span class="number">03</span><div><strong>Refresh model metadata</strong><small>Illustrative connection failure · keep the previous snapshot</small></div><strong class="state bad">FAILED</strong></div>
  <div class="notice"><strong>Offline / previous snapshot is stale</strong><p>Show the source and age. A failed refresh does not erase saved settings or prove the gateway is unavailable for inference.</p></div>
  <div class="action-row">${action('Review configuration drift', 'review')}${action('Back to clients', 'dashboard', true)}</div>`;
}

function render() {
  screen.innerHTML = ({ dashboard, models, review, jobs })[activeScreen]();
  document.querySelectorAll('[data-screen]').forEach(button => {
    if (button.dataset.screen === activeScreen) button.setAttribute('aria-current', 'page');
    else button.removeAttribute('aria-current');
  });
  document.getElementById('arrow-hint').hidden = activeScreen !== 'dashboard';
}

function openScreen(name, focusHeading = false) {
  activeScreen = name;
  render();
  announcement.textContent = `${name} screen. Illustrative data.`;
  if (focusHeading) {
    const heading = screen.querySelector('h2');
    heading.tabIndex = -1;
    heading.focus();
  }
}

function chooseClient(index, focusClient = false) {
  selectedClient = (index + clients.length) % clients.length;
  render();
  announcement.textContent = `Selected ${clients[selectedClient].name}. ${clients[selectedClient].state}.`;
  if (focusClient) screen.querySelector(`[data-client="${selectedClient}"]`).focus();
}

document.querySelector('.section-nav').addEventListener('click', event => {
  const button = event.target.closest('[data-screen]');
  if (button) openScreen(button.dataset.screen);
});
screen.addEventListener('click', event => {
  const button = event.target.closest('button');
  if (!button) return;
  if (button.dataset.client !== undefined) chooseClient(Number(button.dataset.client), true);
  else if (button.dataset.open === 'next') chooseClient(selectedClient + 1, true);
  else if (button.dataset.open === 'previous') chooseClient(selectedClient - 1, true);
  else if (button.dataset.open) openScreen(button.dataset.open, true);
});
screen.addEventListener('keydown', event => {
  const button = event.target.closest('[data-client]');
  if (!button || !['ArrowDown', 'ArrowUp'].includes(event.key)) return;
  event.preventDefault();
  chooseClient(Number(button.dataset.client) + (event.key === 'ArrowDown' ? 1 : -1), true);
});
document.getElementById('theme').addEventListener('change', event => {
  document.documentElement.dataset.theme = event.target.value;
});
document.getElementById('layout').addEventListener('change', event => {
  consoleView.dataset.layout = event.target.value;
});
render();
