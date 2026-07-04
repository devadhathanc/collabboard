import { api, ApiError } from "./client";
import type { Task, ConflictResponse, BulkUpdateResponse } from "../types";

export async function createTask(
  boardId: string,
  title: string,
  position: number,
): Promise<Task> {
  return api.post<Task>(`/boards/${boardId}/tasks`, { title, position });
}

export async function getTask(boardId: string, taskId: string): Promise<Task> {
  return api.get<Task>(`/boards/${boardId}/tasks/${taskId}`);
}

export async function updateTask(
  boardId: string,
  taskId: string,
  updates: Partial<Task> & { version: number },
): Promise<Task | ConflictResponse> {
  try {
    return await api.patch<Task>(`/boards/${boardId}/tasks/${taskId}`, updates);
  } catch (e) {
    if (e instanceof ApiError && e.status === 409) {
      return JSON.parse(e.message) as ConflictResponse;
    }
    throw e;
  }
}

export async function bulkUpdateTasks(
  boardId: string,
  tasks: { id: string; position: number; version: number }[],
): Promise<BulkUpdateResponse> {
  try {
    const res = await api.patch<BulkUpdateResponse>(
      `/boards/${boardId}/tasks`,
      { tasks },
    );
    return res;
  } catch (e) {
    if (e instanceof ApiError && e.status === 409) {
      return JSON.parse(e.message) as BulkUpdateResponse;
    }
    throw e;
  }
}

export async function searchTasks(
  boardId: string,
  query: string,
): Promise<Task[]> {
  return api.get<Task[]>(
    `/boards/${boardId}/search?q=${encodeURIComponent(query)}`,
  );
}
