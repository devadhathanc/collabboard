import { api, ApiError } from "./client";
import type { LoginResponse } from "../types";

export async function login(userId: string, password: string): Promise<LoginResponse> {
  const res = await api.post<LoginResponse>("/auth/login", { user_id: userId, password });
  localStorage.setItem("access_token", res.access_token);
  localStorage.setItem("refresh_token", res.refresh_token);
  return res;
}

export async function revoke(): Promise<void> {
  try {
    await api.del("/auth/revoke");
  } catch {
    // ignore
  }
  localStorage.removeItem("access_token");
  localStorage.removeItem("refresh_token");
}

export function logout(): void {
  revoke();
}

export function isAuthenticated(): boolean {
  return !!localStorage.getItem("access_token");
}
