import { useState, useRef, useEffect } from "react";
import type { Task } from "../types";

interface Props {
  task: Task;
  onUpdate: (id: string, updates: Partial<Task> & { version: number }) => void;
  onStartDrag?: (id: string, e: React.DragEvent) => void;
}

export function TaskCard({ task, onUpdate, onStartDrag }: Props) {
  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState(task.title);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    setTitle(task.title);
  }, [task.title]);

  useEffect(() => {
    if (editing) inputRef.current?.focus();
  }, [editing]);

  const handleBlur = () => {
    setEditing(false);
    if (title !== task.title) {
      onUpdate(task.id, { title, version: task.version });
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      (e.target as HTMLInputElement).blur();
    }
    if (e.key === "Escape") {
      setTitle(task.title);
      setEditing(false);
    }
  };

  const cycleStatus = () => {
    const next = task.status === "todo" ? "in_progress"
      : task.status === "in_progress" ? "done"
      : "todo";
    onUpdate(task.id, { status: next, version: task.version });
  };

  return (
    <div
      draggable
      onDragStart={(e) => onStartDrag?.(task.id, e)}
      style={{
        background: "#fff",
        border: "1px solid #ddd",
        borderRadius: 6,
        padding: 12,
        marginBottom: 8,
        cursor: "grab",
      }}
    >
      {editing ? (
        <input
          ref={inputRef}
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          onBlur={handleBlur}
          onKeyDown={handleKeyDown}
          style={{ width: "100%", boxSizing: "border-box" }}
        />
      ) : (
        <div onClick={() => setEditing(true)} style={{ fontWeight: 500, marginBottom: 4 }}>
          {title}
        </div>
      )}
      <div style={{ display: "flex", gap: 8, alignItems: "center", fontSize: 12 }}>
        <button onClick={cycleStatus} style={{ fontSize: 11, padding: "2px 6px" }}>
          {task.status}
        </button>
        <span style={{ color: "#999" }}>v{task.version}</span>
      </div>
    </div>
  );
}
