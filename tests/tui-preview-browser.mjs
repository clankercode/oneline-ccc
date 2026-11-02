// Run against an already published preview; this script never copies or serves sources.
// PLAYWRIGHT_MODULE may be a package name, file URL, absolute path, or a path
// relative to this file (e.g. ../.cache/browser/node_modules/playwright/index.mjs).
// TUI_PREVIEW_URL defaults to the local design-preview server.
// Optional: TUI_PREVIEW_SCREENSHOT_DIR, or TUI_PREVIEW_SCREENSHOTS=1 for scratch.
import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { registerIMEJourneys } from './tui-preview-ime.mjs';

const baseURL = process.env.TUI_PREVIEW_URL || 'http://127.0.0.1:10435/oneline-ccc/tui-preview.html';
const origin = new URL(baseURL).origin;
const screenshotDir = process.env.TUI_PREVIEW_SCREENSHOT_DIR || (process.env.TUI_PREVIEW_SCREENSHOTS === '1'
  ? fileURLToPath(new URL('../.cache/tui-usability/browser-screenshots/', import.meta.url)) : null);
const cases = [];
const journey = (name, run, options = {}) => cases.push({ name, run, ...options });
registerIMEJourneys(journey);
const command = (page, id) => page.locator(`#console [data-command="${id}"]`);
const row = (page, id) => page.locator(`[data-client="${id}"]`);
const checkbox = (page, id) => page.locator(`[data-enable="${id}"]`);
const heading = page => page.locator('#screen h2');
const dialog = page => page.locator('dialog[open]');
const status = page => page.locator('#console-status');
const normalize = value => value.replace(/\s+/g, ' ').trim();

async function text(locator, expected) {
  const actual = normalize(await locator.innerText());
  if (expected instanceof RegExp) assert.match(actual, expected);
  else assert.equal(actual, expected);
}
async function value(page, id, expected) {
  assert.equal(await page.locator(`#${id}`).inputValue(), expected, `${id} value`);
}
async function focus(page, selector) {
  await page.waitForFunction(selector => document.activeElement?.matches(selector), selector, { timeout: 3000 });
}
async function pending(page, count) {
  await text(page.locator('.summary-line > span'), `${count} pending client${count === 1 ? '' : 's'}`);
}
async function detail(page, label, expected) {
  const values = await page.locator('#screen .detail dl').evaluate(dl => Object.fromEntries(
    [...dl.querySelectorAll('dt')].map(dt => [dt.textContent.trim(), dt.nextElementSibling.textContent])));
  const actual = normalize(values[label] ?? '');
  if (expected instanceof RegExp) assert.match(actual, expected, label);
  else assert.equal(actual, expected, label);
}
async function editor(page, id, name) {
  await row(page, id).click();
  await row(page, id).press('Enter');
  await text(heading(page), `${name} / settings`);
}
async function dashboard(page) {
  await page.locator('[data-screen="dashboard"]').click();
  await text(heading(page), 'Your clients');
}
async function noDialog(page) {
  assert.equal(await dialog(page).count(), 0, 'No modal should be open');
}
async function escapeDialog(page, invoker) {
  await page.keyboard.press('Escape');
  await noDialog(page);
  await focus(page, invoker);
}
async function trapTabs(page) {
  const count = await dialog(page).locator('button:not(:disabled),input:not(:disabled),select:not(:disabled),a[href]').count();
  assert.ok(count > 0, 'Modal has keyboard controls');
  for (const key of ['Tab', 'Shift+Tab']) {
    for (let i = 0; i < count + 2; i++) {
      await page.keyboard.press(key);
      const focused = await dialog(page).evaluate(el => ({
        inside: el.contains(document.activeElement),
        element: document.activeElement?.outerHTML.slice(0, 180)
      }));
      assert.ok(focused.inside, `${key} stays inside modal; active: ${focused.element}`);
    }
  }
}
async function noOverflow(page, where) {
  const sizes = await page.evaluate(() => ({
    viewport: document.documentElement.clientWidth,
    document: document.documentElement.scrollWidth,
    body: document.body.scrollWidth
  }));
  assert.ok(sizes.document <= sizes.viewport && sizes.body <= sizes.viewport,
    `${where}: document overflow ${JSON.stringify(sizes)}`);
}
async function screenshot(page, name) {
  if (!screenshotDir) return;
  await mkdir(screenshotDir, { recursive: true });
  await page.screenshot({ path: resolve(screenshotDir, `${name.replace(/[^a-z0-9-]+/gi, '-')}.png`), fullPage: true });
}

