export interface ApiErrorResponse {
    message: string;
    code?: string;
}
export interface HealthStatus {
    service: "api";
    status: "ok" | "degraded";
    timestamp: string;
    uptimeSeconds: number;
    mongo: {
        readyState: number;
        status: "connected" | "connecting" | "disconnected" | "disconnecting";
    };
}
