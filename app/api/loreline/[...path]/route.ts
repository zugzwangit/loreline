import { getChatGPTUser } from "../../../chatgpt-auth";

export const dynamic = "force-dynamic";

const ALLOWED = [
  /^v1\/dashboard$/,
  /^v1\/sources(?:\/[^/]+\/sync)?$/,
  /^v1\/documents$/,
  /^v1\/candidates(?:\/[^/]+\/decision)?$/,
  /^v1\/assistant\/(?:ask|stream)$/,
  /^v1\/feedback$/,
  /^v1\/audit$/,
  /^v1\/api-keys(?:\/[^/]+)?$/,
  /^v1\/jobs(?:\/[^/]+\/replay)?$/,
];

async function proxy(request: Request, context: { params: Promise<{ path: string[] }> }) {
  const user = await getChatGPTUser();
  const allowLocal = process.env.LORELINE_ALLOW_ANONYMOUS_PROXY === "true";
  if (!user && !allowLocal) return Response.json({ code: "unauthorized", detail: "Sign in is required" }, { status: 401 });
  const { path } = await context.params;
  const joined = path.join("/");
  if (!ALLOWED.some((rule) => rule.test(joined))) return Response.json({ code: "not_found" }, { status: 404 });
  const gateway = process.env.LORELINE_GATEWAY_URL;
  const token = process.env.LORELINE_GATEWAY_TOKEN;
  if (!gateway || !token) return Response.json({ code: "service_not_configured", detail: "The operational API connection has not been configured." }, { status: 503 });
  const incoming = new URL(request.url);
  const target = new URL(joined + incoming.search, gateway.endsWith("/") ? gateway : gateway + "/");
  const headers = new Headers({ Authorization: `Bearer ${token}`, "X-Request-ID": request.headers.get("X-Request-ID") ?? crypto.randomUUID() });
  for (const name of ["Content-Type", "Idempotency-Key"]) { const value=request.headers.get(name); if(value) headers.set(name,value); }
  const body = request.method === "GET" || request.method === "HEAD" ? undefined : await request.arrayBuffer();
  try {
    const response = await fetch(target, { method: request.method, headers, body, redirect: "manual" });
    const outgoing = new Headers();
    for (const name of ["content-type","x-request-id","retry-after"]) { const value=response.headers.get(name); if(value) outgoing.set(name,value); }
    return new Response(response.body,{status:response.status,headers:outgoing});
  } catch {
    return Response.json({ code:"gateway_unavailable",detail:"The operational API is unavailable." },{status:502});
  }
}

export const GET=proxy; export const POST=proxy; export const PATCH=proxy; export const DELETE=proxy;
