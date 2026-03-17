"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { browserApiFetch } from "../../lib/api-client";

export function LogoutButton() {
  const router = useRouter();
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleLogout() {
    setIsSubmitting(true);

    try {
      await browserApiFetch("/auth/logout", {
        method: "POST"
      });
    } finally {
      router.replace("/login");
      router.refresh();
      setIsSubmitting(false);
    }
  }

  return (
    <button
      className="rounded-full border border-slate-700 px-4 py-2 text-sm font-medium text-slate-200 transition hover:border-slate-500 hover:text-white disabled:cursor-not-allowed disabled:opacity-70"
      disabled={isSubmitting}
      onClick={handleLogout}
      type="button"
    >
      {isSubmitting ? "Signing out..." : "Logout"}
    </button>
  );
}
