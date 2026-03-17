export interface ApiHealthStatus {
  service: "api";
  status: "ok" | "degraded";
  timestamp: string;
  uptimeSeconds: number;
  mongo: {
    readyState: number;
    status: "connected" | "connecting" | "disconnected" | "disconnecting";
  };
}
