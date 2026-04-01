import { Injectable } from "@nestjs/common";
import { ProviderHttpError } from "./provider-http.service";
import { ProviderReliabilityService } from "../provider-reliability.service";

@Injectable()
export class MusicBrainzThrottleService {
  private queue: Promise<void> = Promise.resolve();
  private nextAvailableAt = 0;

  constructor(
    private readonly providerReliabilityService: ProviderReliabilityService
  ) {}

  schedule<T>(task: () => Promise<T>): Promise<T> {
    const scheduledTask = this.queue.then(async () => {
      const settings =
        this.providerReliabilityService.getMusicBrainzThrottleSettings();
      const waitMs = Math.max(0, this.nextAvailableAt - Date.now());

      if (waitMs > 0) {
        await delay(waitMs);
      }

      try {
        const result = await task();
        this.nextAvailableAt = Date.now() + settings.minIntervalMs;
        return result;
      } catch (error) {
        this.nextAvailableAt =
          Date.now() + resolveCooldownMs(error, settings) + settings.minIntervalMs;
        throw error;
      }
    });

    this.queue = scheduledTask.then(
      () => undefined,
      () => undefined
    );

    return scheduledTask;
  }
}

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
}

function resolveCooldownMs(
  error: unknown,
  settings: ReturnType<ProviderReliabilityService["getMusicBrainzThrottleSettings"]>
): number {
  if (error instanceof ProviderHttpError) {
    if (error.code === "rate_limited") {
      return settings.rateLimitCooldownMs;
    }

    if (
      error.code === "timeout" ||
      error.code === "network" ||
      error.code === "upstream"
    ) {
      return settings.retryableCooldownMs;
    }
  }

  return 0;
}
