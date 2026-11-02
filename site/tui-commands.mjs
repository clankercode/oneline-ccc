import { clientById, pendingChanges, validateDraft, unavailableReason } from './tui-state.mjs';

// One catalog for visible actions, palette and shortcut help.
export function commandsFor(s, screen, filter, reviewIds) {
  const c = clientById(s, s.selected), pending = pendingChanges(s);
  const own = pending.some(change => change.id === c.id);
  const reviewed = pending.filter(change => !reviewIds || reviewIds.includes(change.id));
  const errors = reviewed.map(change => validateDraft(s, change.id)).filter(Boolean);
  const command = (id, label, options = {}) => ({ id, label, ...options });
  return [
    command('dashboard', 'Back to clients'), command('models', `${c.name} model settings`, { palette: false }),
    command('jobs', 'Jobs & receipts'),
    command('configure', `${c.supported ? 'Configure' : 'Inspect'} ${c.name}`),
    command('toggle', `${s.drafts[c.id].enabled ? 'Disable' : 'Enable'} ${c.name} management`, { disabled: !!unavailableReason(c), reason: unavailableReason(c) }),
    command('review', `Review ${c.name} draft`, { disabled: !own, reason: 'No pending changes for this client.' }),
    command('review-all', `Review all drafts (${pending.length})`, { disabled: !pending.length, reason: 'No pending changes.' }),
    command('apply', 'Simulate apply', { palette: false, disabled: !reviewed.length || !!errors.length, reason: errors.join(' ') || 'No changes to apply.' }),
    command('back', 'Back to settings', { palette: false }),
    command('discard', `Discard ${c.name} draft`, { disabled: !own, reason: 'No pending changes for this client.' }),
    command('install', `${c.name} installation unavailable`, { disabled: true, reason: c.installed ? 'Already installed in this preview.' : unavailableReason(c) }),
    command('refresh', 'Refresh preview snapshot', { shortcut: 'r' }),
    command('filter', 'Filter clients', { shortcut: '/' }),
    command('clear-filter', 'Clear filter', { disabled: !filter, reason: 'Filter is already empty.' }),
    command('previous', 'Previous client', { palette: false }), command('next', 'Next client', { palette: false }),
    command('help', 'Keyboard shortcuts', { shortcut: '?' }),
    command('palette', 'Commands', { shortcut: 'Ctrl+P', palette: false })
  ];
}
