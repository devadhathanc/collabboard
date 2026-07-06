import type { Idea, BrainstormResult } from "../types";

const API: string = (import.meta.env.VITE_API_URL as string) || "http://localhost:8090";


async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...init,
  });
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(`${res.status}: ${body}`);
  }
  // 204 No Content
  if (res.status === 204) return undefined as T;
  return res.json();
}

export async function listIdeas(boardId: string): Promise<Idea[]> {
  const data = await req<{ ideas: Idea[] }>(`/api/boards/${boardId}/ideas`);
  return data.ideas ?? [];
}

export async function submitIdea(
  boardId: string,
  text: string,
  createdBy: string,
): Promise<Idea> {
  return req<Idea>(`/api/boards/${boardId}/ideas`, {
    method: "POST",
    body: JSON.stringify({ text, created_by: createdBy }),
  });
}

export async function synthesizeIdeas(
  boardId: string,
): Promise<BrainstormResult> {
  return req<BrainstormResult>(`/api/boards/${boardId}/ideas/synthesize`, {
    method: "POST",
  });
}