journey('selection inspects; row and checkbox Space stage exactly once', async page => {
  await row(page, 'claude').click();
  assert.equal(await row(page, 'claude').getAttribute('aria-pressed'), 'true');
  assert.equal(await checkbox(page, 'claude').isChecked(), false);
  await pending(page, 0);
  await detail(page, 'Management', 'Disabled');
  for (const checked of [true, false]) {
    await row(page, 'claude').press('Space');
    assert.equal(await checkbox(page, 'claude').isChecked(), checked);
    await pending(page, checked ? 1 : 0);
    await focus(page, '[data-client="claude"]');
    await text(heading(page), 'Your clients');
  }
  for (const checked of [true, false]) {
    await checkbox(page, 'claude').press('Space');
    assert.equal(await checkbox(page, 'claude').isChecked(), checked);
    await pending(page, checked ? 1 : 0);
    await focus(page, '[data-enable="claude"]');
  }
  await checkbox(page, 'claude').click();
  assert.equal(await checkbox(page, 'claude').isChecked(), true);
  await pending(page, 1);
  await detail(page, 'Management', 'Disabled · draft enabled');
  await row(page, 'claude').press('Enter');
  await text(heading(page), 'Claude Code / settings');
  assert.equal(await page.locator('#editor-enabled').isChecked(), true);
});

journey('Claude defaults: enable, review values, Back focus, apply, receipt and dashboard agree', async page => {
  await editor(page, 'claude', 'Claude Code');
  assert.equal(await page.locator('#model').isDisabled(), true);
  await page.locator('#editor-enabled').check();
  await value(page, 'model', 'gpt-6.1-sol');
  await value(page, 'budget', '500000');
  await value(page, 'tier', 'Standard');
  await command(page, 'review').click();
  await text(page.locator('.review-item h3'), 'Claude Code');
  await text(page.locator('.review-item'), /enabled.*false.*true/);
  const reviewText = normalize(await page.locator('.review-item').innerText());
  // Enabling management applies these defaults even though their values did not
  // change. Review must expose the effective configuration before consent.
  await command(page, 'back').click();
  await focus(page, '#editor-enabled');
  await value(page, 'model', 'gpt-6.1-sol');
  await command(page, 'review').click();
  await command(page, 'apply').click();
  await text(heading(page), 'Jobs & receipts');
  assert.equal(await page.locator('.job').count(), 1);
  await text(page.locator('.job'), /Claude Code.*enabled false → true.*PREVIEW APPLIED/);
  const receiptText = normalize(await page.locator('.job').innerText());
  await dashboard(page);
  await pending(page, 0);
  await detail(page, 'Management', 'Enabled');
  await detail(page, 'Model', 'gpt-6.1-sol · Standard');
  await detail(page, 'Budget', '500000 tokens');
  await text(row(page, 'claude'), /Managed.*CONFIGURED/);
  // Finish the simulated journey before reporting omissions, so the other
  // default-enable behavior is still exercised when one summary is incomplete.
  for (const expected of [/gpt-6\.1-sol/, /500000/, /Standard/]) {
    assert.match(reviewText, expected, 'Review exposes effective enabled-client defaults');
    assert.match(receiptText, expected, 'Receipt records effective enabled-client defaults');
  }
});

