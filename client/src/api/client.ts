const API = "/api";

function token(): string | null {
  return localStorage.getItem("access_token");
}

async function request<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...((init?.headers as Record<string, string>) || {}),
  };
  const t = token();
  if (t) headers["Authorization"] = `Bearer ${t}`;

  const res = await fetch(`${API}${path}`, { ...init, headers });

  if (res.status === 401) {
    const ok = await tryRefresh();
    if (ok) {
      headers["Authorization"] = `Bearer ${token()}`;
      const retry = await fetch(`${API}${path}`, { ...init, headers });
      if (!retry.ok) throw new ApiError(retry.status, await retry.text());
      return retry.json();
    }
    throw new ApiError(401, "unauthorized");
  }

  if (!res.ok) {
    const body = await res.text();
    throw new ApiError(res.status, body);
  }

  if (res.status === 204) return undefined as T;
  return res.json();
}

export class ApiError extends Error {
  constructor(
    public status: number,
    body: string,
  ) {
    super(body);
  }
}

async function tryRefresh(): Promise<boolean> {
  const rt = localStorage.getItem("refresh_token");
  if (!rt) return false;
  try {
    const res = await fetch(`${API}/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: rt }),
    });
    if (!res.ok) {
      localStorage.removeItem("access_token");
      localStorage.removeItem("refresh_token");
      return false;
    }
    const data = await res.json();
    localStorage.setItem("access_token", data.access_token);
    localStorage.setItem("refresh_token", data.refresh_token);
    return true;
  } catch {
    return false;
  }
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "POST", body: body ? JSON.stringify(body) : undefined }),
  patch: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "PATCH", body: body ? JSON.stringify(body) : undefined }),
  del: <T>(path: string) => request<T>(path, { method: "DELETE" }),
};
