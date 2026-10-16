// Run with Playwright available (locally or through NODE_PATH).
// Uses only the development preview adapter; no production data is written.
const assert = require('node:assert/strict');
const fs = require('node:fs/promises');
const path = require('node:path');
const { chromium } = require('playwright');

const origin = process.env.ATLAS_PREVIEW_ORIGIN || 'http://localhost:5176';
const output = path.resolve(__dirname, '../../../.gocache/shadcn-verification');
const results = [];

async function run() {
  await fs.mkdir(output, { recursive: true });
  const browser = await chromium.launch({ channel: 'msedge', headless: true });
  const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } });
  await page.route('**/api/v1/**', route => route.request().url().endsWith('/auth/csrf')
    ? route.fulfill({ json: { code: 200, data: { csrf_token: 'preview-only' } } })
    : route.abort());
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  const openRoute = async (route, scenario = '') => {
    await page.goto(`${origin}/tests/design-preview.html?${new URLSearchParams({ route, scenario })}`);
    await page.locator('main').waitFor();
  };
  const dialog = () => page.getByRole('dialog').last();
  const closeWithEscape = async () => {
    await page.keyboard.press('Escape');
    await page.locator('[data-slot="dialog-content"][data-state="open"]').waitFor({ state: 'hidden' });
  };
  const assertFits = async () => {
    const bounds = await dialog().evaluate(el => {
      const r = el.getBoundingClientRect();
      const style = getComputedStyle(el);
      return { left: r.left, right: r.right, top: r.top, bottom: r.bottom,
        overflow: el.scrollWidth - el.clientWidth, viewport: innerWidth,
        height: innerHeight, radius: style.borderRadius };
    });
    assert(bounds.left >= 0 && bounds.right <= bounds.viewport + 1, JSON.stringify(bounds));
    assert(bounds.top >= 0 && bounds.bottom <= bounds.height + 1, JSON.stringify(bounds));
    assert(bounds.overflow <= 1, JSON.stringify(bounds));
    assert.equal(bounds.radius, '8px');
  };
  try {
    await openRoute('/projects');
    const create = page.getByRole('button', { name: '新建项目', exact: true });
    await create.click();
    await dialog().waitFor();
    await assertFits();
    assert.equal(await dialog().locator('input').first().evaluate(el => getComputedStyle(el).height), '36px');
    await dialog().locator('input').first().focus();
    assert.notEqual(await dialog().locator('input').first().evaluate(el => getComputedStyle(el).boxShadow), 'none');
    await dialog().evaluate(el => Promise.all(el.getAnimations().map(animation => animation.finished)));
    await page.screenshot({ path: path.join(output, 'project-dialog-desktop.png') });
    // Tab stays inside the modal even when the form is longer than the viewport.
    for (let i = 0; i < 30; i++) {
      await page.keyboard.press('Tab');
      assert(await dialog().evaluate(el => el.contains(document.activeElement)));
    }
    await dialog().getByRole('combobox').first().click();
    await page.getByRole('listbox').waitFor();
    await page.keyboard.press('ArrowDown');
    await page.keyboard.press('Escape');
    await page.getByRole('listbox').waitFor({ state: 'hidden' });
    assert(await dialog().isVisible());
    await closeWithEscape();
    assert(await create.evaluate(el => el === document.activeElement));
    await create.click();
    await page.mouse.click(5, 5);
    await page.locator('[data-slot="dialog-content"][data-state="open"]').waitFor({ state: 'hidden' });
    results.push('Project dialog: default geometry, Tab trap, nested Select Escape, dialog Escape, backdrop dismissal, focus restoration');

    await openRoute('/public-links');
    await page.getByRole('button', { name: '新增工具', exact: true }).click();
    const toggle = dialog().getByRole('switch', { name: '启用工具入口' });
    await toggle.waitFor();
    assert.equal(await toggle.getAttribute('aria-checked'), 'true');
    await toggle.click();
    assert.equal(await toggle.getAttribute('aria-checked'), 'false');
    await page.keyboard.press('Space');
    assert.equal(await toggle.getAttribute('aria-checked'), 'true');
    await closeWithEscape();
    results.push('Tool Switch: initial value, pointer and keyboard toggle use current Reka v-model');

    await openRoute('/projects/101');
    const tabs = page.getByRole('tab');
    await tabs.first().focus();
    await page.keyboard.press('ArrowRight');
    assert.equal(await tabs.nth(1).getAttribute('aria-selected'), 'true');
    await page.keyboard.press('ArrowLeft');
    assert.equal(await tabs.first().getAttribute('aria-selected'), 'true');
    results.push('Project Tabs: default automatic Arrow-key selection');

    await openRoute('/my-accounts');
    await page.getByRole('button', { name: '添加账号', exact: true }).click();
    await dialog().getByPlaceholder('例如：宝塔登录、运营主账号').fill('未保存的示例账号');
    await page.mouse.click(5, 5);
    await page.getByRole('heading', { name: '放弃未保存的修改？' }).waitFor();
    await page.getByRole('button', { name: '继续编辑', exact: true }).click();
    assert.equal(await dialog().getByPlaceholder('例如：宝塔登录、运营主账号').inputValue(), '未保存的示例账号');
    await page.keyboard.press('Escape');
    await page.getByRole('button', { name: '放弃修改', exact: true }).click();
    await page.locator('[data-slot="dialog-content"][data-state="open"]').waitFor({ state: 'hidden' });
    results.push('Account dialog: backdrop and Escape both request closure; discard confirmation preserves or discards the draft');

    await openRoute('/my-accounts', 'project-switch');
    await page.getByRole('button', { name: '添加账号', exact: true }).click();
    const draftName = dialog().getByPlaceholder('例如：宝塔登录、运营主账号');
    await draftName.fill('切换项目时保留的账号');
    await draftName.evaluate(el => { el.dataset.identity = 'persistent-field'; });
    const projectPicker = dialog().getByRole('combobox', { name: '选择所属项目' });
    await projectPicker.scrollIntoViewIfNeeded();
    const initialBounds = await dialog().boundingBox();
    const chooseProject = async name => {
      await projectPicker.click();
      await page.getByRole('option', { name, exact: true }).click();
    };
    await chooseProject('Nexus 多语言平台');
    await dialog().getByRole('status').filter({ hasText: '正在加载项目入口' }).waitFor();
    const loadingBounds = await dialog().boundingBox();
    assert(Math.abs(initialBounds.height - loadingBounds.height) <= 1);
    assert(Math.abs(initialBounds.y - loadingBounds.y) <= 1);
    assert.equal(await draftName.getAttribute('data-identity'), 'persistent-field');
    assert.equal(await draftName.inputValue(), '切换项目时保留的账号');
    assert(await projectPicker.evaluate(el => el === document.activeElement));
    const resources = dialog().getByRole('combobox', { name: '关联资源入口（选填）' });
    assert(await resources.isDisabled());
    assert(await dialog().getByRole('button', { name: '保存账号', exact: true }).isDisabled());
    await chooseProject('Helios 认证中心');
    await page.waitForTimeout(1700); // The superseded Nexus response must not replace Helios.
    assert(await resources.isEnabled());
    await resources.click();
    await page.getByRole('option', { name: /API 文档/ }).waitFor();
    assert.equal(await page.getByRole('option', { name: /GitLab/ }).count(), 0);
    await page.keyboard.press('Escape');
    await chooseProject('云手机用户端');
    await dialog().getByRole('alert').waitFor();
    assert.equal(await draftName.getAttribute('data-identity'), 'persistent-field');
    assert.equal(await draftName.inputValue(), '切换项目时保留的账号');
    assert(await resources.isDisabled());
    await chooseProject('Helios 认证中心');
    await resources.waitFor();
    await page.waitForFunction(() => document.querySelector('[aria-labelledby="account-resource-label"]')?.getAttribute('aria-busy') === 'false');
    assert(await dialog().getByRole('button', { name: '保存账号', exact: true }).isEnabled());
    await page.keyboard.press('Escape');
    await page.getByRole('button', { name: '放弃修改', exact: true }).click();
    await page.locator('[data-slot="dialog-content"][data-state="open"]').waitFor({ state: 'hidden' });
    results.push('Account project switching: stable modal geometry and field nodes, preserved draft/focus, local busy state, stale response ignored, failure recovery');

    await page.setViewportSize({ width: 390, height: 844 });
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await openRoute('/projects');
    await page.getByRole('button', { name: '新建项目', exact: true }).click();
    await assertFits();
    await dialog().getByRole('button', { name: '保存项目', exact: true }).scrollIntoViewIfNeeded();
    await page.screenshot({ path: path.join(output, 'project-dialog-mobile.png') });
    await closeWithEscape();
    results.push('390px viewport and reduced motion: long form stays in viewport, scrolls to Save and closes');

    await page.setViewportSize({ width: 1440, height: 1000 });
    await openRoute('/my-accounts', 'save-error');
    await page.getByRole('button', { name: '编辑', exact: true }).first().click();
    const accountName = dialog().getByPlaceholder('例如：宝塔登录、运营主账号');
    await accountName.fill('保存失败后保留的示例账号');
    await dialog().getByRole('button', { name: '保存账号', exact: true }).click();
    await dialog().getByRole('alert').filter({ hasText: /保存失败|网络连接失败/ }).waitFor();
    assert.equal(await accountName.inputValue(), '保存失败后保留的示例账号');
    assert(await dialog().getByRole('button', { name: '保存账号', exact: true }).isEnabled());
    results.push('Account save failure: modal remains open, draft preserved and retry enabled');
    assert.deepEqual(errors, []);
    await fs.writeFile(path.join(output, 'results.json'), JSON.stringify({ results, errors }, null, 2));
    console.log(JSON.stringify({ passed: results.length, results, errors }, null, 2));
  } finally {
    await browser.close();
  }
}
run().catch(error => { console.error(error); process.exitCode = 1; });
