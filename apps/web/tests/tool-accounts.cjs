// Uses the development-only in-memory adapter, never the production API.
const assert = require('node:assert/strict');
const { chromium } = require('playwright');
const path = require('node:path');
const fs = require('node:fs/promises');
async function run() {
  const browser = await chromium.launch({ channel: 'msedge', headless: true });
  const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } });
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  await page.route('**/api/v1/**', route => route.request().url().endsWith('/auth/csrf')
    ? route.fulfill({ json: { code: 200, data: { csrf_token: 'preview-only' } } }) : route.abort());
  const preview = (route, scenario = '') => page.goto(`http://localhost:5176/tests/design-preview.html?${new URLSearchParams({ route, scenario })}`);
  const modal = () => page.getByRole('dialog').last();
  const choose = async (label, option) => {
    await modal().getByRole('combobox', { name: label, exact: true }).click();
    await page.getByRole('option', { name: option, exact: true }).click();
  };
  const fill = async name => {
    await modal().getByPlaceholder('例如：宝塔登录、运营主账号').fill(name);
    await modal().getByPlaceholder('账号或邮箱').fill('private-tool-user');
  };
  try {
    await preview('/public-links');
    await page.locator('.tool-table').getByRole('link', { name: '我的账号', exact: true }).first().click();
    await page.getByRole('heading', { name: 'GitLab · 我的账号', exact: true }).waitFor();
    assert.equal(await page.locator('.account-card').count(), 0);
    await page.getByRole('button', { name: '添加账号', exact: true }).click();
    await fill('GitLab 个人账号');
    await choose('保存位置', '服务器同步');
    assert.equal(await modal().getByLabel('所属工具', { exact: true }).inputValue(), 'GitLab');
    await choose('密码处理', '保存密码');
    await modal().getByPlaceholder('请输入要保存的密码').fill('fixture-only-tool-password');
    await modal().getByPlaceholder('仅本人可见的账号说明').fill('fixture-only-tool-note');
    await modal().getByRole('button', { name: '保存账号', exact: true }).click();
    await page.getByText('用途：GitLab 个人账号', { exact: true }).waitFor();
    const card = page.locator('.account-card').filter({ hasText: 'GitLab 个人账号' });
    assert.equal(await card.getByRole('heading').innerText(), 'GitLab');
    await card.getByRole('button', { name: '显示GitLab 个人账号密码', exact: true }).click();
    await card.getByText('fixture-only-tool-password', { exact: true }).waitFor();
    await card.getByRole('button', { name: '编辑', exact: true }).click();
    await modal().getByPlaceholder('仅本人可见的账号说明').waitFor();
    assert.equal(await modal().getByPlaceholder('仅本人可见的账号说明').inputValue(), 'fixture-only-tool-note');
    await modal().getByPlaceholder('仅本人可见的账号说明').fill('updated private note');
    await modal().getByRole('button', { name: '保存账号', exact: true }).click();
    await page.locator('[data-slot="dialog-content"][data-state="open"]').waitFor({ state: 'hidden' });
    await card.getByRole('button', { name: '删除', exact: true }).click();
    await modal().getByRole('button', { name: '移入回收站', exact: true }).click();
    await page.locator('[data-slot="dialog-content"][data-state="open"]').waitFor({ state: 'hidden' });
    await page.getByRole('button', { name: '回收站', exact: true }).click();
    await page.getByRole('button', { name: '恢复账号', exact: true }).click();
    await page.getByRole('button', { name: '返回账号', exact: true }).click();
    await page.getByText('用途：GitLab 个人账号', { exact: true }).waitFor();
    await page.getByRole('link', { name: '全部账号', exact: true }).click();
    await page.getByText('用途：测试后台账号', { exact: true }).waitFor();
    await page.getByText('用途：GitLab 个人账号', { exact: true }).waitFor();
    await page.getByRole('button', { name: '添加账号', exact: true }).click();
    await fill('监控个人账号');
    await choose('关联类型', '工具');
    await choose('选择所属工具', '监控平台');
    await modal().getByRole('button', { name: '保存账号', exact: true }).click();
    await page.getByText('用途：监控个人账号', { exact: true }).waitFor();
    const output = path.resolve(__dirname, '../../../.gocache/shadcn-verification');
    await fs.mkdir(output, { recursive: true });
    await page.screenshot({ path: path.join(output, 'tool-account-aggregate.png'), fullPage: true });
    await page.setViewportSize({ width: 390, height: 844 });
    await page.getByRole('button', { name: '添加账号', exact: true }).click();
    await fill('手机工具账号');
    await choose('关联类型', '工具');
    await choose('选择所属工具', 'GitLab');
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
    await modal().getByRole('button', { name: '保存账号', exact: true }).scrollIntoViewIfNeeded();
    await page.screenshot({ path: path.join(output, 'tool-account-mobile.png') });
    await modal().getByRole('button', { name: '保存账号', exact: true }).click();
    await page.getByText('用途：手机工具账号', { exact: true }).waitFor();
    await preview('/my-accounts?toolId=301', 'save-error');
    await page.getByRole('button', { name: '添加账号', exact: true }).click();
    await fill('失败后保留');
    await choose('保存位置', '服务器同步');
    await modal().getByRole('button', { name: '保存账号', exact: true }).click();
    await modal().getByRole('alert').waitFor();
    assert.equal(await modal().getByPlaceholder('例如：宝塔登录、运营主账号').inputValue(), '失败后保留');
    assert(await modal().getByRole('button', { name: '保存账号', exact: true }).isEnabled());
    assert.deepEqual(errors, []);
    console.log('PASS: tool entry, scoped list, create, reveal, edit, delete/restore, aggregate project+tool list, generic tool selection, narrow viewport and save failure recovery');
  } finally { await browser.close(); }
}
run().catch(error => { console.error(error); process.exitCode = 1; });
