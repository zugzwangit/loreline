import assert from "node:assert/strict";
import test from "node:test";

async function application() {
  const workerUrl = new URL("../dist/server/index.js", import.meta.url);
  workerUrl.searchParams.set("test", `${process.pid}-${Date.now()}`);
  const { default: worker } = await import(workerUrl.href);
  return worker;
}

async function dispatch(request) {
  const worker = await application();
  return worker.fetch(request, { ASSETS: { fetch: async () => new Response("Not found", { status: 404 }) } }, { waitUntil() {}, passThroughOnException() {} });
}

test("server-renders the Loreline application", async () => {
  const response = await dispatch(new Request("http://localhost/", { headers: { accept: "text/html" } }));
  assert.equal(response.status, 200);
  assert.match(response.headers.get("content-type") ?? "", /^text\/html\b/i);
  const html = await response.text();
  assert.match(html, /<title>Loreline/);
  assert.match(html, /Good (morning|afternoon|evening)/);
  assert.match(html, /Knowledge pipeline/);
  assert.match(html, /Ask Loreline/);
  assert.match(html, /Audit log/);
  assert.match(html, /Settings/);
  assert.match(html, /Sample data · writes disabled/);
  assert.doesNotMatch(html, /codex-preview|Starter Project|loading skeleton/i);
});

test("server proxy requires an authenticated operator", async () => {
  const response = await dispatch(new Request("http://localhost/api/loreline/v1/dashboard"));
  assert.equal(response.status, 401);
  assert.equal((await response.json()).code, "unauthorized");
});

test("server proxy rejects non-UUID resource paths before forwarding", async () => {
  const response = await dispatch(new Request("http://localhost/api/loreline/v1/sources/not-a-uuid/sync", {
    headers: {
      "oai-authenticated-user-id": "operator-1",
      "oai-authenticated-user-email": "operator@example.com",
    },
  }));
  assert.equal(response.status, 404);
  assert.equal((await response.json()).code, "not_found");
});
