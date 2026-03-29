import {
  BadRequestException,
  Injectable,
  Logger
} from "@nestjs/common";
import {
  normalizeBarcode,
  normalizeBarcodes
} from "@media-library/provider-sdk";
import type {
  BarcodeLookupCandidate,
  BarcodeLookupFailure,
  BarcodeLookupFailureCode,
  BarcodeLookupFallback,
  BarcodeLookupLocalCandidate,
  BarcodeLookupProviderCandidate,
  BarcodeLookupRequest,
  BarcodeLookupResponse,
  LibraryEntry,
  MediaRecord,
  MediaType,
  NormalizedSearchResult,
  ProviderName
} from "@media-library/types";
import { LibraryService } from "../library/library.service";
import { MediaService } from "../media/media.service";
import { ScanLogRepository } from "../media/repositories/scan-log.repository";
import { ProviderRegistryService } from "../providers/provider-registry.service";
import { ProvidersService } from "../providers/providers.service";
import {
  getBarcodeLookupStages,
  getLocalLookupMediaType,
  hasWeakBarcodeCoverage,
  inferBarcodeLookupHint
} from "./barcode-lookup.strategy";

const PROVIDER_RESULT_LIMIT = 10;

interface RankedCandidate {
  candidate: BarcodeLookupCandidate;
  hasLinkedLibraryEntry: boolean;
  sortKey: string;
  sortTier: number;
  stageIndex: number;
  year: number | null;
}

@Injectable()
export class BarcodeService {
  private readonly logger = new Logger(BarcodeService.name);

  constructor(
    private readonly providerRegistry: ProviderRegistryService,
    private readonly providersService: ProvidersService,
    private readonly mediaService: MediaService,
    private readonly libraryService: LibraryService,
    private readonly scanLogRepository: ScanLogRepository
  ) {}

  async lookup(
    userId: string,
    request: BarcodeLookupRequest
  ): Promise<BarcodeLookupResponse> {
    const barcode = normalizeBarcode(request.barcode);
    if (!barcode) {
      throw new BadRequestException(
        "Barcode must contain numeric digits or ISBN characters."
      );
    }

    const hint = inferBarcodeLookupHint(barcode, request.preferredMediaType);
    const localCandidates = await this.buildLocalCandidates(
      userId,
      barcode,
      getLocalLookupMediaType(request.preferredMediaType)
    );
    const providerLookup = await this.buildProviderCandidates(barcode, hint);
    const candidates = rankCandidates([
      ...localCandidates,
      ...providerLookup.candidates
    ]);
    const mediaType = getResolvedMediaType(candidates, hint);
    const fallback = buildFallback({
      barcode,
      candidates,
      failures: providerLookup.failures,
      hint,
      mediaType
    });

    await this.logLookupAttempt(userId, barcode, candidates[0]?.candidate);

    return {
      barcode,
      mediaType,
      candidates: candidates.map((candidate) => candidate.candidate),
      failures: providerLookup.failures,
      fallback
    };
  }

  private async buildLocalCandidates(
    userId: string,
    barcode: string,
    mediaType?: MediaType
  ): Promise<RankedCandidate[]> {
    const mediaRecords = await this.mediaService.findByBarcodeCandidate(barcode, mediaType);
    if (mediaRecords.length === 0) {
      return [];
    }

    const linkedEntries = await this.libraryService.getEntriesForUserByMediaRecordIds(
      userId,
      mediaRecords.map((record) => record.id)
    );
    const linkedEntriesByMediaRecordId = groupEntriesByMediaRecordId(linkedEntries);

    return mediaRecords.map((record) => {
      const mediaRecordEntries = linkedEntriesByMediaRecordId.get(record.id) ?? [];
      const candidate: BarcodeLookupLocalCandidate = {
        source: "local",
        mediaRecordId: record.id,
        mediaType: record.mediaType,
        title: record.title,
        year: record.year ?? null,
        imageUrl: record.imageUrl ?? null,
        creatorLine: getMediaCreatorLine(record),
        hasLinkedLibraryEntry: mediaRecordEntries.length > 0,
        linkedLibraryEntries: mediaRecordEntries.map((entry) => ({
          entryId: entry.id,
          bucket: entry.bucket
        }))
      };

      return {
        candidate,
        hasLinkedLibraryEntry: candidate.hasLinkedLibraryEntry,
        sortKey: `local:${record.id}`,
        sortTier: 0,
        stageIndex: 0,
        year: candidate.year ?? null
      };
    });
  }

