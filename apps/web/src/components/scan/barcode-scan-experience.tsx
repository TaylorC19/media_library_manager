"use client";

import type {
  BarcodeLookupResponse,
  LibraryBucket,
} from "@media-library/types";
import { useTranslations } from "next-intl";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { getLocalizedApiErrorMessageFromResponse } from "../../i18n/errors";
import { Link, useRouter } from "../../i18n/navigation";
import {
  addLocalCandidateToBucket,
  importProviderCandidateToBucket,
  lookupBarcode
} from "../../lib/barcode-api";
import { ScanCandidateList, getCandidateKey } from "./scan-candidate-list";
import {
  type CameraPermissionState,
  ScanStatusBanner,
  type ScanViewState
} from "./scan-status-banner";
import { SelectedScanCandidatePanel } from "./selected-scan-candidate-panel";

const RECENT_SCAN_COOLDOWN_MS = 4000;
const SUPPORTED_FORMATS = ["ean_13", "ean_8", "upc_a", "upc_e"];

interface BarcodeDetectorLike {
  detect(source: HTMLVideoElement): Promise<Array<{ rawValue?: string | null }>>;
}

interface BarcodeDetectorLikeConstructor {
  new (options?: { formats?: string[] }): BarcodeDetectorLike;
  getSupportedFormats?: () => Promise<string[]>;
}

