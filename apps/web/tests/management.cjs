const assert = require("node:assert/strict");
const { chromium } = require("playwright");
const origin = process.env.ATLAS_PREVIEW_ORIGIN || "http://127.0.0.1:5177";
async function run() {
  const browser = await chromium.launch({ channel: "msedge", headless: true });
  const page = await browser.newPage({
    viewport: { width: 1440, height: 1000 },
    locale: "zh-CN",
  });
  const errors = [];
  page.on("pageerror", (err) => errors.push(err.message));
  await page.route("**/api/v1/**", (route) =>
    route.request().url().endsWith("/auth/csrf")
      ? route.fulfill({
          json: { code: 200, data: { csrf_token: "preview-only" } },
        })
      : route.abort(),
  );
  const goto = async (route = "/manage", scenario = "") => {
    await page.goto(
      `${origin}/tests/design-preview.html?route=${encodeURIComponent(route)}&scenario=${scenario}`,
    );
    await page
      .getByRole("heading", {
        name: route.includes("editor") ? "编辑文档" : "管理中心",
        exact: true,
      })
      .waitFor();
  };
  const dialog = () => page.getByRole("dialog").last();
  const select = async (name, option) => {
    await page.getByRole("combobox", { name, exact: true }).click();
    await page.getByRole("option", { name: option, exact: true }).click();
  };
  try {
    await goto();
    await page.getByText("团队环境说明模板", { exact: true }).waitFor();
    assert.equal(await page.locator("tbody tr").count(), 20);
    await page.getByRole("button", { name: "下一页" }).click();
    await page.getByText("第 2 / 2 页", { exact: false }).waitFor();
    await page.waitForFunction(
      () => document.querySelectorAll("tbody tr").length === 3,
    );
    await page.getByRole("button", { name: "上一页" }).click();
    await page.getByText("团队环境说明模板", { exact: true }).waitFor();
    await page.getByLabel("名称", { exact: true }).fill("不存在");
    await page.getByRole("button", { name: "查询", exact: true }).click();
    await page.getByText("没有符合条件的记录").waitFor();
    await page.getByRole("button", { name: "清除关键词" }).click();
    await page.getByText("团队环境说明模板", { exact: true }).waitFor();
    await select("文档类型", "环境说明");
    await page.getByRole("button", { name: "查询", exact: true }).click();
    await page.waitForFunction(
      () => document.querySelectorAll("tbody tr").length === 1,
    );
    await page.getByRole("button", { name: "停用", exact: true }).click();
    await dialog()
      .getByRole("button", { name: "停用模板", exact: true })
      .click();
    await page.getByRole("button", { name: "启用", exact: true }).waitFor();
    await page.getByRole("button", { name: "新增模板", exact: true }).click();
    await dialog()
      .getByRole("button", { name: "保存模板", exact: true })
      .click();
    await page.getByText("请填写模板名称", { exact: true }).waitFor();
    await dialog()
      .getByLabel("模板名称", { exact: true })
      .fill("浏览器回归模板");
    const editable = dialog().locator("[data-slate-editor]");
    await editable.click();
    await page.keyboard.type("浏览器模板正文");
    await dialog().getByRole("button", { name: "取消", exact: true }).click();
    await page.getByRole("heading", { name: "放弃未保存的模板？" }).waitFor();
    await dialog().getByRole("button", { name: "继续编辑" }).click();
    await dialog().getByLabel("模板名称", { exact: true }).waitFor();
    await dialog()
      .getByRole("button", { name: "保存模板", exact: true })
      .click();
    await page.getByRole("dialog").waitFor({ state: "hidden" });
    await page.getByText("浏览器回归模板", { exact: true }).waitFor();
    const row = page.getByRole("row").filter({ hasText: "浏览器回归模板" });
    await row.getByRole("button", { name: "编辑", exact: true }).click();
    await dialog().getByLabel("模板名称").fill("浏览器已编辑模板");
    await dialog().getByRole("button", { name: "保存模板" }).click();
    await page.getByText("浏览器已编辑模板", { exact: true }).waitFor();
    const edited = page
      .getByRole("row")
      .filter({ hasText: "浏览器已编辑模板" });
    await edited.getByRole("button", { name: "删除", exact: true }).click();
    await dialog().getByRole("button", { name: "取消", exact: true }).click();
    await page.getByRole("dialog").waitFor({ state: "hidden" });
    assert.equal(await edited.count(), 1);
    await edited.getByRole("button", { name: "删除", exact: true }).click();
    await dialog()
      .getByRole("button", { name: "删除模板", exact: true })
      .click();
    await edited.waitFor({ state: "hidden" });
    await page.getByRole("tab", { name: "回收站", exact: true }).click();
    await page.getByRole("row").filter({ hasText: "测试发布入口" }).waitFor();
    assert.equal(
      await page
        .getByRole("row")
        .filter({ hasText: "测试发布入口" })
        .getByRole("button", { name: "恢复", exact: true })
        .isDisabled(),
      true,
    );
    await page
      .getByRole("row")
      .filter({ hasText: "监控工具" })
      .getByRole("button", { name: "恢复", exact: true })
      .click();
    await dialog().getByRole("button", { name: "恢复", exact: true }).click();
    await page
      .getByRole("row")
      .filter({ hasText: "监控工具" })
      .waitFor({ state: "hidden" });
    await page.getByRole("tab", { name: "操作审计", exact: true }).click();
    await page
      .getByRole("row")
      .filter({ hasText: "生产环境部署指南" })
      .waitFor();
    await page.getByLabel("操作人", { exact: true }).fill("杨林");
    await page.getByLabel("开始时间（北京时间）").fill("2026-10-15T09:00");
    await page.getByRole("button", { name: "查询", exact: true }).click();
    await page.waitForFunction(() =>
      window.__atlasPreviewRequests.some(
        (r) =>
          r.url === "/manage/audit" &&
          r.params?.from === "2026-10-15T09:00:00+08:00",
      ),
    );
    await page.getByRole("button", { name: "详情", exact: true }).click();
    await dialog().getByText("正文、状态").waitFor();
    await page.keyboard.press("Escape");
    await page.getByRole("dialog").waitFor({ state: "hidden" });
    await page.screenshot({
      path: "tests/management-desktop.png",
      fullPage: true,
    });
    await page.setViewportSize({ width: 390, height: 844 });
    await page.screenshot({
      path: "tests/management-mobile.png",
      fullPage: true,
    });
    assert.equal(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
      true,
    );
    await goto("/manage", "empty");
    await page.getByText("还没有文档模板", { exact: true }).waitFor();
    await goto("/manage", "error");
    await page.getByRole("alert").waitFor();
    assert.equal(
      await page.getByRole("button", { name: "重试", exact: true }).count(),
      1,
    );
    await page.setViewportSize({ width: 1440, height: 1000 });
    await goto("/manage", "management-failure");
    await page.getByRole("button", { name: "新增模板", exact: true }).click();
    await dialog().getByLabel("模板名称").fill("失败重试模板");
    await dialog().locator("[data-slate-editor]").click();
    await page.keyboard.type("保留正文");
    await dialog().getByRole("button", { name: "保存模板" }).click();
    await dialog().getByRole("alert").waitFor();
    assert.equal(
      await dialog().getByLabel("模板名称").inputValue(),
      "失败重试模板",
    );
    await dialog().getByRole("button", { name: "保存模板" }).click();
    await page.getByRole("dialog").waitFor({ state: "hidden" });
    await goto("/documents/editor?projectId=103&type=environment");
    await page
      .getByText("已使用模板「团队环境说明模板」，可直接修改正文。", {
        exact: true,
      })
      .waitFor();
    await page.locator("[data-slate-editor]").waitFor();
    assert.match(
      await page.locator("[data-slate-editor]").innerText(),
      /团队模板正文/,
    );
    await goto("/documents/editor?projectId=101&type=environment");
    await page.locator("[data-slate-editor]").waitFor();
    assert.match(
      await page.locator("[data-slate-editor]").innerText(),
      /联系项目负责人申请账号/,
    );
    assert.doesNotMatch(
      await page.locator("[data-slate-editor]").innerText(),
      /团队模板正文/,
    );
    await goto(
      "/documents/editor?projectId=103&type=environment",
      "template-error",
    );
    await page.getByRole("button", { name: "重试模板" }).click();
    await page
      .getByText("已使用模板「团队环境说明模板」，可直接修改正文。", {
        exact: true,
      })
      .waitFor();
    await goto("/documents/editor?projectId=103&type=faq", "template-slow");
    await page.locator("[data-slate-editor]").waitFor();
    await page.getByRole("combobox").nth(1).click();
    await page.getByRole("option", { name: "环境说明", exact: true }).click();
    await page
      .locator("[data-slate-editor]")
      .fill("正在编写的正文不能被模板覆盖");
    await page.waitForTimeout(1700);
    assert.match(
      await page.locator("[data-slate-editor]").innerText(),
      /正在编写的正文不能被模板覆盖/,
    );
    assert.doesNotMatch(
      await page.locator("[data-slate-editor]").innerText(),
      /团队模板正文/,
    );
    await goto("/manage", "management-slow");
    await page.locator('[aria-busy="true"]').waitFor();
    await page.getByText("团队环境说明模板", { exact: true }).waitFor();
    await page.getByRole("combobox", { name: "文档类型", exact: true }).focus();
    await page.keyboard.press("Space");
    await page.getByRole("option", { name: "全部类型", exact: true }).waitFor();
    await page.keyboard.press("ArrowDown");
    await page.keyboard.press("Escape");
    assert.equal(
      await page
        .getByRole("combobox", { name: "文档类型", exact: true })
        .getAttribute("aria-expanded"),
      "false",
    );
    await page.setViewportSize({ width: 390, height: 844 });
    await page.emulateMedia({ reducedMotion: "reduce" });
    await page.getByRole("button", { name: "新增模板", exact: true }).click();
    await dialog()
      .getByRole("combobox", { name: "文档类型", exact: true })
      .click();
    await page.getByRole("option", { name: "环境说明", exact: true }).waitFor();
    await page.keyboard.press("Escape");
    const bounds = await dialog().boundingBox();
    assert.ok(bounds.x >= 0 && bounds.x + bounds.width <= 391);
    await dialog().getByRole("button", { name: "取消", exact: true }).click();
    if (await page.getByRole("heading", { name: "放弃未保存的模板？" }).count())
      await dialog().getByRole("button", { name: "放弃更改" }).click();
    await page.getByRole("dialog").waitFor({ state: "hidden" });
    assert.deepEqual(errors, []);
    console.log(
      "PASS management CRUD, filters/pagination, trash recovery, audit detail/timezone, dirty confirmation, failure/retry, narrow layout, template fill and existing-content protection",
    );
  } catch (error) {
    console.error((await page.locator("body").innerText()).slice(-4500));
    console.error(
      await page.evaluate(() => window.__atlasPreviewRequests.slice(-8)),
    );
    await page.screenshot({
      path: "tests/management-failure.png",
      fullPage: true,
    });
    throw error;
  } finally {
    await browser.close();
  }
}
run().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