journey('independent drafts and client-scoped review/apply preserve the other client', async page => {
  await editor(page, 'claude', 'Claude Code');
  await page.locator('#editor-enabled').check();
  await page.locator('#model').fill('claude-exact-id');
  await page.locator('#budget').selectOption('353000');
  await dashboard(page);
  await editor(page, 'codex', 'Codex');
  await page.locator('#budget').selectOption('272000');
  await page.locator('#tier').selectOption('Fast');
  await dashboard(page);
  await pending(page, 2);
  await editor(page, 'claude', 'Claude Code');
  await value(page, 'model', 'claude-exact-id');
  await value(page, 'budget', '353000');
  await command(page, 'review').click();
  assert.equal(await page.locator('.review-item').count(), 1);
  await text(page.locator('.review-item h3'), 'Claude Code');
  await text(page.locator('.diff'), /model.*gpt-6\.1-sol.*claude-exact-id.*budget.*500000.*353000.*enabled.*false.*true/);
  await command(page, 'back').click();
  await focus(page, '#editor-enabled');
  await command(page, 'review').click();
  await command(page, 'apply').click();
  await text(page.locator('.job'), /Claude Code.*model gpt-6\.1-sol → claude-exact-id.*budget 500000 → 353000.*enabled false → true.*PREVIEW APPLIED/);
  assert.doesNotMatch(await page.locator('.job').innerText(), /Codex/);
  await dashboard(page);
  await pending(page, 1);
  await detail(page, 'Model', 'claude-exact-id · Standard');
  await detail(page, 'Budget', '353000 tokens');
  await row(page, 'codex').click();
  await detail(page, 'Model', 'gpt-6.1-sol · Standard');
  await detail(page, 'Budget', '500000 tokens');
  await row(page, 'codex').press('Enter');
  await value(page, 'budget', '272000');
  await value(page, 'tier', 'Fast');
  await page.locator('[data-screen="review"]').click();
  await text(page.locator('.review-item h3'), 'Codex');
  await text(page.locator('.diff'), /budget.*500000.*272000.*tier.*Standard.*Fast/);
});

journey('tier capability differs from selected tier; model change resets unavailable acceleration', async page => {
  await detail(page, 'Model', 'gpt-6.1-sol · Standard');
  await detail(page, 'Tiers', 'Up to Ultrafast');
  await editor(page, 'codex', 'Codex');
  await page.locator('#tier').selectOption('Ultrafast');
  await value(page, 'model', 'gpt-6.1-sol');
  await command(page, 'review').click();
  await text(page.locator('.diff'), /tier.*Standard.*Ultrafast/);
  await command(page, 'apply').click();
  await text(page.locator('.job'), /tier Standard → Ultrafast/);
  await dashboard(page);
  await detail(page, 'Model', 'gpt-6.1-sol · Ultrafast');
  await detail(page, 'Tiers', 'Up to Ultrafast');
  await editor(page, 'codex', 'Codex');
  await page.locator('#model').fill('glm-5.3');
  await value(page, 'model', 'glm-5.3');
  await value(page, 'tier', 'Standard');
  assert.deepEqual(await page.locator('#tier option').allTextContents(), ['Standard']);
  await text(status(page), /tier reset to Standard/);
  await text(page.locator('#settings-form .selected-note'), /Accelerated tiers not verified/);
});

journey('invalid custom budgets block Apply; a corrected exact custom value applies', async page => {
  await editor(page, 'codex', 'Codex');
  await page.locator('#advanced summary').click();
  for (const invalid of ['0', '-1', '3.5', 'NaN', '9007199254740992', '']) {
    await page.locator('#custom-budget').fill(invalid);
    await text(page.locator('#form-error'), 'Context budget must be a positive whole number.');
    assert.equal(await page.locator('#custom-budget').getAttribute('aria-invalid'), 'true');
    await value(page, 'custom-budget', invalid);
  }
  await command(page, 'review').click();
  await text(page.locator('#review-error'), /positive whole number/);
  assert.equal(await command(page, 'apply').isDisabled(), true);
  await command(page, 'back').click();
  await focus(page, '#editor-enabled');
  await page.locator('#advanced summary').click();
  await page.locator('#custom-budget').fill('123456');
  await text(page.locator('#form-error'), '');
  await value(page, 'budget', '123456');
  await command(page, 'review').click();
  assert.equal(await command(page, 'apply').isDisabled(), false);
  await text(page.locator('.diff'), /budget.*500000.*123456/);
  await command(page, 'apply').click();
  await text(page.locator('.job'), /budget 500000 → 123456/);
  await dashboard(page);
  await detail(page, 'Budget', '123456 tokens');
});

