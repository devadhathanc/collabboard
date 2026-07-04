import { useEffect, useRef, useCallback, useState } from "react";
import type { WsEvent } from "../types";

interface UseWsOptions {
  boardId: string;
  userId: string;
  onEvent: (evt: WsEvent) => void;
}

const RECONNECT_BASE = 1000;
const RECONNECT_MAX = 30000;

export function useWebSocket({ boardId, userId, onEvent }: UseWsOptions) {
  const wsRef = useRef<WebSocket | null>(null);
  const lastSeqRef = useRef(0);
  const retriesRef = useRef(0);
  const [connected, setConnected] = useState(false);

  const connect = useCallback(() => {
    const proto = location.protocol === "https:" ? "wss:" : "ws:";
    const host = location.host;
    const url = `${proto}//${host}/ws?board_id=${boardId}&user_id=${userId}&last_seq=${lastSeqRef.current}`;

    const ws = new WebSocket(url);
    wsRef.current = ws;

    ws.onopen = () => {
      setConnected(true);
      retriesRef.current = 0;
    };

    ws.onmessage = (msg) => {
      try {
        const evt: WsEvent = JSON.parse(msg.data);
        if (evt.sequence_id > lastSeqRef.current) {
          lastSeqRef.current = evt.sequence_id;
        }
        onEvent(evt);
      } catch {
        // ignore parse errors
      }
    };

    ws.onclose = () => {
      setConnected(false);
      wsRef.current = null;
      scheduleReconnect();
    };

    ws.onerror = () => {
      ws.close();
    };
  }, [boardId, userId, onEvent]);

  const scheduleReconnect = useCallback(() => {
    retriesRef.current++;
    const delay = Math.min(
      RECONNECT_BASE * Math.pow(2, retriesRef.current - 1) + Math.random() * 1000,
      RECONNECT_MAX,
    );
    setTimeout(connect, delay);
  }, [connect]);

  useEffect(() => {
    connect();
    return () => {
      wsRef.current?.close();
    };
  }, [connect]);

  const send = useCallback((data: unknown) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data));
    }
  }, []);

  return { connected, send, lastSeq: lastSeqRef };
}
