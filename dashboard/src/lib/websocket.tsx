"use client";

import { createContext, useContext, useEffect, useRef, useState, useCallback, ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useAuthStore } from "./store";
import { toast } from "sonner";
import type { SecurityEvent } from "./api";

type MessageType = "event" | "alert" | "server_status" | "server_created" | "server_deleted" | "dashboard";

interface WebSocketMessage {
  type: MessageType;
  payload: Record<string, unknown>;
}

interface EventPayload {
  server_id: string;
  hostname: string;
  events: SecurityEvent[];
}

interface WebSocketContextType {
  isConnected: boolean;
  lastMessage: WebSocketMessage | null;
}

const WebSocketContext = createContext<WebSocketContextType>({
  isConnected: false,
  lastMessage: null,
});

export function useWebSocket() {
  return useContext(WebSocketContext);
}

interface WebSocketProviderProps {
  children: ReactNode;
}

export function WebSocketProvider({ children }: WebSocketProviderProps) {
  const [isConnected, setIsConnected] = useState(false);
  const [lastMessage, setLastMessage] = useState<WebSocketMessage | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const queryClient = useQueryClient();
  const token = useAuthStore((s) => s.token);

  const connect = useCallback(() => {
    if (!token || wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    // Get WebSocket URL from API URL
    const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
    const wsProtocol = apiUrl.startsWith("https") ? "wss" : "ws";
    const wsHost = apiUrl.replace(/^https?:\/\//, "");
    const wsUrl = `${wsProtocol}://${wsHost}/ws?token=${token}`;

    try {
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        console.log("WebSocket connected");
        setIsConnected(true);
        reconnectAttemptsRef.current = 0;
      };

      ws.onclose = (event) => {
        console.log("WebSocket closed:", event.code, event.reason);
        setIsConnected(false);
        wsRef.current = null;

        // Reconnect with exponential backoff
        if (token && reconnectAttemptsRef.current < 10) {
          const delay = Math.min(1000 * Math.pow(2, reconnectAttemptsRef.current), 30000);
          reconnectAttemptsRef.current++;
          console.log(`Reconnecting in ${delay}ms (attempt ${reconnectAttemptsRef.current})`);
          reconnectTimeoutRef.current = setTimeout(connect, delay);
        }
      };

      ws.onerror = (error) => {
        console.error("WebSocket error:", error);
      };

      ws.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data);
          setLastMessage(message);
          handleMessage(message);
        } catch (error) {
          console.error("Failed to parse WebSocket message:", error);
        }
      };
    } catch (error) {
      console.error("Failed to create WebSocket:", error);
    }
  }, [token]);

  const handleMessage = useCallback((message: WebSocketMessage) => {
    switch (message.type) {
      case "event": {
        // Add events directly to cache for instant updates
        const eventPayload = message.payload as unknown as EventPayload;
        if (eventPayload.events && eventPayload.events.length > 0) {
          const serverId = eventPayload.server_id;

          // Update all server-events queries for this server
          queryClient.setQueriesData(
            { queryKey: ["server-events", serverId] },
            (oldData: { events: SecurityEvent[]; total: number } | undefined) => {
              if (!oldData) return oldData;
              // Prepend new events to the list
              const newEvents = [...eventPayload.events, ...oldData.events];
              return {
                events: newEvents,
                total: oldData.total + eventPayload.events.length,
              };
            }
          );
        }
        // Also invalidate dashboard summary for updated counts
        queryClient.invalidateQueries({ queryKey: ["dashboard-summary"] });
        break;
      }

      case "alert":
        // Invalidate alerts queries
        queryClient.invalidateQueries({ queryKey: ["alerts"] });
        queryClient.invalidateQueries({ queryKey: ["dashboard-summary"] });
        // Show toast for new alerts
        if (message.payload.title && !message.payload.status) {
          toast.warning(`New Alert: ${message.payload.title}`, {
            description: message.payload.description as string,
          });
        }
        break;

      case "server_status":
        // Invalidate server queries
        queryClient.invalidateQueries({ queryKey: ["servers"] });
        queryClient.invalidateQueries({ queryKey: ["server"] });
        queryClient.invalidateQueries({ queryKey: ["dashboard-summary"] });
        break;

      case "server_created":
        // Invalidate servers list
        queryClient.invalidateQueries({ queryKey: ["servers"] });
        queryClient.invalidateQueries({ queryKey: ["dashboard-summary"] });
        break;

      case "server_deleted":
        // Invalidate servers list
        queryClient.invalidateQueries({ queryKey: ["servers"] });
        queryClient.invalidateQueries({ queryKey: ["dashboard-summary"] });
        break;

      case "dashboard":
        // General dashboard refresh
        queryClient.invalidateQueries({ queryKey: ["dashboard-summary"] });
        queryClient.invalidateQueries({ queryKey: ["alerts"] });
        queryClient.invalidateQueries({ queryKey: ["servers"] });
        break;

      default:
        console.log("Unknown message type:", message.type);
    }
  }, [queryClient]);

  useEffect(() => {
    if (token) {
      connect();
    }

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [token, connect]);

  return (
    <WebSocketContext.Provider value={{ isConnected, lastMessage }}>
      {children}
    </WebSocketContext.Provider>
  );
}
