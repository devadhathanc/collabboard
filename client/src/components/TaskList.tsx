import { useState, useCallback } from "react";
import { TaskCard } from "./TaskCard";
import { ConflictDialog } from "./ConflictDialog";
import type { Task, ConflictResponse } from "../types";
import { updateTask as apiUpdateTask, bulkUpdateTasks } from "../api/tasks";

interface Props {
  boardId: string;
  tasks: Task[];
  onTasksChange: (tasks: Task[]) => void;
}

export function TaskList({ boardId, tasks, onTasksChange }: Props) {
  const [conflict, setConflict] = useState<{
    local: Task;
    server: Task;
  } | null>(null);
  const [newTitle, setNewTitle] = useState("");

  const handleUpdate = useCallback(async (id: string, updates: Partial<Task> & { version: number }) => {
    const result = await apiUpdateTask(boardId, id, updates);
    if ("current" in result && "error" in result) {
      const cr = result as ConflictResponse;
      const local = tasks.find((t) => t.id === id);
      if (local) setConflict({ local, server: cr.current });
      return;
    }
    const updated = result as Task;
    onTasksChange(tasks.map((t) => (t.id === id ? updated : t)));
  }, [boardId, tasks, onTasksChange]);

  const handleKeepMine = (local: Task) => {
    if (!conflict) return;
    handleUpdate(conflict.local.id, {
      title: conflict.local.title,
      status: conflict.local.status,
      version: conflict.server.version,
    });
    setConflict(null);
  };

  const handleTakeTheirs = (server: Task) => {
    onTasksChange(tasks.map((t) => (t.id === server.id ? server : t)));
    setConflict(null);
  };

  const handleDragEnd = async (e: React.DragEvent) => {
    e.preventDefault();
    const id = e.dataTransfer.getData("text/plain");
    if (!id) return;
    const dragged = tasks.find((t) => t.id === id);
    if (!dragged) return;
    const dropY = e.clientY;
    const sorted = [...tasks].sort((a, b) => a.position - b.position);
    let insertBefore = sorted.length;
    for (let i = 0; i < sorted.length; i++) {
      const el = document.getElementById(`task-${sorted[i].id}`);
      if (el) {
        const rect = el.getBoundingClientRect();
        if (dropY < rect.top + rect.height / 2) {
          insertBefore = i;
          break;
        }
      }
    }
    const reordered = tasks.filter((t) => t.id !== id);
    let newPos: number;
    if (insertBefore === 0) {
      newPos = sorted[0] ? sorted[0].position - 1 : 0;
    } else if (insertBefore >= reordered.length) {
      newPos = reordered.length > 0 ? reordered[reordered.length - 1].position + 1 : 0;
    } else {
      const before = reordered[insertBefore - 1];
      const after = reordered[insertBefore];
      newPos = before ? (before.position + after.position) / 2 : after.position - 1;
    }
    const result = await bulkUpdateTasks(boardId, [{ id, position: newPos, version: dragged.version }]);
    if (result.conflicts?.length) {
      setConflict({ local: dragged, server: result.conflicts[0].current });
    }
    if (result.updated?.length) {
      onTasksChange(tasks.map((t) => result.updated.find((u) => u.id === t.id) || t));
    }
  };

  const handleCreate = async () => {
    if (!newTitle.trim()) return;
    const maxPos = tasks.reduce((m, t) => Math.max(m, t.position), 0);
    const task = await apiUpdateTask(boardId, "", { title: newTitle.trim(), version: 0 });
    setNewTitle("");
  };

  const sorted = [...tasks].sort((a, b) => a.position - b.position);

  return (
    <div onDragOver={(e) => e.preventDefault()} onDrop={handleDragEnd}>
      <div style={{ display: "flex", gap: 8, marginBottom: 16 }}>
        <input
          value={newTitle}
          onChange={(e) => setNewTitle(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && handleCreate()}
          placeholder="New task..."
          style={{ flex: 1, padding: 8 }}
        />
        <button onClick={handleCreate} style={{ padding: "8px 16px" }}>Add</button>
      </div>
      {sorted.map((task) => (
        <div key={task.id} id={`task-${task.id}`}>
          <TaskCard
            task={task}
            onUpdate={handleUpdate}
            onStartDrag={(id, e) => e.dataTransfer?.setData("text/plain", id)}
          />
        </div>
      ))}
      {conflict && (
        <ConflictDialog
          task={conflict.local}
          serverTask={conflict.server}
          onKeepMine={handleKeepMine}
          onTakeTheirs={handleTakeTheirs}
          onCancel={() => setConflict(null)}
        />
      )}
    </div>
  );
}
