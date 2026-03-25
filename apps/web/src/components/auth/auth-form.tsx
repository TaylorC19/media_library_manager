"use client";

import type {
  LoginRequest,
  RegisterRequest
} from "@media-library/types";
import { useTranslations } from "next-intl";
import type { FormEvent } from "react";
import { useMemo, useState } from "react";
import { getLocalizedApiErrorMessageFromResponse } from "../../i18n/errors";
import { Link, useRouter } from "../../i18n/navigation";
import { browserApiFetch } from "../../lib/api-client";

type AuthMode = "login" | "register";

interface AuthFormProps {
  mode: AuthMode;
}

export function AuthForm({ mode }: AuthFormProps) {
  const router = useRouter();
  const tAuth = useTranslations("auth");
  const tCommon = useTranslations("common");
  const tErrors = useTranslations("errors");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const copy = useMemo(
    () =>
      mode === "login"
        ? {
            actionLabel: tCommon("actions.signIn"),
            altHref: "/register",
            altLabel: tAuth("login.altLabel"),
            endpoint: "/auth/login",
            subtitle: tAuth("login.subtitle"),
            title: tAuth("login.title")
          }
        : {
            actionLabel: tCommon("actions.createAccount"),
            altHref: "/login",
            altLabel: tAuth("register.altLabel"),
            endpoint: "/auth/register",
            subtitle: tAuth("register.subtitle"),
            title: tAuth("register.title")
          },
    [mode, tAuth, tCommon]
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
        setErrorMessage(
          await getLocalizedApiErrorMessageFromResponse(response, tErrors)
        );
        return;
      }

      await response.json().catch(() => undefined);
      router.replace("/");
      router.refresh();
    } catch {
      setErrorMessage(tErrors("apiUnavailable"));
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className="w-full max-w-md rounded-3xl border border-slate-800 bg-slate-950/70 p-8 shadow-2xl shadow-slate-950/30">
      <div className="space-y-3">
        <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
          {tCommon("appName")}
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
              {tAuth("fields.displayName")}
            </span>
            <input
              autoComplete="nickname"
              className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
              name="displayName"
              onChange={(event) => setDisplayName(event.target.value)}
              placeholder={tAuth("placeholders.displayName")}
              value={displayName}
            />
          </label>
        ) : null}

        <label className="block space-y-2">
          <span className="text-sm font-medium text-slate-200">
            {tAuth("fields.username")}
          </span>
          <input
            autoCapitalize="none"
            autoComplete="username"
            className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
            name="username"
            onChange={(event) => setUsername(event.target.value)}
            placeholder={tAuth("placeholders.username")}
            required
            value={username}
          />
        </label>

        <label className="block space-y-2">
          <span className="text-sm font-medium text-slate-200">
            {tAuth("fields.password")}
          </span>
          <input
            autoComplete={
              mode === "login" ? "current-password" : "new-password"
            }
            className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
            minLength={mode === "register" ? 8 : 1}
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
          {isSubmitting ? tCommon("actions.working") : copy.actionLabel}
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
