const assert = require('node:assert/strict');
const { chromium } = require('playwright');
const path = require('node:path');
const fs = require('node:fs/promises');

async function run() {
  const browser = await chromium.launch({ channel: 'msedge', headless: true });
  const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
  const page = await context.newPage();
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  await context.route('**/api/v1/**', route => route.request().url().endsWith('/auth/csrf')
    ? route.fulfill({ json: { code: 200, data: { csrf_token: 'preview-only' } } }) : route.abort());
  const visit = (target, scenario = '') => target.goto(`http://localhost:5176/tests/design-preview.html?${new URLSearchParams({ route: '/my-accounts', scenario })}`);
  const modal = target => target.getByRole('dialog', { name: /^(添加|编辑)账号$/ });
  const choose = async (target, label, option) => {
    await modal(target).getByRole('combobox', { name: label, exact: true }).click();
    await target.getByRole('option', { name: option, exact: true }).click();
  };
  const card = target => target.locator('article').filter({ has: target.getByText('用途：本地测试账号', { exact: true }) });
  const fill = async target => {
    await target.getByRole('button', { name: '添加账号', exact: true }).click();
    await modal(target).getByPlaceholder('例如：宝塔登录、运营主账号').fill('本地测试账号');
    await choose(target, '选择所属项目', 'Nexus 多语言平台');
    await modal(target).getByPlaceholder('账号或邮箱').fill('local-only-username');
    await choose(target, '密码处理', '保存密码');
    await modal(target).getByPlaceholder('请输入要保存的密码').fill('local-only-password');
    await modal(target).getByPlaceholder('仅本人可见的账号说明').fill('local-only-note');
  };
  const noLeak = async target => {
    const safe = await target.evaluate(() => {
      const requests = window.__atlasPreviewRequests;
      const mutations = requests.filter(r => r.url?.startsWith('/my/accounts') && r.method !== 'get');
      return mutations.length === 0 && !['local-only-username', 'local-only-password', 'local-only-note'].some(s => JSON.stringify(requests).includes(s));
    });
    assert(safe, 'local credentials must never enter an API request');
  };
  try {
    await visit(page);
    await page.getByText('用途：测试后台账号', { exact: true }).waitFor();
    await fill(page);
    assert.equal(await modal(page).getByRole('combobox', { name: '保存位置', exact: true }).innerText(), '仅此浏览器');
    await modal(page).getByText(/本地数据不加密/).waitFor();
    await modal(page).getByRole('button', { name: '保存账号', exact: true }).click();
    await modal(page).waitFor({ state: 'hidden' });
    await card(page).waitFor();

    assert.match(await card(page).innerText(), /本地/);
    await card(page).getByRole('button', { name: '显示本地测试账号密码', exact: true }).click();
    await card(page).getByText('local-only-password', { exact: true }).waitFor();
    await noLeak(page);
    const beforeLocalFilter = await page.evaluate(() => window.__atlasPreviewRequests.filter(r => r.url === '/my/accounts').length);
    await page.getByRole('combobox', { name: '筛选保存位置', exact: true }).click();
    await page.getByRole('option', { name: '仅此浏览器', exact: true }).click();
    await page.getByText('用途：测试后台账号', { exact: true }).waitFor({ state: 'hidden' });
    assert.equal(await page.evaluate(() => window.__atlasPreviewRequests.filter(r => r.url === '/my/accounts').length), beforeLocalFilter);
    await page.reload();
    await card(page).waitFor();
    await card(page).getByRole('button', { name: '编辑', exact: true }).click();
    assert.equal(await modal(page).getByPlaceholder('仅本人可见的账号说明').inputValue(), 'local-only-note');
    assert(await modal(page).getByRole('combobox', { name: '保存位置', exact: true }).isDisabled());
    await modal(page).getByPlaceholder('账号或邮箱').fill('local-only-username');
    await modal(page).getByRole('button', { name: '保存账号', exact: true }).click();
    await modal(page).waitFor({ state: 'hidden' });
    await card(page).waitFor();
    await card(page).getByRole('button', { name: '显示本地测试账号密码', exact: true }).click();
    await card(page).getByText('local-only-password', { exact: true }).waitFor();

    // Isolation and logout remove secrets from the rendered state, while retaining the owner's local records.
    await page.evaluate(async () => (await import('/src/store/auth.ts')).useAuthStore().setUser({ id: 'another-preview-user' }));
    await card(page).waitFor({ state: 'hidden' });
    const isolated = await page.evaluate(async () => {
      const { listLocalAccounts } = await import('/src/utils/local-accounts.ts');
      return (await listLocalAccounts('another-preview-user')).length === 0;
    });
    assert(isolated);
    await page.evaluate(async () => (await import('/src/store/auth.ts')).useAuthStore().setUser({ id: 'ui-preview' }));
    await card(page).waitFor();

    // Two tabs must not silently overwrite each other's password or notes.
    const second = await context.newPage();
    await visit(second);
    await card(second).getByRole('button', { name: '编辑', exact: true }).click();
    await modal(second).getByPlaceholder('仅本人可见的账号说明').fill('stale draft');
    await card(page).getByRole('button', { name: '编辑', exact: true }).click();
    await modal(page).getByPlaceholder('仅本人可见的账号说明').fill('local-only-note');
    await modal(page).getByRole('button', { name: '保存账号', exact: true }).click();
    await modal(page).waitFor({ state: 'hidden' });
    await card(page).waitFor();
    await modal(second).getByRole('button', { name: '保存账号', exact: true }).click();
    await modal(second).getByRole('alert').filter({ hasText: '账号已更新' }).waitFor();
    assert.equal(await modal(second).getByPlaceholder('仅本人可见的账号说明').inputValue(), 'stale draft');
    await second.close();

    await card(page).getByRole('button', { name: '删除', exact: true }).click();
    await page.getByRole('dialog', { name: '删除账号', exact: true }).getByRole('button', { name: '移入回收站', exact: true }).click();
    await card(page).waitFor({ state: 'hidden' });
    await page.getByRole('button', { name: '回收站', exact: true }).click();
    await card(page).getByRole('button', { name: '恢复账号', exact: true }).click();
    await card(page).waitFor({ state: 'hidden' });
    await page.getByRole('button', { name: '返回账号', exact: true }).click();
    await card(page).waitFor();
    await page.getByRole('textbox', { name: '搜索我的账号', exact: true }).fill('local-only-username');
    await page.waitForTimeout(500);
    assert.equal(await page.locator('article').count(), 1);
    await noLeak(page);

    await visit(page, 'error');
    await card(page).waitFor();
    await page.getByText(/服务器账号加载失败/).waitFor();
    await noLeak(page);
    await visit(page);
    await card(page).waitFor();
    await page.setViewportSize({ width: 390, height: 844 });
    await card(page).getByRole('button', { name: '编辑', exact: true }).click();
    await modal(page).getByPlaceholder('账号或邮箱').waitFor();
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
    const output = path.resolve(__dirname, '../../../.gocache/shadcn-verification');
    await fs.mkdir(output, { recursive: true });
    await page.screenshot({ path: path.join(output, 'local-accounts-mobile.png') });
    await page.setViewportSize({ width: 1440, height: 1000 });
    await page.waitForFunction(() => {
      const dialog = document.querySelector('[data-slot="dialog-content"]');
      return dialog && getComputedStyle(dialog).maxWidth === '512px' && dialog.getBoundingClientRect().width <= 513;
    });
    await page.waitForTimeout(250);
    await page.screenshot({ path: path.join(output, 'local-accounts-desktop.png') });
    await noLeak(page);

    // A blocked IndexedDB must preserve inputs and never fall back to uploading.
    const blocked = await browser.newContext();
    await blocked.addInitScript(() => Object.defineProperty(window, 'indexedDB', { value: undefined }));
    const blockedPage = await blocked.newPage();
    await visit(blockedPage);
    await fill(blockedPage);
    await modal(blockedPage).getByRole('button', { name: '保存账号', exact: true }).click();
    await modal(blockedPage).getByRole('alert').filter({ hasText: '账号未上传服务器' }).waitFor();
    assert.equal(await modal(blockedPage).getByPlaceholder('请输入要保存的密码').inputValue(), 'local-only-password');
    await noLeak(blockedPage);
    await blocked.close();
    // Both destinations accept a blank purpose; the owner remains the visible title.
    await visit(page);
    for (const storage of ['local', 'server']) {
      await fill(page);
      await modal(page).getByPlaceholder('例如：宝塔登录、运营主账号').fill('');
      await modal(page).getByPlaceholder('账号或邮箱').fill(`optional-purpose-${storage}`);
      if (storage === 'server') await choose(page, '保存位置', '服务器同步');
      await modal(page).getByRole('button', { name: '保存账号', exact: true }).click();
    await modal(page).waitFor({ state: 'hidden' });
      const optional = page.locator('.account-card').filter({ has: page.getByText(`optional-purpose-${storage}`, { exact: true }) });
      await optional.waitFor();
      assert.equal(await optional.getByRole('heading').innerText(), 'Nexus 多语言平台');
      assert.equal(await optional.locator('p').filter({ hasText: /^用途：/ }).count(), 0);
      await optional.getByRole('button', { name: '编辑', exact: true }).click();
      assert.equal(await modal(page).getByPlaceholder('例如：宝塔登录、运营主账号').inputValue(), '');
      await modal(page).getByRole('button', { name: '取消', exact: true }).click();
      await modal(page).waitFor({ state: 'hidden' });
    }
    assert.deepEqual(errors, []);
    console.log('PASS: local CRUD, refresh persistence, zero credential requests, owner isolation, multi-tab conflict, partial server failure, blocked storage preserves draft, desktop/mobile, optional purpose for local and server with owner title');
  } finally { await browser.close(); }
}
run().catch(error => { console.error(error); process.exitCode = 1; });