  private async buildProviderCandidates(
    barcode: string,
    hint: ReturnType<typeof inferBarcodeLookupHint>
  ): Promise<{
    candidates: RankedCandidate[];
    failures: BarcodeLookupFailure[];
  }> {
    const failures: BarcodeLookupFailure[] = [];
    const rankedCandidates: RankedCandidate[] = [];
    const seenCandidateKeys = new Set<string>();
    const seenFailures = new Set<string>();

    const stages = getBarcodeLookupStages(hint);
    for (const [stageIndex, stage] of stages.entries()) {
      const availableProviders: ProviderName[] = [];

      for (const providerName of stage.providers) {
        const provider = this.providerRegistry.getProvider(providerName);
        if (
          provider &&
          provider.enabled &&
          provider.capabilities.supportsBarcodeSearch &&
          provider.supportsMediaType(stage.mediaType)
        ) {
          availableProviders.push(providerName);
          continue;
        }

        const failureCode = getMissingProviderFailureCode(providerName, stage.mediaType, provider);
        addFailure(failures, seenFailures, providerName, failureCode);
      }

      if (availableProviders.length === 0) {
        continue;
      }

      const response = await this.providersService.searchByBarcode({
        barcode,
        mediaType: stage.mediaType,
        providers: availableProviders,
        limit: PROVIDER_RESULT_LIMIT
      });

      for (const failure of response.failures) {
        addFailure(
          failures,
          seenFailures,
          failure.provider,
          mapProviderFailureCode(failure.message)
        );
      }

      for (const result of response.results) {
        const candidateKey = `provider:${result.provider}:${result.providerId}`;
        if (seenCandidateKeys.has(candidateKey)) {
          continue;
        }

        seenCandidateKeys.add(candidateKey);
        rankedCandidates.push(
          buildProviderCandidate({
            barcode,
            result,
            stageIndex,
            tier: hasExactProviderBarcodeMatch(barcode, result) ? 1 : stage.rankTier
          })
        );
      }
    }

    return {
      candidates: rankedCandidates,
      failures
    };
  }

  private async logLookupAttempt(
    userId: string,
    barcode: string,
    topCandidate?: BarcodeLookupCandidate
  ): Promise<void> {
    try {
      await this.scanLogRepository.create({
        userId,
        barcode,
        matchedMediaType: topCandidate?.mediaType ?? null,
        matchedProvider:
          topCandidate?.source === "provider" ? topCandidate.provider : null
      });
    } catch (error) {
      const message = error instanceof Error ? error.message : "Unknown error";
      this.logger.warn(`Failed to persist barcode scan log: ${message}`);
    }
  }
}

function buildProviderCandidate(input: {
  barcode: string;
  result: NormalizedSearchResult;
  stageIndex: number;
  tier: number;
}): RankedCandidate {
  const candidate: BarcodeLookupProviderCandidate = {
    source: "provider",
    provider: input.result.provider,
    providerId: input.result.providerId,
    mediaType: input.result.mediaType,
    title: input.result.title,
    year: input.result.year ?? null,
    imageUrl: input.result.imageUrl ?? null,
    creatorLine: input.result.creatorLine ?? null
  };

  return {
    candidate,
    hasLinkedLibraryEntry: false,
    sortKey: `provider:${candidate.provider}:${candidate.providerId}`,
    sortTier: input.tier,
    stageIndex: input.stageIndex + 1,
    year: candidate.year ?? null
  };
}

function rankCandidates(candidates: RankedCandidate[]): RankedCandidate[] {
  return [...candidates].sort((left, right) => {
    if (left.sortTier !== right.sortTier) {
      return left.sortTier - right.sortTier;
    }

    if (left.hasLinkedLibraryEntry !== right.hasLinkedLibraryEntry) {
      return left.hasLinkedLibraryEntry ? -1 : 1;
    }

    if (left.stageIndex !== right.stageIndex) {
      return left.stageIndex - right.stageIndex;
    }

    const titleComparison = left.candidate.title.localeCompare(right.candidate.title);
    if (titleComparison !== 0) {
      return titleComparison;
    }

    if ((left.year ?? 0) !== (right.year ?? 0)) {
      return (right.year ?? 0) - (left.year ?? 0);
    }

    return left.sortKey.localeCompare(right.sortKey);
  });
}

