const assert = require("node:assert/strict");
const { chromium } = require("playwright");
const origin = process.env.ATLAS_PREVIEW_ORIGIN || "http://127.0.0.1:5188";
(async () => {
  const browser = await chromium.launch({ channel: "msedge", headless: true });
  const page = await browser.newPage({
    viewport: { width: 1440, height: 1000 },
    locale: "zh-CN",
  });
  const errors = [];
  page.on("pageerror", (e) => {
    errors.push(e.message);
    console.error("PAGE ERROR", e.stack);
  });
  page.on("dialog", (dialog) => dialog.accept());
  const open = async (route = "/documents/editor", scenario = "") => {
    await page.goto(
      `${origin}/tests/design-preview.html?route=${encodeURIComponent(route)}&scenario=${scenario}`,
    );
    await page
      .getByRole("heading", { name: "编辑文档", exact: true })
      .waitFor();
  };
  const title = () => page.getByLabel("文档标题", { exact: false });
  const editor = () => page.locator("[data-slate-editor]");
  const button = (name) => page.getByRole("button", { name, exact: true });
  const saved = () =>
    page.getByRole("status").filter({ hasText: "草稿已保存" }).waitFor();
  const select = async (name, option) => {
    await page.getByRole("combobox", { name, exact: false }).click();
    await page.getByRole("option", { name: option, exact: true }).click();
  };
  const payloads = () =>
    page.evaluate(() =>
      window.__atlasPreviewRequests
        .filter((r) => r.url === "/documents" && r.method === "post")
        .map((r) => JSON.parse(r.data)),
    );
  try {
    // The project-detail missing slot is a create action, not a broken read link.
    const openProject = async () => {
      await page.goto(
        origin +
          "/tests/design-preview.html?route=" +
          encodeURIComponent("/projects/101"),
      );
      await page.locator(".missing-document-types").waitFor();
    };
    await openProject();
    await page
      .locator(".missing-document-types")
      .getByRole("button", { name: "项目介绍", exact: true })
      .click();
    await title().waitFor();
    assert.equal(await title().inputValue(), "");
    await page.waitForFunction(() =>
      document
        .getElementById("document-project")
        .textContent.includes("Nexus 多语言平台"),
    );
    assert.match(
      await page
        .getByRole("combobox", { name: "关联项目", exact: false })
        .innerText(),
      /Nexus 多语言平台/,
    );
    assert.match(
      await page
        .getByRole("combobox", { name: "文档类型", exact: true })
        .innerText(),
      /项目介绍/,
    );
    assert.equal(
      await page.evaluate(
        () =>
          window.__atlasPreviewRequests.filter((r) =>
            r.url.startsWith("/documents/project/"),
          ).length,
      ),
      0,
    );
    await title().fill("补充项目介绍");
    await editor().fill("从待补充入口创建的文档");
    await button("保存草稿").click();
    await saved();
    const created = (await payloads()).at(-1);
    assert.equal(created.projectId, 101);
    assert.equal(created.type, "intro");
    assert.equal(created.id, undefined);
    await openProject();
    await page
      .getByRole("button", { name: "编辑测试环境与账号申请", exact: true })
      .click();
    await title().waitFor();
    assert.equal(await title().inputValue(), "测试环境与账号申请");
    assert.equal(
      await page.evaluate(
        () =>
          window.__atlasPreviewRequests.filter(
            (r) => r.url === "/documents/203",
          ).length,
      ),
      1,
    );
    // Legacy links receive HTTP 200 with business code 404 from the real API.
    await open("/documents/editor?projectId=103&type=environment");
    await title().waitFor();
    await page
      .getByText("已使用模板「团队环境说明模板」，可直接修改正文。", {
        exact: true,
      })
      .waitFor();
    assert.ok(await button("保存草稿").isEnabled());
    await open(
      "/documents/editor?projectId=103&type=environment",
      "document-business-error",
    );
    await page
      .getByRole("alert")
      .filter({ hasText: "文档读取暂时不可用" })
      .waitFor();
    assert.ok(await button("保存草稿").isDisabled());
    await open();
    await title().waitFor();
    assert.equal(await title().inputValue(), "");
    assert.match(
      await page
        .getByRole("combobox", { name: "关联项目", exact: false })
        .innerText(),
      /不关联项目/,
    );
    await button("保存草稿").click();
    await page
      .getByRole("alert")
      .filter({ hasText: "请填写文档标题" })
      .waitFor();
    assert.equal(await title().getAttribute("aria-invalid"), "true");
    await title().fill("第一篇通用指南");
    await editor().fill("第一篇通用文档的正文");
    await button("预览").click();
    await page
      .getByRole("dialog")
      .getByRole("heading", { name: "第一篇通用指南", exact: true })
      .waitFor();
    await page.keyboard.press("Escape");
    await button("返回文档库").click();
    await button("继续编辑").click();
    assert.equal(await title().inputValue(), "第一篇通用指南");
    await button("保存草稿").click();
    await saved();
    let requests = await payloads();
    assert.equal(requests[0].projectId, null);
    assert.equal(requests[0].id, undefined);
    await title().fill("第一篇通用指南（修订）");
    await button("保存草稿").click();
    await saved();
    requests = await payloads();
    assert.equal(requests[1].id, 999);
    assert.equal(requests[1].projectId, null);
    await button("返回文档库").click();
    await page.getByRole("heading", { name: "文档库", exact: true }).waitFor();
    const firstRow = () =>
      page.getByRole("row").filter({ hasText: "第一篇通用指南（修订）" });
    await firstRow().getByRole("button", { name: "编辑", exact: true }).click();
    await title().waitFor();
    assert.equal(await title().inputValue(), "第一篇通用指南（修订）");
    assert.match(await editor().innerText(), /第一篇通用文档的正文/);
    await button("返回文档库").click();
    await button("新建文档").click();
    await title().waitFor();
    assert.equal(await title().inputValue(), "");
    await title().fill("第二篇通用指南");
    await editor().fill("独立的第二篇正文");
    await button("保存草稿").click();
    await saved();
    requests = await payloads();
    assert.equal(requests.at(-1).id, undefined);
    assert.equal(requests.at(-1).projectId, null);
    await button("返回文档库").click();
    await firstRow().getByRole("button", { name: "查看", exact: true }).click();
    await page
      .getByRole("heading", { name: "第一篇通用指南（修订）", exact: true })
      .waitFor();
    assert.equal(await button("查看项目").count(), 0);
    assert.ok(await page.getByText("不关联项目", { exact: true }).count());
    await button("编辑文档").click();
    await title().waitFor();
    assert.match(await editor().innerText(), /第一篇通用文档的正文/);
    await button("发布文档").click();
    await page.getByRole("status").filter({ hasText: "文档已发布" }).waitFor();

    await open("/documents/editor?projectId=101&type=environment");
    await title().waitFor();
    assert.equal(await title().inputValue(), "测试环境与账号申请");
    await select("关联项目", "不关联项目（通用文档）");
    await button("保存草稿").click();
    await saved();
    requests = await payloads();
    assert.equal(requests.at(-1).id, 203);
    assert.equal(requests.at(-1).projectId, null);

    await open("/documents/editor", "document-save-error");
    await title().waitFor();
    await title().fill("失败后保留");
    await editor().fill("保留正文");
    await button("保存草稿").click();
    await page.getByRole("alert").filter({ hasText: "网络连接失败" }).waitFor();
    assert.equal(await title().inputValue(), "失败后保留");
    await button("保存草稿").click();
    await saved();
    await open("/documents/editor", "project-list-error");
    await title().waitFor();
    await page.getByRole("button", { name: "重试项目列表" }).waitFor();
    await title().fill("不依赖项目列表");
    await editor().fill("即使没有项目也可以写");
    await button("保存草稿").click();
    await saved();
    await open("/documents/editor", "empty");
    await title().waitFor();
    assert.ok(await button("保存草稿").isEnabled());
    await title().fill("没有项目也可保存");
    await button("保存草稿").click();
    await saved();

    await open("/documents/editor?id=203", "document-load-error");
    await button("重新加载").waitFor();
    assert.ok(await button("保存草稿").isDisabled());
    await button("重新加载").click();
    await title().waitFor();
    await open(
      "/documents/editor?projectId=103&type=environment",
      "template-slow",
    );
    await title().waitFor();
    await editor().fill("用户正文不应被模板覆盖");
    await page.waitForTimeout(1800);
    assert.match(await editor().innerText(), /用户正文不应被模板覆盖/);
    await open("/documents/editor");
    await title().waitFor();
    await title().fill("通用文档示例");
    await editor().fill("团队共用的知识不需要绑定某个项目。");
    await page.screenshot({ path: "tests/editor-desktop.png", fullPage: true });
    await page
      .getByLabel("选择文档图片", { exact: true })
      .setInputFiles("tests/image-upload.png");
    await editor().locator("img").waitFor();
    await page.getByText("图片已上传并插入", { exact: true }).waitFor();
    await page.setViewportSize({ width: 390, height: 844 });
    await page.emulateMedia({ reducedMotion: "reduce" });
    await page.screenshot({ path: "tests/editor-mobile.png", fullPage: true });
    assert.ok(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    );
    await page.waitForFunction(
      () =>
        !document.getElementById("document-project").hasAttribute("disabled"),
    );
    await page
      .getByRole("combobox", { name: "关联项目", exact: false })
      .focus();
    await page.keyboard.press("Space");
    await page
      .getByRole("option", { name: "不关联项目（通用文档）", exact: true })
      .waitFor();
    await page.screenshot({
      path: "tests/editor-select-mobile.png",
      fullPage: true,
    });
    await page.keyboard.press("Escape");
    await button("预览").click();
    const bounds = await page.getByRole("dialog").boundingBox();
    assert.ok(bounds.x >= 0 && bounds.x + bounds.width <= 391);
    await page.keyboard.press("Escape");
    assert.deepEqual(errors, []);
    console.log(
      "PASS standalone editor: null project, multiple independent documents, ID updates/reopen, detail/list, association removal, legacy project entry, save/publish, validation/recovery, no-project/failing-project-list, template race, uploads, keyboard and narrow layout",
    );
  } catch (error) {
    console.error((await page.locator("body").innerText()).slice(-2500));
    throw error;
  } finally {
    await browser.close();
  }
})().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