journey('discard cancel/Escape preserve drafts; confirm discards only the named client', async page => {
  await editor(page, 'codex', 'Codex');
  await page.locator('#budget').selectOption('272000');
  await dashboard(page);
  await editor(page, 'claude', 'Claude Code');
  await page.locator('#editor-enabled').check();
  await page.locator('#model').fill('claude-draft');
  await command(page, 'discard').click();
  await text(dialog(page).locator('h2'), 'Discard Claude Code draft?');
  await dialog(page).getByRole('button', { name: 'Keep draft / Back', exact: true }).click();
  await focus(page, '[data-command="discard"]');
  await value(page, 'model', 'claude-draft');
  await command(page, 'discard').click();
  await escapeDialog(page, '[data-command="discard"]');
  await value(page, 'model', 'claude-draft');
  await command(page, 'discard').click();
  await dialog(page).getByRole('button', { name: 'Discard draft', exact: true }).click();
  await noDialog(page);
  await focus(page, '#editor-enabled');
  assert.equal(await page.locator('#editor-enabled').isChecked(), false);
  await value(page, 'model', 'gpt-6.1-sol');
  assert.equal(await command(page, 'discard').isDisabled(), true);
  await dashboard(page);
  await pending(page, 1);
  await editor(page, 'codex', 'Codex');
  await value(page, 'budget', '272000');
  await command(page, 'review').click();
  await text(page.locator('.review-item h3'), 'Codex');
});

journey('refresh, themes and layouts preserve drafts and refresh button focus', async page => {
  await editor(page, 'claude', 'Claude Code');
  await page.locator('#editor-enabled').check();
  await page.locator('#model').fill('preserved-custom-id');
  await page.locator('#budget').selectOption('353000');
  for (const theme of ['light', 'mono', 'dark']) {
    await page.locator('#theme').selectOption(theme);
    assert.equal(await page.locator('html').getAttribute('data-theme'), theme);
    for (const layout of ['compact', 'narrow', 'wide']) {
      await page.locator('#layout').selectOption(layout);
      assert.equal(await page.locator('#console').getAttribute('data-layout'), layout);
      await value(page, 'model', 'preserved-custom-id');
      await value(page, 'budget', '353000');
      assert.equal(await page.locator('#editor-enabled').isChecked(), true);
    }
  }
  await dashboard(page);
  await command(page, 'refresh').click();
  await focus(page, '[data-command="refresh"]');
  await text(status(page), /refreshed \(1\).*Drafts kept/);
  await pending(page, 1);
  await editor(page, 'claude', 'Claude Code');
  await value(page, 'model', 'preserved-custom-id');
  await value(page, 'budget', '353000');
  await page.locator('[data-screen="jobs"]').click();
  assert.equal(await page.locator('.job').count(), 0);
});

for (const [id, name, reason] of [['pi', 'Pi', /Not installed/], ['hermes', 'Hermes', /not supported/]]) {
  journey(`${name} remains inspectable with explicit disabled management/installation`, async page => {
    await row(page, id).click();
    assert.equal(await checkbox(page, id).isDisabled(), true);
    await text(page.locator('.detail .notice'), reason);
    if (id === 'pi') assert.equal(await command(page, 'install').isDisabled(), true);
    await row(page, id).press('Space');
    await pending(page, 0);
    await row(page, id).press('Enter');
    await text(heading(page), `${name} / settings`);
    for (const selector of ['#editor-enabled', '#model', '#budget', '#tier']) {
      assert.equal(await page.locator(selector).isDisabled(), true, `${name} ${selector} disabled`);
    }
    if (id === 'hermes') {
      await value(page, 'tier', 'Unknown / preserved');
      assert.doesNotMatch(await page.locator('.detail').innerText(), /Standard/);
    } else assert.equal(await command(page, 'install').isDisabled(), true);
    await command(page, 'palette').click();
    const toggle = dialog(page).locator('[data-palette-command="toggle"]');
    assert.equal(await toggle.getAttribute('aria-disabled'), 'true');
    await text(toggle, reason);
    assert.equal(await dialog(page).locator('[data-palette-command="install"]').getAttribute('aria-disabled'), 'true');
    await escapeDialog(page, '[data-command="palette"]');
    await page.locator('[data-screen="jobs"]').click();
    assert.equal(await page.locator('.job').count(), 0);
  });
}