function getResolvedMediaType(
  candidates: RankedCandidate[],
  hint: ReturnType<typeof inferBarcodeLookupHint>
): MediaType | null {
  return (
    candidates[0]?.candidate.mediaType ??
    hint.preferredMediaType ??
    hint.inferredMediaType
  );
}

function buildFallback(input: {
  barcode: string;
  candidates: RankedCandidate[];
  failures: BarcodeLookupFailure[];
  hint: ReturnType<typeof inferBarcodeLookupHint>;
  mediaType: MediaType | null;
}): BarcodeLookupFallback {
  if (input.candidates.length > 0) {
    return {
      reason: "manual_confirmation_required",
      manualQuery: input.candidates[0]?.candidate.title ?? null,
      mediaType: input.mediaType
    };
  }

  if (hasWeakBarcodeCoverage(input.hint.preferredMediaType ?? input.hint.inferredMediaType)) {
    return {
      reason: "weak_barcode_coverage",
      manualQuery: null,
      mediaType: input.mediaType
    };
  }

  if (input.failures.length > 0) {
    return {
      reason: "provider_unavailable",
      manualQuery: input.hint.isIsbnLike ? input.barcode : null,
      mediaType: input.mediaType
    };
  }

  return {
    reason: "no_candidates",
    manualQuery: input.hint.isIsbnLike ? input.barcode : null,
    mediaType: input.mediaType
  };
}

function groupEntriesByMediaRecordId(
  entries: LibraryEntry[]
): Map<string, LibraryEntry[]> {
  const entriesByMediaRecordId = new Map<string, LibraryEntry[]>();

  for (const entry of entries) {
    const existingEntries = entriesByMediaRecordId.get(entry.mediaRecordId) ?? [];
    existingEntries.push(entry);
    entriesByMediaRecordId.set(entry.mediaRecordId, existingEntries);
  }

  return entriesByMediaRecordId;
}

function getMediaCreatorLine(media: MediaRecord): string | null {
  switch (media.mediaType) {
    case "album":
      return media.details.artists.join(", ");
    case "book":
      return media.details.authors.join(", ");
    case "movie":
      return media.details.directors?.join(", ") ?? null;
    case "tv":
      return media.details.creators?.join(", ") ?? null;
    case "game":
      return media.details.developers?.join(", ") ?? null;
  }
}

function hasExactProviderBarcodeMatch(
  barcode: string,
  result: NormalizedSearchResult
): boolean {
  return normalizeBarcodes(result.barcodeCandidates).includes(barcode);
}

function addFailure(
  failures: BarcodeLookupFailure[],
  seenFailures: Set<string>,
  provider: ProviderName,
  code: BarcodeLookupFailureCode
): void {
  const failureKey = `${provider}:${code}`;
  if (seenFailures.has(failureKey)) {
    return;
  }

  seenFailures.add(failureKey);
  failures.push({ provider, code });
}

function getMissingProviderFailureCode(
  providerName: ProviderName,
  mediaType: MediaType,
  provider:
    | ReturnType<ProviderRegistryService["getProvider"]>
    | null
): BarcodeLookupFailureCode {
  if (!provider) {
    return "unsupported";
  }

  if (!provider.enabled) {
    return "unavailable";
  }

  if (
    !provider.capabilities.supportsBarcodeSearch ||
    !provider.supportsMediaType(mediaType)
  ) {
    return "unsupported";
  }

  return "unavailable";
}

function mapProviderFailureCode(message: string): BarcodeLookupFailureCode {
  const normalizedMessage = message.toLowerCase();

  if (normalizedMessage.includes("timed out")) {
    return "timeout";
  }

  if (normalizedMessage.includes("unsupported")) {
    return "unsupported";
  }

  if (
    normalizedMessage.includes("unavailable") ||
    normalizedMessage.includes("network request failed")
  ) {
    return "unavailable";
  }

  return normalizedMessage.includes("status")
    ? "invalid_response"
    : "unavailable";
}
