import type {
  BarcodeLookupLocalCandidate,
  BarcodeLookupProviderCandidate,
  BarcodeLookupRequest,
  BarcodeLookupResponse,
  ImportMediaRecordResponse,
  LibraryBucket,
  LibraryEntryResponse
} from "@media-library/types";
import { browserApiFetch } from "./api-client";

type ApiResult<T> = { ok: true; data: T } | { ok: false; response: Response };

export async function lookupBarcode(
  request: BarcodeLookupRequest
): Promise<ApiResult<BarcodeLookupResponse>> {
  const response = await browserApiFetch("/barcode/lookup", {
    method: "POST",
    headers: {
      "content-type": "application/json"
    },
    body: JSON.stringify(request)
  });

  return parseApiResult<BarcodeLookupResponse>(response);
}

export async function importProviderCandidateToBucket(input: {
  barcode: string;
  bucket: LibraryBucket;
  candidate: BarcodeLookupProviderCandidate;
}): Promise<ApiResult<ImportMediaRecordResponse>> {
  const response = await browserApiFetch("/media/import", {
    method: "POST",
    headers: {
      "content-type": "application/json"
    },
    body: JSON.stringify({
      mode: "provider_ref",
      provider: input.candidate.provider,
      providerId: input.candidate.providerId,
      mediaType: input.candidate.mediaType,
      entry: {
        bucket: input.bucket,
        barcode: input.barcode
      }
    })
  });

  return parseApiResult<ImportMediaRecordResponse>(response);
}

export async function addLocalCandidateToBucket(input: {
  barcode: string;
  bucket: LibraryBucket;
  candidate: BarcodeLookupLocalCandidate;
}): Promise<ApiResult<LibraryEntryResponse>> {
  const response = await browserApiFetch("/library", {
    method: "POST",
    headers: {
      "content-type": "application/json"
    },
    body: JSON.stringify({
      mediaRecordId: input.candidate.mediaRecordId,
      bucket: input.bucket,
      barcode: input.barcode
    })
  });

  return parseApiResult<LibraryEntryResponse>(response);
}

async function parseApiResult<T>(response: Response): Promise<ApiResult<T>> {
  if (!response.ok) {
    return {
      ok: false,
      response
    };
  }

  return {
    ok: true,
    data: (await response.json()) as T
  };
}
