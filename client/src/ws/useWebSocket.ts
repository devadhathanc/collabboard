import { useEffect, useRef, useCallback } from "react";
import type { WsMessage } from "../types";

const WS_URL: string = (import.meta.env.VITE_WS_URL as string) || "ws://localhost:8090";


interface Props {
  boardId: string;
  userId: string;
  onMessage: (msg: WsMessage) => void;
  onReconnect: () => void; // called after reconnect so caller can re-fetch
}

export function useWebSocket({ boardId, userId, onMessage, onReconnect }: Props) {
  const ws = useRef<WebSocket | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout>>(undefined);
  const onMessageRef = useRef(onMessage);
  const onReconnectRef = useRef(onReconnect);
  const unmounted = useRef(false);

  onMessageRef.current = onMessage;
  onReconnectRef.current = onReconnect;

  const connect = useCallback(() => {
    if (unmounted.current) return;

    const url = `${WS_URL}/ws?board_id=${encodeURIComponent(boardId)}&user_id=${encodeURIComponent(userId)}`;
    const socket = new WebSocket(url);
    ws.current = socket;

    socket.onmessage = (e) => {
      try {
        const msg: WsMessage = JSON.parse(e.data);
        onMessageRef.current(msg);
      } catch {
        // ignore malformed frames
      }
    };

    socket.onclose = () => {
      if (unmounted.current) return;
      // On disconnect, schedule reconnect and then re-fetch full state.
      reconnectTimer.current = setTimeout(() => {
        connect();
        onReconnectRef.current(); // caller re-fetches full board state
      }, 2000);
    };

    socket.onerror = () => socket.close();
  }, [boardId, userId]);

  useEffect(() => {
    unmounted.current = false;
    connect();
    return () => {
      unmounted.current = true;
      clearTimeout(reconnectTimer.current);
      ws.current?.close();
    };
  }, [connect]);
}
