import { createState, clientById, selectClient, editDraft, pendingChanges, applyDrafts, discardDraft, refreshSnapshot, validateDraft } from './tui-state.mjs';
import { renderView, filteredClients } from './tui-views.mjs';
import { commandsFor } from './tui-commands.mjs';
import { createDialogs } from './tui-dialogs.mjs';

const state = createState();
const screen = document.getElementById('screen');
const consoleView = document.getElementById('console');
const announcement = document.getElementById('announcement');
let activeScreen = 'dashboard', filter = '', reviewIds = null, reviewSnapshot;
let composing = false, compositionInput = null, compositionClient;
const commands = () => commandsFor(state, activeScreen, filter, reviewIds);
const dialogs = createDialogs(consoleView, commands, execute);
const focus = selector => (screen.querySelector(selector) || screen.querySelector('h2')).focus();
function announce(message = state.notice) {
  announcement.textContent = message;
  document.getElementById('console-status').textContent = message || 'Preview only · no live operations';
}
function render(focusSelector) {
  const old = document.activeElement;
  const field = old?.id, start = old?.selectionStart, end = old?.selectionEnd;
  const anchor = ['command', 'client', 'enable', 'screen'].find(key => old?.dataset?.[key]);
  const anchorSelector = anchor ? `[data-${anchor}="${old.dataset[anchor]}"]` : null;
  const advanced = screen.querySelector('#advanced')?.open;
  screen.innerHTML = renderView(state, activeScreen, filter, commands(), reviewIds);
  if (activeScreen === 'review') reviewSnapshot = pendingChanges(state).filter(c => !reviewIds || reviewIds.includes(c.id));
  if (advanced && screen.querySelector('#advanced')) screen.querySelector('#advanced').open = true;
  document.querySelectorAll('[data-screen]').forEach(b => {
    if (b.dataset.screen === activeScreen) b.setAttribute('aria-current', 'page');
    else b.removeAttribute('aria-current');
  });
  if (focusSelector) focus(focusSelector);
  else if (old && !old.isConnected && field && document.getElementById(field)) {
    const next = document.getElementById(field); next.focus();
    if (typeof start === 'number' && next.setSelectionRange) next.setSelectionRange(start, end);
  } else if (old && !old.isConnected && anchorSelector) focus(anchorSelector);
}
function openScreen(name) {
  activeScreen = name; render(name === 'review' ? '[data-command="back"]' : 'h2');
  announce(`${screen.querySelector('h2').textContent}. Drafts kept. Preview only.`);
}
function choose(id, detail = false) {
  selectClient(state, id);
  if (!filteredClients(state, filter).some(c => c.id === id)) filter = '';
  if (detail) openScreen('models');
  else { render(`[data-client="${id}"]`); announce(`Selected ${clientById(state, id).name}.`); }
}
function move(step) {
  const visible = filteredClients(state, filter);
  if (!visible.length) return;
  const i = visible.findIndex(c => c.id === state.selected);
  choose(visible[(i + step + visible.length) % visible.length].id);
}
function stageEnable(id, enabled, selector) {
  editDraft(state, id, 'enabled', enabled);
  render(selector);
  announce(`${clientById(state, id).name} management ${enabled ? 'enabled' : 'disabled'} in draft. Review to apply.`);
}
function execute(id) {
  commitComposition();
  const command = commands().find(c => c.id === id);
  if (!command || command.disabled) { announce(command?.reason || 'Unavailable action.'); return; }
  const c = clientById(state, state.selected);
  try {
    switch (id) {
      case 'dashboard':
        activeScreen = 'dashboard';
        if (!filteredClients(state, filter).some(client => client.id === c.id)) filter = '';
        render(`[data-client="${c.id}"]`); announce(`Back to ${c.name}. Drafts kept.`); break;
      case 'models': case 'jobs': openScreen(id); break;
      case 'configure': openScreen('models'); break;
      case 'review': reviewIds = [c.id]; openScreen('review'); break;
      case 'review-all': reviewIds = null; openScreen('review'); break;
      case 'apply': applyDrafts(state, reviewIds || undefined, reviewSnapshot); activeScreen = 'jobs'; render('h2'); announce(); break;
      case 'back': activeScreen = 'models'; render('#editor-enabled'); announce('Back to settings. Drafts kept.'); break;
      case 'discard': dialogs.confirm(`Discard ${c.name} draft?`, 'Only this client’s unsaved changes will be discarded. Applied values and other drafts stay.', 'Discard draft', () => { discardDraft(state, c.id); render(activeScreen === 'models' ? '#editor-enabled' : `[data-client="${c.id}"]`); announce(); }); break;
      case 'toggle': stageEnable(c.id, !state.drafts[c.id].enabled, activeScreen === 'dashboard' ? `[data-client="${c.id}"]` : '#editor-enabled'); break;
      case 'refresh': refreshSnapshot(state); render(); announce(); break;
      case 'filter': if (activeScreen !== 'dashboard') openScreen('dashboard'); focus('#client-filter'); break;
      case 'clear-filter': filter = ''; render('#client-filter'); announce('Filter cleared. Drafts kept.'); break;
      case 'next': move(1); break;
      case 'previous': move(-1); break;
      case 'help': dialogs.help(); break;
      case 'palette': dialogs.palette(); break;
    }
  } catch (error) { announce(error.message); }
}
consoleView.addEventListener('click', event => {
  if (dialogs.isOpen) return;
  commitComposition();
  const command = event.target.closest('[data-command]');
  const client = event.target.closest('[data-client]');
  const edit = event.target.closest('[data-edit]');
  if (command) execute(command.dataset.command);
  else if (client) choose(client.dataset.client);
  else if (edit) choose(edit.dataset.edit, true);
  const nav = event.target.closest('[data-screen]');
  if (nav) { if (nav.dataset.screen === 'review') reviewIds = null; openScreen(nav.dataset.screen); }
});
screen.addEventListener('change', event => {
  commitComposition();
  const input = event.target;
  if (input.dataset.enable) {
    stageEnable(input.dataset.enable, input.checked, input.id ? `#${input.id}` : `[data-enable="${input.dataset.enable}"]`);
  }
});
function updateInput(input, clientId = state.selected) {
  if (input.id === 'client-filter') {
    filter = input.value;
    const visible = filteredClients(state, filter);
    if (visible.length && !visible.some(c => c.id === state.selected)) selectClient(state, visible[0].id);
    render(); announce(`${visible.length} matching clients. Drafts kept.`);
  } else if (input.dataset.field) {
    state.notice = '';
    editDraft(state, clientId, input.dataset.field, input.value);
    render();
    const error = validateDraft(state, clientId);
    if (state.notice || error) announce(state.notice || error);
  }
}
function commitComposition() {
  if (composing || !compositionInput) return;
  const input = compositionInput, clientId = compositionClient;
  compositionInput = null;
  updateInput(input, clientId);
}
screen.addEventListener('compositionstart', event => {
  composing = true; compositionInput = event.target; compositionClient = state.selected;
});
screen.addEventListener('compositionend', () => {
  composing = false;
  // Keep the same node for the final committed input event, then render once.
  requestAnimationFrame(commitComposition);
});
screen.addEventListener('input', event => {
  if (!event.isComposing && event.target !== compositionInput) updateInput(event.target);
});
screen.addEventListener('submit', event => event.preventDefault());
consoleView.addEventListener('keydown', event => {
  if (event.defaultPrevented || event.isComposing || composing || compositionInput || event.keyCode === 229 || dialogs.isOpen) return;
  if (event.target.closest('input,textarea,select,[contenteditable="true"]')) return;
  const row = event.target.closest('[data-client]');
  if (row && ['ArrowDown', 'ArrowUp', ' ', 'Enter'].includes(event.key)) {
    event.preventDefault();
    if (event.key === ' ') {
      if (event.repeat) return;
      const c = clientById(state, row.dataset.client);
      try { stageEnable(c.id, !state.drafts[c.id].enabled, `[data-client="${c.id}"]`); } catch (error) { announce(error.message); }
    } else if (event.key === 'Enter') choose(row.dataset.client, true);
    else move(event.key === 'ArrowDown' ? 1 : -1);
    return;
  }
  let id;
  if (event.ctrlKey && !event.altKey && !event.metaKey && event.key.toLowerCase() === 'p') id = 'palette';
  else if (!event.ctrlKey && !event.altKey && !event.metaKey) id = ({ '?': 'help', '/': 'filter', r: 'refresh' })[event.key];
  if (id) { event.preventDefault(); execute(id); }
  else if (event.key === 'Escape' && activeScreen !== 'dashboard') { event.preventDefault(); execute(activeScreen === 'review' ? 'back' : 'dashboard'); }
});
document.getElementById('theme').addEventListener('change', event => { document.documentElement.dataset.theme = event.target.value; });
document.getElementById('layout').addEventListener('change', event => { consoleView.dataset.layout = event.target.value; });
render();
