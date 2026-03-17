import { Injectable } from "@nestjs/common";
import { InjectConnection } from "@nestjs/mongoose";
import type { Connection } from "mongoose";
import type { ApiHealthStatus } from "./health.types";

const mongoStatuses: Record<number, ApiHealthStatus["mongo"]["status"]> = {
  0: "disconnected",
  1: "connected",
  2: "connecting",
  3: "disconnecting"
};

@Injectable()
export class HealthService {
  constructor(@InjectConnection() private readonly connection: Connection) {}

  getStatus(): ApiHealthStatus {
    const readyState = this.connection.readyState;
    const mongoStatus = mongoStatuses[readyState] ?? "disconnected";

    return {
      service: "api",
      status: readyState === 1 ? "ok" : "degraded",
      timestamp: new Date().toISOString(),
      uptimeSeconds: Math.round(process.uptime()),
      mongo: {
        readyState,
        status: mongoStatus
      }
    };
  }
}
