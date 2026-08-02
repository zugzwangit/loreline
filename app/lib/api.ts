type ApiError = {
  detail?: string;
};

export async function jsonRequest<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const response = await fetch(path, init);
  const body =
    response.status === 204
      ? null
      : await response.json().catch(() => ({}) as ApiError);

  if (!response.ok) {
    const candidate =
      body && typeof body === "object" && "detail" in body
        ? body.detail
        : undefined;
    const detail =
      typeof candidate === "string" && candidate.trim()
        ? candidate
        : `Request failed (${response.status})`;
    throw new Error(detail);
  }

  return body as T;
}