journey('filter zero/reset and row arrow navigation retain correct client identity and drafts', async page => {
  await checkbox(page, 'claude').check();
  await page.locator('#client-filter').fill('Code');
  assert.equal(await page.locator('[data-client]').count(), 3);
  await row(page, 'claude').click();
  await row(page, 'claude').press('ArrowDown');
  await focus(page, '[data-client="opencode"]');
  assert.equal(await row(page, 'opencode').getAttribute('aria-pressed'), 'true');
  await detail(page, 'Budget', '353000 tokens');
  await row(page, 'opencode').press('ArrowUp');
  await focus(page, '[data-client="claude"]');
  await text(page.locator('.workarea > div:first-child > .selected-note'), 'Selected: Claude Code · 2 of 3 matching clients');
  await page.locator('#client-filter').fill('no-client-with-this-name');
  assert.equal(await page.locator('[data-client]').count(), 0);
  assert.equal(await page.locator('.detail').count(), 0);
  await text(page.locator('.client-list .empty'), /No matching clients/);
  await pending(page, 1);
  await command(page, 'clear-filter').click();
  await focus(page, '#client-filter');
  await value(page, 'client-filter', '');
  assert.equal(await page.locator('[data-client]').count(), 6);
  assert.equal(await row(page, 'claude').getAttribute('aria-pressed'), 'true');
  assert.equal(await checkbox(page, 'claude').isChecked(), true);
  await page.locator('#client-filter').fill('oPeNcOdE');
  assert.equal(await page.locator('[data-client]').count(), 1);
  assert.equal(await row(page, 'opencode').getAttribute('aria-pressed'), 'true');
  await text(page.locator('.workarea > div:first-child > .selected-note'), 'Selected: OpenCode · 1 of 1 matching clients');
  await row(page, 'opencode').press('Enter');
  await text(heading(page), 'OpenCode / settings');
});

journey('shortcuts are scoped; typing ?/r and Ctrl+P in fields retain normal input behavior', async page => {
  // Observe defaultPrevented at window bubbling time, after application handlers.
  // This deliberately does not prevent browser Print on the application's behalf.
  await page.evaluate(() => {
    window.__browserHarnessPrintKeys = [];
    window.addEventListener('keydown', event => {
      if (event.ctrlKey && event.key.toLowerCase() === 'p') {
        window.__browserHarnessPrintKeys.push({ prevented: event.defaultPrevented, target: event.target.id });
      }
    });
  });
  async function printKey(selector, prevented) {
    await page.locator(selector).focus();
    const count = await page.evaluate(() => window.__browserHarnessPrintKeys.length);
    await page.keyboard.press('Control+p');
    const observations = await page.evaluate(() => window.__browserHarnessPrintKeys);
    assert.equal(observations.length, count + 1, 'Ctrl+P reaches window observer');
    assert.equal(observations.at(-1).prevented, prevented, `${selector}: Ctrl+P interception`);
  }
  await page.locator('.wordmark').focus();
  await page.keyboard.press('?');
  await noDialog(page);
  await printKey('#theme', false);
  await noDialog(page);
  await heading(page).focus();
  await page.keyboard.press('/');
  await focus(page, '#client-filter');
  await page.keyboard.type('?/r');
  await value(page, 'client-filter', '?/r');
  await noDialog(page);
  await printKey('#client-filter', false);
  await command(page, 'clear-filter').click();
  await editor(page, 'codex', 'Codex');
  await page.locator('#model').fill('literal');
  await page.keyboard.type('?/r');
  await value(page, 'model', 'literal?/r');
  await noDialog(page);
  await printKey('#model', false);
  await page.locator('#advanced summary').click();
  for (const selector of ['#custom-budget', '#budget', '#tier']) await printKey(selector, false);
  await noDialog(page);
  await heading(page).focus();
  await page.keyboard.press('?');
  await text(dialog(page).locator('h2'), 'Keyboard shortcuts');
  await text(dialog(page), /Text fields keep normal typing/);
  await escapeDialog(page, '#screen h2');
  await printKey('#screen h2', true);
  await text(dialog(page).locator('h2'), 'Commands');
  await escapeDialog(page, '#screen h2');
  await page.keyboard.press('Escape');
  await text(heading(page), 'Your clients');
  await focus(page, '[data-client="codex"]');
  await row(page, 'codex').focus();
  await page.keyboard.press('r');
  await text(status(page), /refreshed \(1\)/);
  await focus(page, '[data-client="codex"]');
});

