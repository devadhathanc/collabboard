export type Role = "owner" | "admin" | "member" | "viewer";

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface User {
  id: string;
  roles: Role[];
}

export interface Task {
  id: string;
  board_id: string;
  title: string;
  status: string;
  assignee_id: string | null;
  position: number;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface WsEvent {
  id: string;
  sequence_id: number;
  board_id: string;
  type: WsEventType;
  actor_id: string;
  data: unknown;
  timestamp: string;
}

export type WsEventType =
  | "task.created"
  | "task.updated"
  | "task.deleted"
  | "task.moved"
  | "board.updated"
  | "cursor.move"
  | "resync";

export interface CursorData {
  user_id: string;
  x: number;
  y: number;
}

export interface ConflictResponse {
  error: string;
  current: Task;
}

export interface BulkUpdateResponse {
  updated: Task[];
  conflicts: { id: string; current: Task }[];
}
