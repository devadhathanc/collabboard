import { useState, useCallback } from "react";
import { submitIdea } from "../api/ideas";
import type { Idea } from "../types";

interface Props {
  boardId: string;
  userId: string;
  ideas: Idea[];
  onIdeaSubmitted: (idea: Idea) => void;
}

export function IdeaInput({ boardId, userId, ideas, onIdeaSubmitted }: Props) {
  const [text, setText] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [lastSubmitted, setLastSubmitted] = useState("");

  const handleSubmit = useCallback(async () => {
    const trimmed = text.trim();
    if (!trimmed || submitting) return;
    setSubmitting(true);
    try {
      const idea = await submitIdea(boardId, trimmed, userId);
      onIdeaSubmitted(idea);
      setLastSubmitted(trimmed);
      setText("");
    } catch {
      // ignore
    } finally {
      setSubmitting(false);
    }
  }, [text, submitting, boardId, userId, onIdeaSubmitted]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  };

  return (
    <div className="idea-input-panel">
      <div className="idea-form">
        <textarea
          id="idea-textarea"
          className="input idea-textarea"
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Type your idea and press Enter..."
          rows={3}
          disabled={submitting}
        />
        <button
          id="submit-idea-btn"
          className="btn btn-primary btn-full"
          onClick={handleSubmit}
          disabled={!text.trim() || submitting}
        >
          {submitting ? "Submitting..." : "Submit Idea ↵"}
        </button>
        {lastSubmitted && (
          <p className="idea-success">✓ Submitted: "{lastSubmitted}"</p>
        )}
      </div>

      <div className="idea-feed">
        <div className="idea-feed-header">
          <span>Recent Ideas</span>
          <span className="idea-feed-count">{ideas.length}</span>
        </div>
        {ideas.length === 0 && (
          <div className="idea-feed-empty">No ideas yet. Be the first!</div>
        )}
        {ideas.map((idea) => (
          <div key={idea.id} className="idea-feed-item">
            <div className="idea-feed-content">
              <span className="idea-feed-text">{idea.text}</span>
              <span className="idea-feed-by">{idea.created_by}</span>
            </div>
            {idea.count > 1 && (
              <span className="idea-feed-badge">×{idea.count}</span>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