journey('palette searches, arrows, Enter, zero results and disabled commands are coherent', async page => {
  await command(page, 'palette').click();
  await focus(page, '#palette-search');
  const search = page.locator('#palette-search');
  const items = () => dialog(page).locator('[data-palette-command]');
  await search.fill('client');
  const ids = await items().evaluateAll(nodes => nodes.map(node => node.id));
  assert.ok(ids.length > 1);
  async function selected(id) {
    assert.equal(await search.getAttribute('aria-activedescendant'), id);
    assert.equal(await dialog(page).locator('[aria-selected="true"]').getAttribute('id'), id);
  }
  await selected(ids[0]);
  await search.press('ArrowDown');
  await selected(ids[1]);
  await search.press('ArrowUp');
  await selected(ids[0]);
  await search.press('ArrowUp');
  await selected(ids.at(-1));
  await search.press('ArrowDown');
  await selected(ids[0]);
  await search.fill('no-command-with-this-name');
  assert.equal(await items().count(), 0);
  assert.equal(await search.getAttribute('aria-activedescendant'), null);
  await text(page.locator('#palette-count'), /No matching commands/);
  await search.press('Enter');
  assert.equal(await dialog(page).count(), 1);
  await text(heading(page), 'Your clients');
  await search.fill('Review Codex draft');
  assert.equal(await items().count(), 1);
  assert.equal(await items().getAttribute('aria-disabled'), 'true');
  await text(items(), /No pending changes for this client/);
  await search.press('Enter');
  assert.equal(await dialog(page).count(), 1);
  // ARIA-disabled options are intentionally unclickable to Playwright's normal
  // actionability check. Force only this pointer attempt to verify the guard.
  await items().click({ force: true });
  assert.equal(await dialog(page).count(), 1);
  await pending(page, 0);
  await search.fill('Refresh preview snapshot');
  assert.equal(await items().count(), 1);
  await search.press('Enter');
  await noDialog(page);
  await text(status(page), /refreshed \(1\)/);
  await pending(page, 0);
});

journey('help and palette block background actions and Escape restores invokers', async page => {
  for (const id of ['help', 'palette']) {
    await command(page, id).click();
    const title = id === 'help' ? 'Keyboard shortcuts' : 'Commands';
    await text(dialog(page).locator('h2'), title);
    const before = await status(page).innerText();
    // A real pointer activation must be blocked by the native modal backdrop.
    await assert.rejects(page.locator('[data-screen="jobs"]').click({ timeout: 400 }), /Timeout/);
    if (id === 'help') {
      for (const key of ['r', '/', '?', 'Control+p']) await page.keyboard.press(key);
    } else {
      await page.locator('#palette-search').fill('');
      await page.keyboard.type('?/r');
      await value(page, 'palette-search', '?/r');
    }
    await text(heading(page), 'Your clients');
    await text(dialog(page).locator('h2'), title);
    assert.equal(await status(page).innerText(), before);
    await escapeDialog(page, `[data-command="${id}"]`);
  }
});

for (const id of ['help', 'palette', 'discard']) {
  journey(`${id} modal traps Tab and Shift+Tab`, async page => {
    if (id === 'discard') {
      await editor(page, 'claude', 'Claude Code');
      await page.locator('#editor-enabled').check();
    }
    await command(page, id).click();
    await trapTabs(page);
    await escapeDialog(page, `[data-command="${id}"]`);
  });
}

