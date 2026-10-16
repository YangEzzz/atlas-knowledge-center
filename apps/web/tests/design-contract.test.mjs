import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import vm from "node:vm";
import ts from "typescript";
const source = readFileSync(
  new URL("../src/utils/project-identity.ts", import.meta.url),
  "utf8",
);
const js = ts.transpileModule(source, {
  compilerOptions: {
    module: ts.ModuleKind.CommonJS,
    target: ts.ScriptTarget.ES2022,
  },
}).outputText;
const exports = {};
vm.runInNewContext(js, { exports, Date });
const { projectIdentity, projectOwnerNames, isStale } = exports;

test("project identity is stable across navigation and does not split Chinese characters", () => {
  const first = projectIdentity(101, "Nexus 多语言平台");
  assert.equal(first.label, projectIdentity(101, "Nexus 多语言平台").label);
  assert.equal(first.tone, projectIdentity(101, "Nexus 多语言平台").tone);
  assert.ok(["blue", "green", "clay"].includes(first.tone));
  assert.equal(projectIdentity(102, "云手机用户端").label, "云手");
  assert.equal(projectIdentity(103, "  ").label, "项");
});
test("owners are deduplicated across roles and absent owners remain truthful", () => {
  assert.equal(
    projectOwnerNames({
      owners: {
        frontend: [{ ownerName: "杨林" }],
        backend: [{ ownerName: "杨林" }, { ownerName: "陈晨" }],
        tester: null,
      },
    }),
    "杨林、陈晨",
  );
  assert.equal(projectOwnerNames({ owners: {} }), "负责人待补充");
});
test("invalid or future dates are never labelled stale", () => {
  assert.equal(isStale("not-a-date"), false);
  assert.equal(isStale(new Date(Date.now() + 86400000).toISOString()), false);
  assert.equal(
    isStale(new Date(Date.now() - 91 * 86400000).toISOString()),
    true,
  );
});
