import type { Task } from "../types";

interface Props {
  task: Task;
  serverTask: Task;
  onKeepMine: (task: Task) => void;
  onTakeTheirs: (task: Task) => void;
  onCancel: () => void;
}

export function ConflictDialog({ task, serverTask, onKeepMine, onTakeTheirs, onCancel }: Props) {
  return (
    <div style={{
      position: "fixed", inset: 0, background: "rgba(0,0,0,0.4)",
      display: "flex", alignItems: "center", justifyContent: "center", zIndex: 100,
    }}>
      <div style={{ background: "#fff", padding: 24, borderRadius: 8, maxWidth: 480, width: "100%" }}>
        <h2>Version Conflict</h2>
        <p>Someone else edited this task while you were editing.</p>
        <div style={{ display: "flex", gap: 24, margin: "16px 0" }}>
          <div style={{ flex: 1 }}>
            <strong>Your version</strong>
            <pre style={{ fontSize: 13, whiteSpace: "pre-wrap" }}>
              {JSON.stringify({ title: task.title, status: task.status }, null, 2)}
            </pre>
            <button onClick={() => onKeepMine(task)} style={{ width: "100%", padding: 8 }}>
              Keep Mine
            </button>
          </div>
          <div style={{ flex: 1 }}>
            <strong>Server version</strong>
            <pre style={{ fontSize: 13, whiteSpace: "pre-wrap" }}>
              {JSON.stringify({ title: serverTask.title, status: serverTask.status }, null, 2)}
            </pre>
            <button onClick={() => onTakeTheirs(serverTask)} style={{ width: "100%", padding: 8 }}>
              Take Theirs
            </button>
          </div>
        </div>
        <button onClick={onCancel} style={{ width: "100%", padding: 8 }}>Cancel</button>
      </div>
    </div>
  );
}