for (const width of [1440, 620, 390]) {
  for (const theme of ['dark', 'light', 'mono']) {
    journey(`overflow and controls: ${width}px ${theme}, every layout and screen`, async page => {
      await page.locator('#theme').selectOption(theme);
      for (const [index, layout] of ['wide', 'compact', 'narrow'].entries()) {
        await page.locator('#layout').selectOption(layout);
        const label = `${width}px ${theme} ${layout}`;
        await dashboard(page);
        await noOverflow(page, `${label} dashboard`);
        if (layout === 'narrow') {
          await command(page, 'next').click();
          assert.equal(await row(page, 'claude').isVisible(), true);
          await command(page, 'previous').click();
          assert.equal(await row(page, 'codex').isVisible(), true);
        }
        await page.locator('[data-screen="models"]').click();
        await page.locator('#budget').selectOption('353000');
        await noOverflow(page, `${label} settings`);
        await command(page, 'review').click();
        await noOverflow(page, `${label} review`);
        await command(page, 'apply').click();
        await noOverflow(page, `${label} receipt`);
        assert.equal(await page.locator('.job').count(), index * 2 + 1);
        for (const id of ['help', 'palette']) {
          await command(page, id).click();
          await noOverflow(page, `${label} ${id}`);
          await escapeDialog(page, `[data-command="${id}"]`);
        }
        // Return the applied budget so each layout exercises a real review/apply.
        await page.locator('[data-screen="models"]').click();
        await page.locator('#budget').selectOption('500000');
        await command(page, 'review').click();
        await command(page, 'apply').click();
        // Receipt counts accumulate; check the latest outcome, not a fixed layout.
        await text(page.locator('.job').first(), /budget 353000 → 500000.*PREVIEW APPLIED/);
        await screenshot(page, `${width}-${theme}-${layout}`);
      }
    }, { viewport: { width, height: 1000 } });
  }
}

async function main() {
  const { chromium } = await import(process.env.PLAYWRIGHT_MODULE || 'playwright');
  const browser = await chromium.launch({ headless: true });
  let passed = 0, failed = 0;
  const started = Date.now();
  try {
    for (const test of cases) {
      const context = await browser.newContext({ viewport: test.viewport || { width: 1440, height: 1000 }, serviceWorkers: 'block' });
      const page = await context.newPage();
      page.setDefaultTimeout(3000);
      page.setDefaultNavigationTimeout(10000);
      const browserIssues = [];
      page.on('pageerror', error => browserIssues.push(`pageerror: ${error.message}`));
      page.on('console', message => { if (message.type() === 'error') browserIssues.push(`console.error: ${message.text()}`); });
      page.on('requestfailed', request => browserIssues.push(`request failed: ${request.url()} (${request.failure()?.errorText})`));
      page.on('response', response => { if (!response.ok()) browserIssues.push(`HTTP ${response.status()}: ${response.url()}`); });
      await context.route('**/*', async route => {
        const request = route.request();
        if (new URL(request.url()).origin !== origin) {
          browserIssues.push(`External request blocked: ${request.method()} ${request.url()}`);
          await route.abort('blockedbyclient');
        } else await route.continue();
      });
      // Fixed per-journey deadline also bounds loops and waits after regressions.
      let timer;
      let failure;
      try {
        await Promise.race([
          (async () => {
            const response = await page.goto(baseURL, { waitUntil: 'networkidle' });
            assert.equal(response?.status(), 200, 'Preview document loads');
            await heading(page).waitFor();
            await text(heading(page), 'Your clients');
            await test.run(page);
          })(),
          new Promise((_, reject) => { timer = setTimeout(() => reject(new Error('Journey exceeded 30 seconds')), 30000); })
        ]);
      } catch (error) {
        failure = error;
        try { await screenshot(page, `FAIL-${test.name}`); } catch (error) { browserIssues.push(`Screenshot: ${error.message}`); }
      } finally {
        clearTimeout(timer);
        // Closing the context cancels any outstanding action after a deadline.
        await context.close();
      }
      if (browserIssues.length || failure) {
        failed++;
        console.error(`FAIL ${test.name}`);
        if (failure) console.error(failure.stack || String(failure));
        for (const issue of browserIssues) console.error(`  ${issue}`);
      } else {
        passed++;
        console.log(`PASS ${test.name}`);
      }
    }
  } finally {
    await browser.close();
  }
  console.log(`\nBrowser journeys: ${passed} passed, ${failed} failed, ${cases.length} total (${((Date.now() - started) / 1000).toFixed(1)}s).`);
  if (failed) process.exitCode = 1;
}

main().catch(error => {
  console.error(error.stack || String(error));
  console.error('Use PLAYWRIGHT_MODULE to point to an installed Playwright module and TUI_PREVIEW_URL to an already served preview.');
  process.exitCode = 1;
});
