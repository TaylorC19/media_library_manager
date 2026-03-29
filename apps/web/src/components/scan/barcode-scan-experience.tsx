"use client";

import {
  BarcodeFormat,
  BrowserMultiFormatReader,
  type IScannerControls
} from "@zxing/browser";
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
const CAMERA_START_TIMEOUT_MS = 12000;
const SUPPORTED_FORMATS = [
  BarcodeFormat.EAN_13,
  BarcodeFormat.EAN_8,
  BarcodeFormat.UPC_A,
  BarcodeFormat.UPC_E
];

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
  const readerRef = useRef<BrowserMultiFormatReader | null>(null);
  const scannerControlsRef = useRef<IScannerControls | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
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
    scannerControlsRef.current?.stop();
    scannerControlsRef.current = null;
    readerRef.current = null;
    BrowserMultiFormatReader.releaseAllStreams();

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

    if (!window.isSecureContext) {
      setErrorMessage(tScan("errors.secureContextRequired"));
      setScanState("cameraUnavailable");
      return;
    }

    if (!navigator.mediaDevices?.getUserMedia) {
      setPermissionState("unsupported");
      setErrorMessage(tScan("errors.cameraUnavailable"));
      setScanState("cameraUnavailable");
      return;
    }

    setScanState("requestingPermission");

    try {
      const sessionId = scanSessionRef.current + 1;
      scanSessionRef.current = sessionId;

      if (!videoRef.current) {
        setErrorMessage(tScan("errors.cameraUnavailable"));
        setScanState("cameraUnavailable");
        stopScanner();
        return;
      }

      const reader = new BrowserMultiFormatReader();
      reader.possibleFormats = SUPPORTED_FORMATS;
      readerRef.current = reader;

      const stream = await requestCameraStream(
        {
          audio: false,
          video: {
            facingMode: {
              ideal: "environment"
            }
          }
        },
        CAMERA_START_TIMEOUT_MS
      );

      if (scanSessionRef.current !== sessionId) {
        for (const track of stream.getTracks()) {
          track.stop();
        }
        return;
      }

      streamRef.current = stream;
      setPermissionState("granted");

      const controls = await reader.decodeFromStream(
        stream,
        videoRef.current,
        (result) => {
          if (scanSessionRef.current !== sessionId || !result) {
            return;
          }

          void handleDecodedBarcode(result.getText());
        }
      );

      if (scanSessionRef.current !== sessionId) {
        controls.stop();
        return;
      }

      scannerControlsRef.current = controls;
      setScanState("ready");
    } catch (error) {
      if (isPermissionDeniedError(error)) {
        setPermissionState("denied");
        setErrorMessage(tScan("errors.permissionDenied"));
        setScanState("permissionDenied");
        return;
      }

      setErrorMessage(getCameraErrorMessage(error, tScan));
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

  useEffect(() => stopScanner, [stopScanner]);

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
              {scanState === "idle"
                ? tScan("actions.startCamera")
                : scanState === "permissionDenied" || scanState === "cameraUnavailable"
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

        <div className="relative mt-6 overflow-hidden rounded-3xl border border-slate-800 bg-black">
          <video
            autoPlay
            className={`aspect-[4/3] w-full object-cover transition ${
              scanState === "ready" || scanState === "requestingPermission"
                ? "opacity-100"
                : "opacity-0"
            }`}
            muted
            playsInline
            ref={videoRef}
          />
          {scanState === "ready" || scanState === "requestingPermission" ? null : (
            <div className="absolute inset-0 flex items-center justify-center px-6 text-center text-sm text-slate-400">
              {scanState === "lookingUp" || scanState === "saving"
                ? tScan("cameraPaused")
                : scanState === "idle"
                  ? tScan("cameraStartPrompt")
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

function normalizeDecodedBarcode(value: string): string {
  return value.replace(/[^0-9xX]/g, "").toUpperCase();
}

function isPermissionDeniedError(error: unknown): boolean {
  return (
    error instanceof DOMException &&
    (error.name === "NotAllowedError" || error.name === "SecurityError")
  );
}

function getCameraErrorMessage(
  error: unknown,
  tScan: ReturnType<typeof useTranslations>
): string {
  if (error instanceof Error && error.message === "CAMERA_START_TIMEOUT") {
    return tScan("errors.cameraStartTimedOut");
  }

  if (!(error instanceof DOMException)) {
    return tScan("errors.cameraUnavailable");
  }

  switch (error.name) {
    case "NotFoundError":
      return tScan("errors.noCameraFound");
    case "NotReadableError":
    case "TrackStartError":
      return tScan("errors.cameraBusy");
    case "OverconstrainedError":
      return tScan("errors.cameraConstraintsFailed");
    case "AbortError":
      return tScan("errors.cameraStartAborted");
    default:
      return tScan("errors.cameraUnavailable");
  }
}

async function requestCameraStream(
  constraints: MediaStreamConstraints,
  timeoutMs: number
): Promise<MediaStream> {
  return await Promise.race([
    navigator.mediaDevices.getUserMedia(constraints),
    new Promise<never>((_, reject) => {
      window.setTimeout(() => {
        reject(new Error("CAMERA_START_TIMEOUT"));
      }, timeoutMs);
    })
  ]);
}
