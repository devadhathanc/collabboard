import { useState, useCallback } from "react";
import { TaskList } from "./TaskList";
import { SearchBar } from "./SearchBar";
import { useWebSocket } from "../ws/useWebSocket";
import { useAuth } from "../context/AuthContext";
import type { Task, WsEvent } from "../types";

interface Props {
  boardId: string;
}

export function Board({ boardId }: Props) {
  const { userId } = useAuth();
  const [tasks, setTasks] = useState<Task[]>([]);
  const [cursors, setCursors] = useState<Map<string, { x: number; y: number }>>(new Map());

  const handleWsEvent = useCallback((evt: WsEvent) => {
    switch (evt.type) {
      case "task.created":
      case "task.updated":
        setTasks((prev) => {
          const data = evt.data as Task;
          const idx = prev.findIndex((t) => t.id === data.id);
          if (idx >= 0) {
            const next = [...prev];
            next[idx] = data;
            return next;
          }
          return [...prev, data];
        });
        break;
      case "task.moved":
        setTasks((prev) => {
          const data = evt.data as Task;
          return prev.map((t) => (t.id === data.id ? data : t));
        });
        break;
      case "task.deleted":
        setTasks((prev) => prev.filter((t) => t.id !== (evt.data as Task).id));
        break;
      case "cursor.move":
        const data = evt.data as { user_id: string; x: number; y: number };
        setCursors((prev) => {
          const next = new Map(prev);
          next.set(data.user_id, { x: data.x, y: data.y });
          return next;
        });
        break;
      case "resync":
        if (Array.isArray(evt.data)) {
          setTasks(evt.data as Task[]);
        }
        break;
    }
  }, []);

  const { connected, send } = useWebSocket({ boardId, userId, onEvent: handleWsEvent });

  const handleTasksChange = useCallback((updated: Task[]) => {
    setTasks(updated);
  }, []);

  const handleSearchResults = useCallback((results: Task[]) => {
    setTasks(results);
  }, []);

  const handleMouseMove = useCallback(
    (e: React.MouseEvent) => {
      send({
        type: "cursor.move",
        payload: { user_id: userId, x: e.clientX, y: e.clientY },
      });
    },
    [send, userId],
  );

  return (
    <div onMouseMove={handleMouseMove} style={{ position: "relative", minHeight: "100vh" }}>
      <div style={{ padding: 16, maxWidth: 600, margin: "0 auto" }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16 }}>
          <h2 style={{ margin: 0 }}>Board {boardId}</h2>
          <span style={{ fontSize: 12, color: connected ? "green" : "red" }}>
            {connected ? "connected" : "disconnected"}
          </span>
        </div>

        <SearchBar boardId={boardId} onResults={handleSearchResults} />

        <TaskList
          boardId={boardId}
          tasks={tasks}
          onTasksChange={handleTasksChange}
        />
      </div>

      {Array.from(cursors.entries()).map(([uid, pos]) =>
        uid !== userId ? (
          <div
            key={uid}
            style={{
              position: "fixed",
              left: pos.x,
              top: pos.y,
              pointerEvents: "none",
              fontSize: 12,
              background: "rgba(0,0,0,0.7)",
              color: "#fff",
              padding: "2px 6px",
              borderRadius: 4,
              zIndex: 999,
            }}
          >
            {uid}
          </div>
        ) : null,
      )}
    </div>
  );
}
