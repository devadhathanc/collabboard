import { useState } from "react";

interface Props {
  onJoin: (name: string, boardId: string) => void;
}

export function Login({ onJoin }: Props) {
  const [name, setName] = useState("");
  const [boardId, setBoardId] = useState("main");
  const [err, setErr] = useState("");

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) { setErr("Please enter your name"); return; }
    if (!boardId.trim()) { setErr("Please enter a session code"); return; }
    onJoin(name.trim(), boardId.trim().toLowerCase().replace(/\s+/g, "-"));
  };

  return (
    <div className="login-container">
      <div className="login-card">
        <div className="login-icon">💡</div>
        <h1>IdeaBoard</h1>
        <p className="subtitle">Real-time collaborative brainstorming</p>
        <form onSubmit={handleSubmit}>
          <div className="input-group">
            <input
              id="name-input"
              className="input"
              placeholder="Your name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              autoFocus
            />
          </div>
          <div className="input-group">
            <input
              id="board-input"
              className="input"
              placeholder="Session code (e.g. team-sprint)"
              value={boardId}
              onChange={(e) => setBoardId(e.target.value)}
            />
          </div>
          <button type="submit" className="btn btn-primary btn-full">
            Join Session →
          </button>
          {err && <p className="error">{err}</p>}
        </form>
        <p className="login-hint">Share the same session code with your team</p>
      </div>
    </div>
  );
}
