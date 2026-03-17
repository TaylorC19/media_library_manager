import { Controller, Get } from "@nestjs/common";
import { Public } from "../auth/decorators/public.decorator";
import { HealthService } from "./health.service";
import type { ApiHealthStatus } from "./health.types";

@Controller("health")
export class HealthController {
  constructor(private readonly healthService: HealthService) {}

  @Public()
  @Get()
  getHealth(): ApiHealthStatus {
    return this.healthService.getStatus();
  }
}
