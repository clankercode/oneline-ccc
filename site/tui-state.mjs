// In-memory specimen only. No host discovery, persistence or native operations.
const presets = [
  ['codex', 'Codex', true, true, true, 'gpt-6.1-sol', '500000', 'CONFIGURED', 'good'],
  ['claude', 'Claude Code', true, true, false, 'gpt-6.1-sol', '500000', 'NEEDS SETUP', 'muted'],
  ['opencode', 'OpenCode', true, true, true, 'glm-5.3', '353000', 'DRIFT', 'warn'],
  ['pi', 'Pi', true, false, false, 'glm-5.3-flash', '272000', 'NOT INSTALLED', 'muted'],
  ['grok', 'Grok Build', false, true, false, 'gpt-6.1-sol', '272000', 'UNVERIFIED', 'warn'],
  ['hermes', 'Hermes', false, true, false, 'Custom configuration', 'Unknown', 'CUSTOM', 'muted']
];
export const models = ['gpt-6.1-sol', 'glm-5.3', 'glm-5.3-flash'];
export const allowedTiers = model => model === 'gpt-6.1-sol' ? ['Standard', 'Fast', 'Ultrafast'] : ['Standard'];
export const clientById = (s, id) => {
  const client = s.clients.find(c => c.id === id);
  if (!client) throw new Error('Unknown client');
  return client;
};
const snapshot = c => ({ ...c.settings, enabled: c.enabled });

export function createState() {
  const clients = presets.map(([id, name, supported, installed, enabled, model, budget, status, tone]) => ({
    id, name, supported, installed, enabled, status, tone, installSupported: false, settings: { model, budget, tier: supported ? 'Standard' : null }
  }));
  return { clients, drafts: Object.fromEntries(clients.map(c => [c.id, snapshot(c)])),
    selected: 'codex', receipts: [], notice: '', refreshes: 0 };
}
export function selectClient(s, id) { clientById(s, id); s.selected = id; }
export function unavailableReason(c) {
  if (!c.supported) return c.id === 'hermes' ? 'Custom configuration: management adapter not supported.' : 'Adapter not verified: inspection only.';
  if (!c.installed) return 'Not installed. Installation adapter unavailable in this preview; inspect settings or skip this client.';
  return '';
}
export function editDraft(s, id, field, value) {
  const c = clientById(s, id);
  if (!['model', 'budget', 'tier', 'enabled'].includes(field)) throw new Error('Unknown setting');
  const reason = unavailableReason(c);
  if (reason) throw new Error(reason);
  const d = s.drafts[id];
  d[field] = field === 'enabled' ? Boolean(value) : String(value);
  if (field === 'model' && !allowedTiers(d.model).includes(d.tier)) {
    d.tier = 'Standard';
    s.notice = 'Default tier reset to Standard. This model has no established accelerated tier in the preview.';
  }
}
export function pendingChanges(s) {
  return s.clients.flatMap(c => {
    const before = snapshot(c), after = { ...s.drafts[c.id] };
    // Disabling management leaves the installed client's configuration intact.
    if (!after.enabled) Object.assign(after, c.settings);
    const fields = Object.keys(before).filter(k => before[k] !== after[k]);
    return fields.length ? [{ id: c.id, name: c.name, before, after, fields }] : [];
  });
}
export function validateDraft(s, id) {
  const c = clientById(s, id), d = s.drafts[id];
  if (!d.enabled) return '';
  const reason = unavailableReason(c);
  if (reason) return reason;
  if (!d.model.trim()) return 'Choose a model ID.';
  if (/\s/.test(d.model)) return 'Model IDs cannot contain spaces.';
  if (!/^[1-9]\d*$/.test(d.budget) || !Number.isSafeInteger(Number(d.budget))) return 'Context budget must be a positive whole number.';
  if (!allowedTiers(d.model).includes(d.tier)) return 'The selected tier is unavailable for this model.';
  return '';
}
export function applyDrafts(s, ids = pendingChanges(s).map(c => c.id), reviewed) {
  const changes = pendingChanges(s).filter(c => ids.includes(c.id));
  if (reviewed && JSON.stringify(changes) !== JSON.stringify(reviewed)) throw new Error('Draft changed since review. Review it again before applying.');
  if (!changes.length) return null;
  // Validate the complete review set before changing any applied values.
  for (const change of changes) {
    const error = validateDraft(s, change.id);
    if (error) throw new Error(`${change.name}: ${error}`);
  }
  for (const change of changes) {
    const c = clientById(s, change.id);
    c.enabled = change.after.enabled;
    if (c.enabled) {
      c.settings = { model: change.after.model, budget: change.after.budget, tier: change.after.tier };
      c.status = 'CONFIGURED'; c.tone = 'good';
    }
    if (c.enabled) s.drafts[c.id] = snapshot(c);
  }
  const receipt = { number: s.receipts.length + 1, outcome: 'PREVIEW APPLIED', changes: structuredClone(changes) };
  s.receipts.unshift(receipt);
  s.notice = 'Preview applied in memory. No files or services changed.';
  return receipt;
}
export function discardDraft(s, id) {
  s.drafts[id] = snapshot(clientById(s, id));
  s.notice = `${clientById(s, id).name} draft discarded. Other drafts kept.`;
}
export function installClient(s, id) {
  const c = clientById(s, id);
  if (c.installed) return null;
  throw new Error('Installation adapter not supported in this preview. Inspect settings or skip this client.');
}
export function refreshSnapshot(s) {
  s.refreshes += 1;
  s.notice = `Illustrative snapshot refreshed (${s.refreshes}). Drafts kept. No host or gateway contacted.`;
}
