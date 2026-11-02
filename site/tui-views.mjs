import { clientById, pendingChanges, validateDraft, unavailableReason, allowedTiers, models } from './tui-state.mjs';

export const escapeHTML = value => String(value).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]);
const e = escapeHTML;
const title = (heading, subtitle, meta = 'Preview / in memory') => `<div class="screen-title"><div><h2 tabindex="-1">${e(heading)}</h2><p>${e(subtitle)}</p></div><span>${e(meta)}</span></div>`;
const modelLabel = settings => `${e(settings.model)}${settings.tier ? ` <span class="tier-label">· ${e(settings.tier)}</span>` : ''}`;
const tiers = model => allowedTiers(model).length > 1 ? 'Up to Ultrafast' : 'Accelerated tiers not verified';
const button = (commands, id, secondary = false) => {
  const c = commands.find(c => c.id === id);
  return `<button type="button" class="action${secondary ? ' secondary' : ''}" data-command="${id}" ${c.disabled ? `disabled title="${e(c.reason)}"` : ''}>${e(c.label)}</button>`;
};
export const filteredClients = (s, filter) => s.clients.filter(c => c.name.toLowerCase().includes(filter.trim().toLowerCase()));
const dirty = (s, id) => pendingChanges(s).some(c => c.id === id);

function dashboard(s, filter, commands) {
  const c = clientById(s, s.selected), d = s.drafts[c.id], reason = unavailableReason(c);
  const visible = filteredClients(s, filter), count = pendingChanges(s).length;
  const details = visible.length ? `<div class="detail"><p class="section-label">SELECTED CLIENT / APPLIED PREVIEW VALUES</p>
    <h3>${e(c.name)} <span class="${c.tone}">/ ${e(c.status)}</span></h3>
    <dl><dt>Management</dt><dd>${c.enabled ? 'Enabled' : 'Disabled'}${dirty(s, c.id) ? ` <span class="warn">· draft ${d.enabled ? 'enabled' : 'disabled'}</span>` : ''}</dd>
    <dt>Gateway</dt><dd>Gateway A / illustrative</dd><dt>Model</dt><dd>${modelLabel(c.settings)}</dd>
    <dt>Budget</dt><dd>${e(c.settings.budget)} tokens</dd><dt>Tiers</dt><dd>${tiers(c.settings.model)}</dd>
    <dt>Auth / API</dt><dd class="muted">Unverified / not tested</dd></dl>
    <div class="notice"><strong>${reason ? 'Adapter status' : 'Ready to review'}</strong><p>${e(reason || (c.id === 'opencode' && c.status === 'DRIFT' ? 'Illustrative drift: applied budget is 353000. Choose the value to keep in settings, then review.' : 'Enable management, keep the suggested settings or adjust them, then review. Nothing changes until you simulate apply.'))}</p></div>
    <div class="action-row">${button(commands, 'configure')}${!c.installed ? button(commands, 'install', true) : ''}${button(commands, 'review', true)}</div>
    <p class="selected-note">${c.supported ? 'Disable management to stop managing this client; installed software and its settings stay.' : 'Settings remain inspectable; management is unavailable.'}</p></div>` : '';
  return title('Your clients', 'Select to inspect. Check to manage. Review before applying.') + `
    <div class="summary-line"><strong>${s.clients.filter(c => c.enabled).length} managed</strong><span class="${count ? 'warn' : 'muted'}">${count} pending client${count === 1 ? '' : 's'}</span>${button(commands, 'refresh', true)}</div>
    <div class="workarea"><div><label class="section-label" for="client-filter">FILTER CLIENTS</label>
    <div class="filter-row"><input id="client-filter" type="search" placeholder="Name… (/ to focus)" value="${e(filter)}" autocomplete="off">${filter ? button(commands, 'clear-filter', true) : ''}</div>
    <p class="section-label">MANAGE / CLIENT / CONFIGURATION</p><div class="client-list" aria-label="Clients">
    ${visible.map(client => `<div class="client-entry"><input type="checkbox" data-enable="${client.id}" aria-label="Manage ${e(client.name)} (preview)" ${s.drafts[client.id].enabled ? 'checked' : ''} ${unavailableReason(client) ? `disabled title="${e(unavailableReason(client))}"` : ''}>
      <button type="button" class="client" data-client="${client.id}" aria-pressed="${client.id === s.selected}"><span class="client-name">${e(client.name)}<small>${modelLabel(client.settings)}</small><small>${client.enabled ? 'Managed' : 'Not managed'}${dirty(s, client.id) ? ' · UNSAVED DRAFT' : ''}</small></span><span class="state ${client.tone}">${e(client.status)}</span></button></div>`).join('')}
    ${visible.length ? '' : '<p class="empty">No matching clients. Clear the filter to see all clients.</p>'}</div>
    <p class="selected-note">${visible.length ? `Selected: ${e(c.name)} · ${visible.findIndex(v => v.id === c.id) + 1} of ${visible.length} matching clients` : '0 matching clients'}</p>
    <div class="narrow-controls">${button(commands, 'previous', true)}${button(commands, 'next', true)}</div></div>${details}</div>
    ${count ? `<div class="action-row">${button(commands, 'review-all')}<span class="muted">Checkboxes stage a preference. No configuration has changed.</span></div>` : ''}`;
}