export function BarcodeScanExperience() {
  const router = useRouter();
  const tErrors = useTranslations("errors");
  const tScan = useTranslations("scan");
  const [scanState, setScanState] = useState<ScanViewState>("idle");
  const [permissionState, setPermissionState] =
    useState<CameraPermissionState>("prompt");
  const [lookupResponse, setLookupResponse] = useState<BarcodeLookupResponse | null>(
    null
  );
  const [selectedCandidateKey, setSelectedCandidateKey] = useState<string | null>(
    null
  );
  const [scannedBarcode, setScannedBarcode] = useState<string | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [actionErrorMessage, setActionErrorMessage] = useState<string | null>(null);
  const [activeBucket, setActiveBucket] = useState<LibraryBucket | null>(null);
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const detectorRef = useRef<BarcodeDetectorLike | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const frameRef = useRef<number | null>(null);
  const scanSessionRef = useRef(0);
  const decodeLockedRef = useRef(false);
  const lookupInFlightRef = useRef(false);
  const recentScanRef = useRef<{ barcode: string; timestamp: number } | null>(null);

  const selectedCandidate = useMemo(() => {
    if (!lookupResponse || !selectedCandidateKey) {
      return null;
    }

    return (
      lookupResponse.candidates.find(
        (candidate) => getCandidateKey(candidate) === selectedCandidateKey
      ) ?? null
    );
  }, [lookupResponse, selectedCandidateKey]);

  const manualSearchHref = useMemo(() => {
    const query =
      lookupResponse?.fallback?.manualQuery?.trim() ||
      scannedBarcode?.trim() ||
      "";
    const mediaType =
      lookupResponse?.fallback?.mediaType ?? lookupResponse?.mediaType ?? null;
    const params = new URLSearchParams();

    if (query.length > 0) {
      params.set("q", query);
    }

    if (mediaType) {
      params.set("mediaType", mediaType);
    }

    const suffix = params.toString();
    return suffix.length > 0 ? `/search?${suffix}` : "/search";
  }, [lookupResponse, scannedBarcode]);

  const shouldLeadWithManualSearch =
    scanState === "permissionDenied" ||
    scanState === "cameraUnavailable" ||
    scanState === "lookupEmpty";

  const stopScanner = useCallback(() => {
    scanSessionRef.current += 1;

    if (frameRef.current !== null) {
      window.cancelAnimationFrame(frameRef.current);
      frameRef.current = null;
    }

    detectorRef.current = null;

    if (streamRef.current) {
      for (const track of streamRef.current.getTracks()) {
        track.stop();
      }

      streamRef.current = null;
    }

    if (videoRef.current) {
      videoRef.current.pause();
      videoRef.current.srcObject = null;
    }
  }, []);

  const handleLookup = useCallback(
    async (barcode: string) => {
      lookupInFlightRef.current = true;
      setLookupResponse(null);
      setSelectedCandidateKey(null);
      setActionErrorMessage(null);
      setErrorMessage(null);
      setScanState("lookingUp");

      try {
        const result = await lookupBarcode({ barcode });

        if (!result.ok) {
          setErrorMessage(
            await getLocalizedApiErrorMessageFromResponse(result.response, tErrors)
          );
          setScanState("error");
          return;
        }

        setLookupResponse(result.data);
        setScanState(result.data.candidates.length > 0 ? "results" : "lookupEmpty");
      } catch {
        setErrorMessage(tErrors("apiUnavailable"));
        setScanState("error");
      } finally {
        lookupInFlightRef.current = false;
      }
    },
    [tErrors]
  );

  const handleDecodedBarcode = useCallback(
    async (rawValue: string) => {
      const barcode = normalizeDecodedBarcode(rawValue);
      const recentScan = recentScanRef.current;

      if (!barcode || decodeLockedRef.current || lookupInFlightRef.current) {
        return;
      }

      if (
        recentScan &&
        recentScan.barcode === barcode &&
        Date.now() - recentScan.timestamp < RECENT_SCAN_COOLDOWN_MS
      ) {
        return;
      }

      recentScanRef.current = {
        barcode,
        timestamp: Date.now()
      };
      decodeLockedRef.current = true;
      setScannedBarcode(barcode);
      setErrorMessage(null);
      setActionErrorMessage(null);
      setScanState("decoded");
      stopScanner();
      await handleLookup(barcode);
    },
    [handleLookup, stopScanner]
  );

  const startScanner = useCallback(async () => {
    stopScanner();
    decodeLockedRef.current = false;
    setLookupResponse(null);
    setSelectedCandidateKey(null);
    setScannedBarcode(null);
    setActionErrorMessage(null);
    setErrorMessage(null);
    setPermissionState("prompt");

    const barcodeDetector = getBarcodeDetectorConstructor();
    if (!navigator.mediaDevices?.getUserMedia || !barcodeDetector) {
      setPermissionState("unsupported");
      setErrorMessage(tScan("errors.cameraUnavailable"));
      setScanState("cameraUnavailable");
      return;
    }

    if (!(await supportsRequestedFormats(barcodeDetector))) {
      setPermissionState("unsupported");
      setErrorMessage(tScan("errors.cameraUnavailable"));
      setScanState("cameraUnavailable");
      return;
    }

    setScanState("requestingPermission");

    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: false,
        video: {
          facingMode: {
            ideal: "environment"
          }
        }
      });
      const sessionId = scanSessionRef.current + 1;
      scanSessionRef.current = sessionId;
      streamRef.current = stream;

      if (!videoRef.current) {
        setErrorMessage(tScan("errors.cameraUnavailable"));
        setScanState("cameraUnavailable");
        stopScanner();
        return;
      }

      videoRef.current.srcObject = stream;
      await videoRef.current.play();
      detectorRef.current = new barcodeDetector({
        formats: SUPPORTED_FORMATS
      });
      setPermissionState("granted");
      setScanState("ready");

      const scanNextFrame = async () => {
        if (scanSessionRef.current !== sessionId || !videoRef.current || !detectorRef.current) {
          return;
        }

        try {
          if (videoRef.current.readyState >= HTMLMediaElement.HAVE_CURRENT_DATA) {
            const detections = await detectorRef.current.detect(videoRef.current);
            const detectedBarcode = detections
              .map((detection) => detection.rawValue ?? "")
              .find((value) => normalizeDecodedBarcode(value).length > 0);

            if (detectedBarcode) {
              await handleDecodedBarcode(detectedBarcode);
              return;
            }
          }
        } catch {
          // Ignore transient detector errors and keep polling frames.
        }

        if (scanSessionRef.current === sessionId) {
          frameRef.current = window.requestAnimationFrame(() => {
            void scanNextFrame();
          });
        }
      };

      frameRef.current = window.requestAnimationFrame(() => {
        void scanNextFrame();
      });
    } catch (error) {
      if (isPermissionDeniedError(error)) {
        setPermissionState("denied");
        setErrorMessage(tScan("errors.permissionDenied"));
        setScanState("permissionDenied");
        return;
      }

      setErrorMessage(tScan("errors.cameraUnavailable"));
      setScanState("cameraUnavailable");
    }
  }, [handleDecodedBarcode, stopScanner, tScan]);

  useEffect(() => {
    let isActive = true;
    let permissionStatus: PermissionStatus | null = null;

    if (!navigator.permissions?.query) {
      return;
    }

    void navigator.permissions
      .query({ name: "camera" as PermissionName })
      .then((status) => {
        if (!isActive) {
          return;
        }

        permissionStatus = status;
        const applyState = () => {
          setPermissionState((status.state as CameraPermissionState) ?? "prompt");
        };

        applyState();
        status.onchange = applyState;
      })
      .catch(() => {
        setPermissionState("prompt");
      });

    return () => {
      isActive = false;

      if (permissionStatus) {
        permissionStatus.onchange = null;
      }
    };
  }, []);

  useEffect(() => {
    void startScanner();

    return () => {
      stopScanner();
    };
  }, [startScanner, stopScanner]);

  async function handleAdd(bucket: LibraryBucket) {
    if (!selectedCandidate) {
      return;
    }

    setActiveBucket(bucket);
    setActionErrorMessage(null);
    setScanState("saving");

    try {
      if (
        selectedCandidate.source === "local" &&
        selectedCandidate.linkedLibraryEntries.some((entry) => entry.bucket === bucket)
      ) {
        setScanState("results");
        return;
      }

      if (selectedCandidate.source === "provider") {
        const result = await importProviderCandidateToBucket({
          barcode: scannedBarcode ?? "",
          bucket,
          candidate: selectedCandidate
        });

        if (!result.ok) {
          setActionErrorMessage(
            await getLocalizedApiErrorMessageFromResponse(result.response, tErrors)
          );
          setScanState("results");
          return;
        }

        const entryId = result.data.libraryEntry?.entry.id ?? null;
        if (!entryId) {
          setActionErrorMessage(tErrors("importMissingEntry"));
          setScanState("results");
          return;
        }

        router.push(`/library/${entryId}`);
      } else {
        const result = await addLocalCandidateToBucket({
          barcode: scannedBarcode ?? "",
          bucket,
          candidate: selectedCandidate
        });

        if (!result.ok) {
          setActionErrorMessage(
            await getLocalizedApiErrorMessageFromResponse(result.response, tErrors)
          );
          setScanState("results");
          return;
        }

        router.push(`/library/${result.data.entry.id}`);
      }

      router.refresh();
    } catch {
      setActionErrorMessage(tErrors("apiUnavailable"));
      setScanState("results");
    } finally {
      setActiveBucket(null);
    }
  }

  function handleSelectCandidate(candidateKey: string) {
    setSelectedCandidateKey(candidateKey);
    setActionErrorMessage(null);
  }

  function handleRetryScan() {
    void startScanner();
  }

  return (
    <div className="space-y-8">
      <section className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6 sm:p-8">
        <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
          {tScan("pageLabel")}
        </p>
        <h1 className="mt-3 text-3xl font-semibold text-white">{tScan("title")}</h1>
        <p className="mt-3 max-w-3xl text-sm text-slate-300">{tScan("description")}</p>
        <p className="mt-4 text-sm text-slate-400">{tScan("supportedFormats")}</p>
      </section>

      <ScanStatusBanner
        errorMessage={errorMessage}
        permissionState={permissionState}
        scanState={scanState}
      />

      <section className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
              {tScan("cameraLabel")}
            </p>
            <h2 className="mt-2 text-2xl font-semibold text-white">
              {tScan("cameraTitle")}
            </h2>
          </div>

          <div className="flex flex-col gap-3 sm:flex-row">
            <button
              className="rounded-2xl border border-slate-700 px-4 py-3 text-sm font-semibold text-slate-100 transition hover:border-sky-400 hover:text-white"
              onClick={handleRetryScan}
              type="button"
            >
              {scanState === "permissionDenied" || scanState === "cameraUnavailable"
                ? tScan("actions.retryCamera")
                : tScan("actions.retryScan")}
            </button>
            <Link
              className="rounded-2xl border border-slate-700 px-4 py-3 text-center text-sm font-semibold text-slate-100 transition hover:border-sky-400 hover:text-white"
              href={manualSearchHref}
            >
              {tScan("actions.manualSearch")}
            </Link>
          </div>
        </div>

        <div className="mt-6 overflow-hidden rounded-3xl border border-slate-800 bg-black">
          {scanState === "ready" || scanState === "requestingPermission" ? (
            <video
              autoPlay
              className="aspect-[4/3] w-full object-cover"
              muted
              playsInline
              ref={videoRef}
            />
          ) : (
            <div className="flex aspect-[4/3] items-center justify-center px-6 text-center text-sm text-slate-400">
              {scanState === "lookingUp" || scanState === "saving"
                ? tScan("cameraPaused")
                : scanState === "permissionDenied"
                  ? tScan("cameraPermissionDenied")
                  : scanState === "cameraUnavailable"
                    ? tScan("cameraUnavailable")
                    : tScan("cameraReadyPlaceholder")}
            </div>
          )}
        </div>
      </section>

      <section className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
        <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
          {tScan("barcodeLabel")}
        </p>
        <h2 className="mt-2 text-2xl font-semibold text-white">
          {tScan("barcodeTitle")}
        </h2>
        <div className="mt-4 rounded-2xl border border-slate-800 bg-slate-900 px-4 py-4 text-lg font-medium tracking-[0.2em] text-white">
          {scannedBarcode ?? tScan("barcodeEmpty")}
        </div>
      </section>

      {lookupResponse?.candidates.length ? (
        <ScanCandidateList
          candidates={lookupResponse.candidates}
          onSelect={handleSelectCandidate}
          selectedCandidateKey={selectedCandidateKey}
        />
      ) : null}

      <SelectedScanCandidatePanel
        activeBucket={activeBucket}
        candidate={selectedCandidate}
        errorMessage={actionErrorMessage}
        onAdd={handleAdd}
      />

      <section
        className={`rounded-3xl border p-6 ${
          shouldLeadWithManualSearch
            ? "border-slate-800 bg-slate-950/70"
            : "border-dashed border-slate-800 bg-slate-950/40"
        }`}
      >
        <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
          {tScan("manualSearchLabel")}
        </p>
        <h2 className="mt-2 text-2xl font-semibold text-white">
          {shouldLeadWithManualSearch
            ? tScan("manualSearchPrimaryTitle")
            : tScan("manualSearchSecondaryTitle")}
        </h2>
        <p className="mt-2 text-sm text-slate-300">
          {shouldLeadWithManualSearch
            ? tScan("manualSearchPrimaryDescription")
            : tScan("manualSearchSecondaryDescription")}
        </p>
        <div className="mt-4 flex flex-col gap-3 sm:flex-row">
          <Link
            className="rounded-2xl border border-sky-400/40 bg-sky-400/10 px-4 py-3 text-center text-sm font-semibold text-sky-100 transition hover:border-sky-300 hover:text-white"
            href={manualSearchHref}
          >
            {tScan("actions.manualSearch")}
          </Link>
          <button
            className="rounded-2xl border border-slate-700 px-4 py-3 text-sm font-semibold text-slate-100 transition hover:border-sky-400 hover:text-white"
            onClick={handleRetryScan}
            type="button"
          >
            {tScan("actions.retryScan")}
          </button>
        </div>
      </section>
    </div>
  );
}

function getBarcodeDetectorConstructor(): BarcodeDetectorLikeConstructor | null {
  if (typeof window === "undefined") {
    return null;
  }

  return (
    (window as Window & { BarcodeDetector?: BarcodeDetectorLikeConstructor })
      .BarcodeDetector ?? null
  );
}

async function supportsRequestedFormats(
  barcodeDetector: BarcodeDetectorLikeConstructor
): Promise<boolean> {
  if (!barcodeDetector.getSupportedFormats) {
    return true;
  }

  const supportedFormats = await barcodeDetector.getSupportedFormats();
  return SUPPORTED_FORMATS.every((format) => supportedFormats.includes(format));
}

function normalizeDecodedBarcode(value: string): string {
  return value.replace(/[^0-9xX]/g, "").toUpperCase();
}

function isPermissionDeniedError(error: unknown): boolean {
  return (
    error instanceof DOMException &&
    (error.name === "NotAllowedError" || error.name === "SecurityError")
  );
}
