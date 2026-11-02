import { escapeHTML as e } from './tui-views.mjs';

export function createDialogs(root, getCommands, execute) {
  const dialog = document.createElement('dialog');
  dialog.className = 'preview-dialog';
  dialog.setAttribute('aria-labelledby', 'dialog-title');
  root.append(dialog);
  let invoker, restore = true, matches = [], index = 0;
  let composing = false;
  dialog.addEventListener('compositionstart', () => { composing = true; });
  dialog.addEventListener('compositionend', () => { requestAnimationFrame(() => { composing = false; }); });
  function close(command) {
    restore = !command;
    dialog.close();
    if (command) command();
  }
  dialog.addEventListener('close', () => {
    if (restore && invoker?.isConnected) invoker.focus();
  });
  dialog.addEventListener('cancel', event => { event.preventDefault(); close(); });
  dialog.addEventListener('click', event => {
    if (event.target.closest('[data-close]')) close();
    const item = event.target.closest('[data-palette-command]');
    if (item && !item.hasAttribute('aria-disabled')) close(() => execute(item.dataset.paletteCommand));
    if (event.target.closest('[data-confirm]')) close(dialog.confirmAction);
  });
  dialog.addEventListener('keydown', event => {
    event.stopPropagation();
    if (event.isComposing || composing || event.keyCode === 229) return;
    if (event.key === 'Tab') {
      const controls = [...dialog.querySelectorAll('button:not(:disabled),input:not(:disabled),select:not(:disabled),a[href],[tabindex="0"]')].filter(el => el.getClientRects().length);
      const current = controls.indexOf(document.activeElement);
      if (event.shiftKey && current <= 0) { event.preventDefault(); controls.at(-1)?.focus(); }
      else if (!event.shiftKey && (current === controls.length - 1 || current < 0)) { event.preventDefault(); controls[0]?.focus(); }
      return;
    }
    if (!dialog.querySelector('#palette-search')) return;
    if (['ArrowDown', 'ArrowUp'].includes(event.key)) {
      event.preventDefault();
      if (matches.length) index = (index + (event.key === 'ArrowDown' ? 1 : -1) + matches.length) % matches.length;
      highlight();
    } else if (event.key === 'Enter' && event.target.id === 'palette-search') {
      event.preventDefault();
      const command = matches[index];
      if (command && !command.disabled) close(() => execute(command.id));
    }
  });
  function open(heading, body) {
    invoker = document.activeElement; restore = true; composing = false;
    dialog.innerHTML = `<div class="dialog-heading"><h2 id="dialog-title">${e(heading)}</h2><button type="button" class="action secondary" data-close>Close / Esc</button></div>${body}`;
    dialog.showModal();
  }
  function highlight() {
    const input = dialog.querySelector('#palette-search');
    dialog.querySelectorAll('[data-palette-command]').forEach((item, i) => {
      item.setAttribute('aria-selected', String(i === index));
      if (i === index) { input.setAttribute('aria-activedescendant', item.id); item.scrollIntoView({ block: 'nearest' }); }
    });
    if (!matches.length) input.removeAttribute('aria-activedescendant');
  }
  function results(query = '') {
    matches = getCommands().filter(c => c.palette !== false && `${c.label} ${c.reason || ''}`.toLowerCase().includes(query.toLowerCase().trim()));
    index = 0;
    dialog.querySelector('#palette-results').innerHTML = matches.map((c, i) => `<div role="option" id="command-${i}" class="palette-item" data-palette-command="${c.id}" ${c.disabled ? 'aria-disabled="true"' : ''}><strong>${e(c.label)}</strong><span>${e(c.shortcut || '')}</span>${c.disabled ? `<small>${e(c.reason)}</small>` : ''}</div>`).join('');
    dialog.querySelector('#palette-count').textContent = matches.length ? `${matches.length} commands. ↑ ↓ choose · Enter open · Esc close` : 'No matching commands. Try “client”, “review” or “refresh”.';
    highlight();
  }
  return {
    get isOpen() { return dialog.open; },
    close,
    palette() {
      open('Commands', '<label class="sr-only" for="palette-search">Search commands</label><input id="palette-search" type="text" role="combobox" aria-autocomplete="list" aria-expanded="true" aria-controls="palette-results" autocomplete="off" placeholder="Search commands…"><p id="palette-count" class="muted" role="status"></p><div id="palette-results" role="listbox" aria-label="Commands"></div>');
      results();
      const input = dialog.querySelector('#palette-search');
      input.addEventListener('input', () => results(input.value)); input.focus();
    },
    help() {
      open('Keyboard shortcuts', `<p class="muted">Active inside the preview console. Text fields keep normal typing. Tab reaches every action.</p><dl class="shortcut-list">${getCommands().filter(c => c.shortcut).map(c => `<dt><kbd>${e(c.shortcut)}</kbd></dt><dd>${e(c.label)}</dd>`).join('')}<dt><kbd>↑ ↓</kbd></dt><dd>Move between matching client rows</dd><dt><kbd>Space</kbd></dt><dd>On a client row or checkbox: stage management on/off</dd><dt><kbd>Enter</kbd></dt><dd>On a client row: open settings; otherwise activate focused control</dd><dt><kbd>Esc</kbd></dt><dd>Close the top panel or return one screen; keep drafts</dd><dt><kbd>Tab / Shift+Tab</kbd></dt><dd>Next / previous control</dd></dl><p class="muted">Ctrl+P outside the console retains browser Print. Apply is a separate explicit button on Review. No native quit or client-launch shortcuts run here.</p>`);
      dialog.querySelector('[data-close]').focus();
    },
    confirm(title, description, label, action) {
      open(title, `<p>${e(description)}</p><div class="action-row"><button type="button" class="action secondary" data-close>Keep draft / Back</button><button type="button" class="action" data-confirm>${e(label)}</button></div>`);
      dialog.confirmAction = action;
      dialog.querySelector('[data-close]').focus();
    }
  };
}