function editor(s, commands) {
  const c = clientById(s, s.selected), d = s.drafts[c.id], reason = unavailableReason(c);
  const error = validateDraft(s, c.id), disabled = reason || !d.enabled;
  const options = ['500000', '353000', '272000'];
  if (!options.includes(d.budget)) options.push(d.budget);
  return title(`${c.name} / settings`, 'Suggested defaults first. Your safe draft follows you between screens.') + `
    <div class="workarea"><div><p class="section-label">MANAGEMENT PREFERENCE</p>
    <label class="check-label"><input id="editor-enabled" data-enable="${c.id}" type="checkbox" ${d.enabled ? 'checked' : ''} ${reason ? 'disabled' : ''}> Manage ${e(c.name)} in this preview</label>
    <p class="muted">${e(reason || (d.enabled ? 'Review the suggested settings, or choose your own.' : 'Settings are kept while management is disabled. Enable to edit.'))}</p>
    <form id="settings-form" novalidate><fieldset ${disabled ? 'disabled' : ''}><legend class="sr-only">${e(c.name)} model settings</legend>
    <label for="model">Model ID</label><input id="model" data-field="model" list="model-choices" value="${e(d.model)}" autocomplete="off" spellcheck="false" aria-invalid="${!!error && /model/i.test(error)}" aria-describedby="model-hint form-error">
    <datalist id="model-choices">${models.map(m => `<option value="${e(m)}"></option>`).join('')}</datalist>
    <small id="model-hint" class="muted">Choose a suggested model or type an exact custom ID.</small>
    <label for="budget">Context budget</label><select id="budget" data-field="budget">${options.map(v => `<option value="${e(v)}" ${v === d.budget ? 'selected' : ''}>${e(v)} tokens</option>`).join('')}</select>
    <label for="tier">Default request tier</label><select id="tier" data-field="tier">${d.tier === null ? '<option>Unknown / preserved</option>' : allowedTiers(d.model).map(t => `<option ${t === d.tier ? 'selected' : ''}>${t}</option>`).join('')}</select>
    <p class="selected-note">Available tiers: ${tiers(d.model)}. Model IDs stay unchanged.</p>
    <details id="advanced"><summary>Advanced / custom context budget</summary><label for="custom-budget">Budget in tokens</label><input id="custom-budget" data-field="budget" value="${e(d.budget)}" inputmode="numeric" aria-invalid="${!!error && /budget/i.test(error)}" aria-describedby="budget-hint form-error"><small id="budget-hint" class="muted">A positive whole number. Capacity is not verified by this preview.</small></details>
    </fieldset><p id="form-error" class="bad" role="status">${e(error)}</p></form></div>
    <div class="detail"><p class="section-label">DRAFT / NOT YET APPLIED</p><h3>${e(c.name)} <span class="${dirty(s, c.id) ? 'warn' : 'muted'}">/ ${dirty(s, c.id) ? 'UNSAVED' : 'NO CHANGES'}</span></h3>
    <dl><dt>Applied</dt><dd>${modelLabel(c.settings)}</dd><dt>Draft</dt><dd>${modelLabel(d)}</dd><dt>Budget</dt><dd>${e(d.budget)} tokens</dd><dt>Management</dt><dd>${d.enabled ? 'Enabled' : 'Disabled'}</dd></dl>
    <div class="notice"><strong>Preview boundary</strong><p>No gateway, credential, native configuration or software is read or changed. Tier choices here are illustrative, not live entitlement checks.</p></div>
    <div class="action-row">${button(commands, 'review')}${button(commands, 'discard', true)}${button(commands, 'dashboard', true)}</div>
    ${!c.installed && c.supported ? `<div class="action-row">${button(commands, 'install')}</div>` : ''}
    <p class="selected-note">Review covers this client. Other clients' drafts remain separate.</p></div></div>`;
}

