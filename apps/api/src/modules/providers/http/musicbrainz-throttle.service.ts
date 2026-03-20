import { Injectable } from "@nestjs/common";

@Injectable()
export class MusicBrainzThrottleService {
  private queue: Promise<void> = Promise.resolve();
  private nextAvailableAt = 0;

  schedule<T>(task: () => Promise<T>): Promise<T> {
    const scheduledTask = this.queue.then(async () => {
      const waitMs = Math.max(0, this.nextAvailableAt - Date.now());

      if (waitMs > 0) {
        await delay(waitMs);
      }

      try {
        return await task();
      } finally {
        this.nextAvailableAt = Date.now() + 1000;
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
