import { useState } from "react";
import { Login } from "./components/Login";
import { Board } from "./components/Board";

export function App() {
  const [session, setSession] = useState<{ name: string; boardId: string } | null>(null);

  if (!session) {
    return (
      <Login
        onJoin={(name, boardId) => setSession({ name, boardId })}
      />
    );
  }

  return (
    <div>
      <div className="board-header">
        <h2>
          <span className="logo-icon">💡</span>
          IdeaBoard
          <span className="board-id-badge">#{session.boardId}</span>
        </h2>
        <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
          <span style={{ fontSize: 13, color: "var(--text-muted)" }}>
            👤 {session.name}
          </span>
          <button
            className="btn btn-ghost btn-sm"
            onClick={() => setSession(null)}
          >
            Leave
          </button>
        </div>
      </div>
      <Board boardId={session.boardId} userId={session.name} />
    </div>
  );
}
