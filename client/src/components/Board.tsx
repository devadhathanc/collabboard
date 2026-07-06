import { useState, useCallback, useEffect } from "react";
import { IdeaInput } from "./IdeaInput";
import { IdeaCloud } from "./IdeaCloud";
import type { IdeaLink } from "./IdeaCloud";
import { useWebSocket } from "../ws/useWebSocket";
import { listIdeas, synthesizeIdeas } from "../api/ideas";
import type { Idea, WsMessage } from "../types";

interface Props {
  boardId: string;
  userId: string;
}

export function Board({ boardId, userId }: Props) {
  const [ideas, setIdeas] = useState<Idea[]>([]);
  const [links, setLinks] = useState<IdeaLink[]>([]);
  const [loading, setLoading] = useState(true);
  const [brainstorming, setBrainstorming] = useState(false);
  const [brainstormMsg, setBrainstormMsg] = useState("");

  const fetchIdeas = useCallback(async () => {
    try {
      const data = await listIdeas(boardId);
      setIdeas(data);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [boardId]);

  useEffect(() => { fetchIdeas(); }, [fetchIdeas]);

  const handleWsMessage = useCallback((msg: WsMessage) => {
    setIdeas((prev) => {
      if (msg.type === "idea.added") {
        if (prev.some((i) => i.id === msg.data.id)) return prev;
        return [msg.data, ...prev];
      }
      if (msg.type === "idea.updated") {
        return prev.map((i) => (i.id === msg.data.id ? msg.data : i));
      }
      return prev;
    });
  }, []);

  useWebSocket({ boardId, userId, onMessage: handleWsMessage, onReconnect: fetchIdeas });

  const handleIdeaSubmitted = useCallback((idea: Idea) => {
    setIdeas((prev) => {
      if (prev.some((i) => i.id === idea.id)) return prev;
      return [idea, ...prev];
    });
  }, []);

  const handleBrainstorm = useCallback(async () => {
    if (brainstorming || ideas.length === 0) return;
    setBrainstorming(true);
    setBrainstormMsg("");

    // Anchor = the human idea with the highest vote count.
    // Tie-break by oldest (earliest created_at) — that's the center bubble in the cloud.
    const humanIdeas = ideas.filter((i) => i.created_by !== "AI");
    const anchor = humanIdeas.sort((a, b) => {
      if (b.count !== a.count) return b.count - a.count;
      return new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
    })[0];

    try {
      const res = await synthesizeIdeas(boardId);
      const count = res.added ?? 0;

      // Add ideas immediately from HTTP response — don't wait for WS.
      if (res.ideas && res.ideas.length > 0) {
        setIdeas((prev) => {
          const next = [...prev];
          for (const idea of res.ideas) {
            if (!next.some((i) => i.id === idea.id)) {
              next.unshift(idea);
            }
          }
          return next;
        });

        // Create relationship links: each new AI idea → anchor idea.
        if (anchor) {
          const newLinks = res.ideas.map((idea) => ({
            from: anchor.id,
            to: idea.id,
          }));
          setLinks((prev) => [...prev, ...newLinks]);
        }
      }

      setBrainstormMsg(count > 0 ? `✨ ${count} ideas added!` : "No new ideas generated.");
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "";
      if (msg.includes("501")) {
        setBrainstormMsg("⚠ Add GEMINI_API_KEY to server .env");
      } else {
        setBrainstormMsg("⚠ Brainstorm failed. Try again.");
      }
    } finally {
      setBrainstorming(false);
      setTimeout(() => setBrainstormMsg(""), 4000);
    }
  }, [brainstorming, ideas, boardId]);

  return (
    <div className="board-layout">
      {/* Left: Idea input */}
      <div className="board-panel ideas-panel">
        <div className="panel-header">
          <h3>💡 Enter Ideas</h3>
          <span style={{ fontSize: 12, color: "var(--text-muted)" }}>
            {ideas.length} idea{ideas.length !== 1 ? "s" : ""}
          </span>
        </div>
        <IdeaInput
          boardId={boardId}
          userId={userId}
          ideas={ideas}
          onIdeaSubmitted={handleIdeaSubmitted}
        />
      </div>

      {/* Right: Visualizer */}
      <div className="board-panel visualizer-panel">
        <div className="panel-header">
          <h3>Visualizer</h3>
          <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
            {brainstormMsg && (
              <span className="brainstorm-msg">{brainstormMsg}</span>
            )}
            <button
              id="brainstorm-btn"
              className="btn btn-primary btn-sm"
              onClick={handleBrainstorm}
              disabled={brainstorming || ideas.length === 0}
            >
              {brainstorming ? <><span className="spinner" /> Brainstorming…</> : "Brainstorm"}
            </button>
          </div>
        </div>

        {loading ? (
          <div className="cloud-empty">
            <div style={{ fontSize: 32, marginBottom: 8 }}>⏳</div>
            <p>Loading...</p>
          </div>
        ) : (
          <IdeaCloud ideas={ideas} links={links} />
        )}
      </div>
    </div>
  );
}
