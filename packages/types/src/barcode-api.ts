import type { LibraryBucket, MediaType } from "./library.js";
import type { ProviderName } from "./provider.js";

export interface BarcodeLookupRequest {
  barcode: string;
  preferredMediaType?: MediaType;
}

export interface BarcodeLookupLinkedEntry {
  entryId: string;
  bucket: LibraryBucket;
}

interface BarcodeLookupCandidateBase {
  mediaType: MediaType;
  title: string;
  year?: number | null;
  imageUrl?: string | null;
  creatorLine?: string | null;
}

export interface BarcodeLookupLocalCandidate extends BarcodeLookupCandidateBase {
  source: "local";
  mediaRecordId: string;
  hasLinkedLibraryEntry: boolean;
  linkedLibraryEntries: BarcodeLookupLinkedEntry[];
}

export interface BarcodeLookupProviderCandidate extends BarcodeLookupCandidateBase {
  source: "provider";
  provider: ProviderName;
  providerId: string;
}

export type BarcodeLookupCandidate =
  | BarcodeLookupLocalCandidate
  | BarcodeLookupProviderCandidate;

export const barcodeLookupFailureCodes = [
  "unavailable",
  "timeout",
  "unsupported",
  "invalid_response"
] as const;

export type BarcodeLookupFailureCode = (typeof barcodeLookupFailureCodes)[number];

export interface BarcodeLookupFailure {
  provider: ProviderName;
  code: BarcodeLookupFailureCode;
}

export const barcodeLookupFallbackReasons = [
  "no_candidates",
  "weak_barcode_coverage",
  "provider_unavailable",
  "manual_confirmation_required"
] as const;

export type BarcodeLookupFallbackReason =
  (typeof barcodeLookupFallbackReasons)[number];

export interface BarcodeLookupFallback {
  reason: BarcodeLookupFallbackReason;
  manualQuery?: string | null;
  mediaType?: MediaType | null;
}

export interface BarcodeLookupResponse {
  barcode: string;
  mediaType: MediaType | null;
  candidates: BarcodeLookupCandidate[];
  failures: BarcodeLookupFailure[];
  fallback: BarcodeLookupFallback | null;
}
