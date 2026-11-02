import assert from 'node:assert/strict';

export function registerIMEJourneys(journey) {
  async function compose(page, selector, expected) {
    const input = page.locator(selector);
    await input.fill(''); await input.focus();
    const original = await input.elementHandle();
    const cdp = await page.context().newCDPSession(page);
    await cdp.send('Input.imeSetComposition', { text: 'に', selectionStart: 1, selectionEnd: 1 });
    assert.equal(await original.evaluate(el => el.isConnected), true, 'Composition preserves its original input node');
    await cdp.send('Input.imeSetComposition', { text: expected, selectionStart: expected.length, selectionEnd: expected.length });
    await cdp.send('Input.insertText', { text: expected });
    await page.waitForFunction(({ selector, expected }) => document.querySelector(selector)?.value === expected, { selector, expected });
    // Await the deferred commit/render before inspecting draft-dependent controls.
    await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
    assert.equal(await input.inputValue(), expected);
    assert.equal(await input.evaluate(el => document.activeElement === el), true);
    await cdp.detach();
  }
  journey('IME model and filter composition preserve exact committed text and draft identity', async page => {
    await page.locator('[data-client="codex"]').press('Enter');
    await compose(page, '#model', '日本');
    await page.locator('[data-command="review"]').click();
    assert.match(await page.locator('.review-item').innerText(), /日本/);
    await page.locator('[data-command="apply"]').click();
    assert.match(await page.locator('.job').innerText(), /日本/);
    await page.locator('[data-command="dashboard"]').click();
    assert.match(await page.locator('[data-client="codex"]').innerText(), /日本/);
    await compose(page, '#client-filter', '日本');
    assert.equal(await page.locator('[data-client]').count(), 0);
    await page.locator('[data-command="clear-filter"]').click();
    assert.equal(await page.locator('[data-client]').count(), 6);
  });
  journey('IME palette candidate keys cannot navigate or stage management', async page => {
    for (const tracked of [false, true]) {
      await page.locator('[data-command="palette"]').click();
      const input = page.locator('#palette-search');
      await input.fill('Disable Codex management');
      const selected = await input.getAttribute('aria-activedescendant');
      const prevented = await input.evaluate((el, tracked) => {
        if (tracked) el.dispatchEvent(new CompositionEvent('compositionstart', { bubbles: true }));
        return ['ArrowDown', 'ArrowUp', 'Enter'].map(key => !el.dispatchEvent(new KeyboardEvent('keydown', {
          key, code: key, isComposing: !tracked, bubbles: true, cancelable: true
        })));
      }, tracked);
      assert.deepEqual(prevented, [false, false, false], 'Candidate keys retain native behavior');
      assert.equal(await page.locator('dialog[open]').count(), 1);
      assert.equal(await input.getAttribute('aria-activedescendant'), selected);
      assert.equal(await page.locator('[data-enable="codex"]').isChecked(), true);
      if (tracked) {
        await input.evaluate(el => el.dispatchEvent(new CompositionEvent('compositionend', { bubbles: true })));
        await page.evaluate(() => new Promise(resolve => requestAnimationFrame(resolve)));
      }
      if (tracked) {
        await input.press('Enter');
        assert.equal(await page.locator('dialog[open]').count(), 0);
        assert.equal(await page.locator('[data-enable="codex"]').isChecked(), false);
      } else await input.press('Escape');
    }
  });
}
