import { useState, useEffect, useRef } from "react";
import { searchTasks } from "../api/tasks";
import type { Task } from "../types";

interface Props {
  boardId: string;
  onResults: (tasks: Task[]) => void;
}

export function SearchBar({ boardId, onResults }: Props) {
  const [query, setQuery] = useState("");
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    if (!query.trim()) return;
    clearTimeout(timer.current);
    timer.current = setTimeout(async () => {
      try {
        const results = await searchTasks(boardId, query);
        onResults(results);
      } catch {
        // ignore
      }
    }, 300);
    return () => clearTimeout(timer.current);
  }, [query, boardId, onResults]);

  return (
    <input
      value={query}
      onChange={(e) => setQuery(e.target.value)}
      placeholder="Search tasks..."
      style={{ width: "100%", padding: 8, boxSizing: "border-box", marginBottom: 16 }}
    />
  );
}
