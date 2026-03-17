"use client";

import type {
  LoginRequest,
  RegisterRequest
} from "@media-library/types";
import Link from "next/link";
import { useRouter } from "next/navigation";
import type { FormEvent } from "react";
import { useMemo, useState } from "react";
import { browserApiFetch } from "../../lib/api-client";

type AuthMode = "login" | "register";

interface AuthFormProps {
  mode: AuthMode;
}

function getErrorMessage(payload: unknown): string {
  if (
    typeof payload === "object" &&
    payload !== null &&
    "message" in payload &&
    typeof payload.message === "string"
  ) {
    return payload.message;
  }

  if (
    typeof payload === "object" &&
    payload !== null &&
    "message" in payload &&
    Array.isArray(payload.message)
  ) {
    return payload.message.join(", ");
  }

  return "Something went wrong. Please try again.";
}

export function AuthForm({ mode }: AuthFormProps) {
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const copy = useMemo(
    () =>
      mode === "login"
        ? {
            actionLabel: "Sign in",
            altHref: "/register",
            altLabel: "Create an account",
            endpoint: "/auth/login",
            subtitle: "Sign in to access your catalog and wishlist.",
            title: "Welcome back"
          }
        : {
            actionLabel: "Create account",
            altHref: "/login",
            altLabel: "Already have an account?",
            endpoint: "/auth/register",
            subtitle: "Create a private account to start building your library.",
            title: "Create your account"
          },
    [mode]
  );

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setErrorMessage(null);
    setIsSubmitting(true);

    try {
      const payload: LoginRequest | RegisterRequest =
        mode === "login"
          ? {
              password,
              username
            }
          : {
              displayName: displayName || undefined,
              password,
              username
            };

      const response = await browserApiFetch(copy.endpoint, {
        body: JSON.stringify(payload),
        headers: {
          "content-type": "application/json"
        },
        method: "POST"
      });

      if (!response.ok) {
        const errorPayload = (await response.json().catch(() => null)) as unknown;
        setErrorMessage(getErrorMessage(errorPayload));
        return;
      }

      await response.json().catch(() => undefined);
      router.replace("/");
      router.refresh();
    } catch {
      setErrorMessage("Unable to reach the API right now.");
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className="w-full max-w-md rounded-3xl border border-slate-800 bg-slate-950/70 p-8 shadow-2xl shadow-slate-950/30">
      <div className="space-y-3">
        <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
          Media Library Manager
        </p>
        <h1 className="text-3xl font-semibold tracking-tight text-white">
          {copy.title}
        </h1>
        <p className="text-sm text-slate-300">{copy.subtitle}</p>
      </div>

      <form className="mt-8 space-y-5" onSubmit={handleSubmit}>
        {mode === "register" ? (
          <label className="block space-y-2">
            <span className="text-sm font-medium text-slate-200">
              Display name
            </span>
            <input
              autoComplete="nickname"
              className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
              name="displayName"
              onChange={(event) => setDisplayName(event.target.value)}
              placeholder="Optional"
              value={displayName}
            />
          </label>
        ) : null}

        <label className="block space-y-2">
          <span className="text-sm font-medium text-slate-200">Username</span>
          <input
            autoCapitalize="none"
            autoComplete="username"
            className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
            name="username"
            onChange={(event) => setUsername(event.target.value)}
            placeholder="your_username"
            required
            value={username}
          />
        </label>

        <label className="block space-y-2">
          <span className="text-sm font-medium text-slate-200">Password</span>
          <input
            autoComplete={
              mode === "login" ? "current-password" : "new-password"
            }
            className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
            minLength={mode === "register" ? 10 : 1}
            name="password"
            onChange={(event) => setPassword(event.target.value)}
            required
            type="password"
            value={password}
          />
        </label>

        {errorMessage ? (
          <p className="rounded-2xl border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-200">
            {errorMessage}
          </p>
        ) : null}

        <button
          className="w-full rounded-2xl bg-sky-400 px-4 py-3 text-sm font-semibold text-slate-950 transition hover:bg-sky-300 disabled:cursor-not-allowed disabled:opacity-70"
          disabled={isSubmitting}
          type="submit"
        >
          {isSubmitting ? "Working..." : copy.actionLabel}
        </button>
      </form>

      <p className="mt-6 text-sm text-slate-300">
        <Link className="text-sky-300 hover:text-sky-200" href={copy.altHref}>
          {copy.altLabel}
        </Link>
      </p>
    </div>
  );
}
