import test from 'node:test';
import assert from 'node:assert/strict';
import { createState, selectClient, editDraft, pendingChanges, applyDrafts, discardDraft, installClient, refreshSnapshot, validateDraft, allowedTiers } from '../site/tui-state.mjs';

test('selection and refresh preserve independent drafts and management choices', () => {
  const s = createState();
  editDraft(s, 'codex', 'budget', '353000');
  editDraft(s, 'claude', 'enabled', true);
  selectClient(s, 'opencode');
  refreshSnapshot(s);
  selectClient(s, 'codex');
  assert.equal(s.drafts.codex.budget, '353000');
  assert.equal(s.drafts.claude.enabled, true);
  assert.equal(pendingChanges(s).length, 2);
  assert.equal(s.clients.find(c => c.id === 'claude').enabled, false);
});

test('review applies the chosen client, preserves other drafts and records preview outcome', () => {
  const s = createState();
  editDraft(s, 'codex', 'tier', 'Fast');
  editDraft(s, 'opencode', 'budget', '500000');
  applyDrafts(s, ['codex']);
  assert.equal(s.clients.find(c => c.id === 'codex').settings.tier, 'Fast');
  assert.equal(s.clients.find(c => c.id === 'opencode').settings.budget, '353000');
  assert.equal(pendingChanges(s)[0].id, 'opencode');
  assert.match(s.receipts[0].outcome, /preview/i);
  assert.deepEqual(pendingChanges(s).map(c => c.id), ['opencode']);
});

test('disabling management retains installation and existing configuration', () => {
  const s = createState();
  const c = s.clients.find(c => c.id === 'codex');
  const before = structuredClone(c.settings);
  editDraft(s, c.id, 'enabled', false);
  editDraft(s, c.id, 'budget', '272000');
  applyDrafts(s);
  assert.equal(c.enabled, false);
  assert.equal(c.installed, true);
  assert.deepEqual(c.settings, before);
  assert.equal(s.drafts.codex.budget, '272000');
  editDraft(s, 'codex', 'enabled', true);
  assert.equal(pendingChanges(s)[0].after.budget, '272000');
});

test('invalid changes prevent an entire batch from partly applying', () => {
  const s = createState();
  const before = structuredClone(s.clients);
  editDraft(s, 'codex', 'tier', 'Fast');
  editDraft(s, 'opencode', 'budget', '-1');
  assert.throws(() => applyDrafts(s), /positive whole/i);
  assert.deepEqual(s.clients, before);
  assert.equal(s.receipts.length, 0);
  assert.equal(pendingChanges(s).length, 2);
});

test('model changes keep identity exact and remove a tier unavailable for the new model', () => {
  const s = createState();
  editDraft(s, 'codex', 'tier', 'Ultrafast');
  editDraft(s, 'codex', 'model', 'glm-5.3');
  assert.equal(s.drafts.codex.model, 'glm-5.3');
  assert.equal(s.drafts.codex.tier, 'Standard');
  assert.deepEqual(allowedTiers('glm-5.3'), ['Standard']);
  assert.match(s.notice, /Standard/);
});

test('discard is scoped to one client and unsupported clients cannot be managed', () => {
  const s = createState();
  editDraft(s, 'codex', 'budget', '353000');
  editDraft(s, 'claude', 'enabled', true);
  discardDraft(s, 'codex');
  assert.equal(s.drafts.codex.budget, '500000');
  assert.equal(s.drafts.claude.enabled, true);
  assert.throws(() => editDraft(s, 'hermes', 'enabled', true), /custom|support/i);
  assert.equal(s.drafts.hermes.enabled, false);
});

test('missing clients retain unavailable installation state without unlocking management', () => {
  const s = createState();
  const before = structuredClone(s);
  assert.throws(() => editDraft(s, 'pi', 'enabled', true), /install/i);
  assert.throws(() => installClient(s, 'pi'), /not supported/i);
  assert.deepEqual(s, before);
});

test('stale review is refused before any batch mutations', () => {
  const s = createState();
  editDraft(s, 'codex', 'budget', '353000');
  editDraft(s, 'opencode', 'budget', '500000');
  const reviewed = pendingChanges(s);
  editDraft(s, 'opencode', 'budget', '272000');
  const before = structuredClone(s);
  assert.throws(() => applyDrafts(s, ['codex', 'opencode'], reviewed), /changed since review/i);
  assert.deepEqual(s, before);
  applyDrafts(s, ['codex', 'opencode'], pendingChanges(s));
  assert.equal(s.receipts[0].changes[1].after.budget, '272000');
});

test('no-change apply is a no-op; custom invalid model and tier are refused', () => {
  const s = createState();
  assert.equal(applyDrafts(s), null);
  editDraft(s, 'codex', 'model', '');
  assert.match(validateDraft(s, 'codex'), /model/i);
  editDraft(s, 'codex', 'model', 'custom-model');
  editDraft(s, 'codex', 'tier', 'Ultrafast');
  assert.match(validateDraft(s, 'codex'), /tier/i);
});
