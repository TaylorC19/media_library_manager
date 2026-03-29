"use client";

import { useTranslations } from "next-intl";

export type CameraPermissionState = "prompt" | "granted" | "denied" | "unsupported";

export type ScanViewState =
  | "idle"
  | "requestingPermission"
  | "ready"
  | "decoded"
  | "lookingUp"
  | "results"
  | "saving"
  | "permissionDenied"
  | "cameraUnavailable"
  | "lookupEmpty"
  | "error";

interface ScanStatusBannerProps {
  errorMessage: string | null;
  permissionState: CameraPermissionState;
  scanState: ScanViewState;
}

export function ScanStatusBanner({
  errorMessage,
  permissionState,
  scanState
}: ScanStatusBannerProps) {
  const tScan = useTranslations("scan");
  const tone = getTone(scanState, errorMessage);
  const title = errorMessage
    ? tScan("status.errorTitle")
    : tScan(`status.${scanState}.title`);
  const description = errorMessage
    ? errorMessage
    : tScan(`status.${scanState}.description`);

  return (
    <section
      className={`rounded-3xl border p-4 ${
        tone === "error"
          ? "border-red-500/30 bg-red-500/10"
          : tone === "success"
            ? "border-emerald-500/30 bg-emerald-500/10"
            : "border-slate-800 bg-slate-950/70"
      }`}
    >
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
            {tScan("status.label")}
          </p>
          <h2 className="mt-2 text-xl font-semibold text-white">{title}</h2>
          <p className="mt-2 text-sm text-slate-300">{description}</p>
        </div>

        <div className="rounded-2xl border border-slate-800 bg-slate-950/60 px-4 py-3 text-sm text-slate-300">
          <p className="font-medium text-slate-100">{tScan("permissionLabel")}</p>
          <p className="mt-1">
            {permissionState === "unsupported"
              ? tScan("permission.unsupported")
              : permissionState === "granted"
                ? tScan("permission.granted")
                : permissionState === "denied"
                  ? tScan("permission.denied")
                  : tScan("permission.prompt")}
          </p>
          <p className="mt-2 text-xs text-slate-400">{tScan("permissionHelp")}</p>
        </div>
      </div>
    </section>
  );
}

function getTone(
  scanState: ScanViewState,
  errorMessage: string | null
): "default" | "error" | "success" {
  if (errorMessage || scanState === "permissionDenied" || scanState === "cameraUnavailable") {
    return "error";
  }

  if (scanState === "results") {
    return "success";
  }

  return "default";
}