function review(s, commands, reviewIds) {
  const changes = pendingChanges(s).filter(c => !reviewIds || reviewIds.includes(c.id));
  const errors = changes.map(c => validateDraft(s, c.id)).filter(Boolean);
  return title('Review changes', 'Check the client and each changed value before you simulate apply.', `${changes.length} client${changes.length === 1 ? '' : 's'} / preview only`) + `
    <div class="action-row review-back">${button(commands, 'back', true)}</div>
    ${changes.map(c => `<article class="review-item"><h3>${e(c.name)}</h3><p class="muted">${c.after.enabled ? 'Effective settings after apply' : 'Existing settings preserved'}: ${modelLabel(c.after)} · ${e(c.after.budget)} tokens</p><pre class="diff">${c.fields.map(f => `<span class="muted">${e(f)}</span>\n<span class="remove">− ${e(c.before[f])}</span>\n<span class="add">+ ${e(c.after[f])}</span>`).join('\n\n')}</pre>
    ${!c.after.enabled ? '<p class="muted">Stop managing only. Existing client configuration and installation are preserved.</p>' : ''}
    <button type="button" class="action secondary" data-edit="${c.id}">Edit ${e(c.name)}</button></article>`).join('')}
    ${changes.length ? '' : '<p class="empty">No pending changes in this review. Choose a client to configure or enable management.</p>'}
    <p id="review-error" class="bad" role="status">${e(errors.join(' '))}</p>
    <div class="notice"><strong>Simulated operation</strong><p>Apply updates this page’s memory and creates a preview receipt. Reloading resets it. No files, services, credentials or installed clients change. Model capacity and authentication remain unverified.</p></div>
    <div class="action-row">${button(commands, 'apply')}${button(commands, 'dashboard', true)}</div>`;
}

function jobs(s, commands) {
  return title('Jobs & receipts', 'Each simulated result links back to the changes you reviewed.', `${s.receipts.length} preview receipts / no native jobs`) + `
    ${s.receipts.length ? s.receipts.map(r => `<article class="job"><span class="number">${String(r.number).padStart(2, '0')}</span><div><strong>${e(r.changes.map(c => c.name).join(', '))}</strong><small>${r.changes.map(c => `${e(c.name)}: ${c.fields.map(f => `${e(f)} ${e(c.before[f])} → ${e(c.after[f])}`).join('; ')}<br>${c.after.enabled ? 'Effective' : 'Preserved'}: ${modelLabel(c.after)} · ${e(c.after.budget)} tokens`).join('<br>')}</small><small>In-memory preview only. No host changes.</small></div><strong class="state good">${e(r.outcome)}</strong></article>`).join('') : '<p class="empty">No receipts yet. Configure a client, review the changes, then simulate apply.</p>'}
    <div class="action-row">${button(commands, 'dashboard')}${button(commands, 'review-all', true)}</div>`;
}

export function renderView(s, name, filter, commands, reviewIds) {
  return ({ dashboard: () => dashboard(s, filter, commands), models: () => editor(s, commands),
    review: () => review(s, commands, reviewIds), jobs: () => jobs(s, commands) })[name]();
}
