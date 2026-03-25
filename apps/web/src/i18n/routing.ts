import { defineRouting } from "next-intl/routing";

export const routing = defineRouting({
  defaultLocale: "en",
  localeDetection: true,
  localePrefix: "always",
  locales: ["en", "ja"]
});

export type AppLocale = (typeof routing.locales)[number];
