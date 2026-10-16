const assert = require('node:assert/strict');
const { chromium } = require('playwright');

async function run() {
  const browser = await chromium.launch({ channel: 'msedge', headless: true });
  const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } });
  const origin = process.env.ATLAS_PREVIEW_ORIGIN || 'http://127.0.0.1:5177';
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  await page.route('**/api/v1/**', route => route.request().url().endsWith('/auth/csrf')
    ? route.fulfill({ json: { code: 200, data: { csrf_token: 'preview-only' } } }) : route.abort());
  const dialog = () => page.getByRole('dialog').last();
  async function fillProject() {
    await page.getByRole('button', { name: '新建项目', exact: true }).click();
    await dialog().getByLabel('项目名称', { exact: true }).fill('双发布系统回归');
    // Reuse the first two rows with the same deploy type; the empty third row is omitted.
    for (const [index, title, url] of [
      [0, '测试发布系统', 'https://deploy-test.example.com'],
      [1, '线上发布系统', 'https://deploy.example.com'],
    ]) {
      const row = dialog().locator('.entry-form-row').nth(index);
      await row.getByRole('combobox').click();
      await page.getByRole('option', { name: '发布系统', exact: true }).click();
      await row.getByLabel('入口名称（选填）', { exact: true }).fill(title);
      await row.getByLabel('URL', { exact: true }).fill(url);
    }
  }
  try {
    await page.goto(`${origin}/tests/design-preview.html?route=/projects`);
    await fillProject();
    await dialog().getByRole('button', { name: '保存项目', exact: true }).click();
    await page.getByRole('dialog').waitFor({ state: 'hidden' });
    const card = page.locator('.project-card').filter({ hasText: '双发布系统回归' });
    assert.equal(await card.getByRole('link', { name: '测试发布系统', exact: true }).getAttribute('href'), 'https://deploy-test.example.com');
    assert.equal(await card.getByRole('link', { name: '线上发布系统', exact: true }).getAttribute('href'), 'https://deploy.example.com');
    await page.getByRole('button', { name: '表格视图', exact: true }).click();
    const tableRow = page.getByRole('row').filter({ hasText: '双发布系统回归' });
    assert.equal(await tableRow.getByRole('link', { name: '测试发布系统', exact: true }).getAttribute('href'), 'https://deploy-test.example.com');
    assert.equal(await tableRow.getByRole('link', { name: '线上发布系统', exact: true }).getAttribute('href'), 'https://deploy.example.com');
    await page.getByRole('button', { name: '卡片视图', exact: true }).click();
    await card.getByRole('link', { name: '双发布系统回归', exact: true }).click();
    await page.getByRole('button', { name: '编辑项目', exact: true }).click();
    assert.equal(await dialog().getByLabel('入口名称（选填）', { exact: true }).nth(0).inputValue(), '测试发布系统');
    assert.equal(await dialog().getByLabel('入口名称（选填）', { exact: true }).nth(1).inputValue(), '线上发布系统');
    await dialog().locator('.entry-form-row').nth(1).getByRole('button', { name: '上移', exact: true }).click();
    await dialog().getByRole('button', { name: '保存修改', exact: true }).click();
    await page.getByRole('dialog').waitFor({ state: 'hidden' });
    await page.getByRole('button', { name: '编辑项目', exact: true }).click();
    assert.equal(await dialog().getByLabel('入口名称（选填）', { exact: true }).nth(0).inputValue(), '线上发布系统');
    await page.keyboard.press('Escape');

    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto(`${origin}/tests/design-preview.html?route=/projects&scenario=project-save-error`);
    await fillProject();
    await dialog().getByRole('button', { name: '保存项目', exact: true }).click();
    await dialog().getByText('网络连接失败，请稍后重试', { exact: true }).waitFor();
    assert.equal(await dialog().getByLabel('入口名称（选填）', { exact: true }).nth(1).inputValue(), '线上发布系统');
    assert.equal(await dialog().getByRole('button', { name: '保存项目', exact: true }).isEnabled(), true);
    assert(await dialog().evaluate(el => el.scrollWidth <= el.clientWidth + 1));
    assert.deepEqual(errors, []);
    console.log('PASS: same-type deploy entries create, card labels, edit hydration, reorder/save, narrow layout and failed-save draft retention');
  } finally {
    await browser.close();
  }
}
run().catch(error => { console.error(error); process.exitCode = 1; });
