import type {
  NormalizedMediaRecordInput,
  NormalizedSearchResult
} from "@media-library/types";

export function normalizeSearchResults(
  results: NormalizedSearchResult[]
): NormalizedSearchResult[] {
  return results.map((result) => ({
    ...result,
    title: result.title.trim()
  }));
}

export function normalizeMediaRecord(
  record: NormalizedMediaRecordInput
): NormalizedMediaRecordInput {
  return {
    ...record,
    title: record.title.trim()
  };
}
