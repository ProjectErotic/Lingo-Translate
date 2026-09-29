import React, { createContext, useContext, useState, useEffect, useCallback } from "react";
import thLocale from "@/locales/th.json";
import enLocale from "@/locales/en.json";

export type Locale = "th" | "en";

export interface LocaleInfo {
  code: Locale;
  name: string;
  nativeName: string;
}

export const SUPPORTED_LOCALES: LocaleInfo[] = [
  { code: "th", name: "Thai", nativeName: "ภาษาไทย" },
  { code: "en", name: "English", nativeName: "English" },
];

const dictionaries: Record<Locale, any> = {
  th: thLocale,
  en: enLocale,
};

interface I18nContextType {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: string, params?: Record<string, string | number>, fallback?: string) => string;
  supportedLocales: LocaleInfo[];
}

const I18nContext = createContext<I18nContextType | null>(null);

function resolveKey(obj: any, path: string): string | undefined {
  if (!obj) return undefined;
  const parts = path.split(".");
  let curr = obj;
  for (const part of parts) {
    if (curr === undefined || curr === null || typeof curr !== "object") {
      return undefined;
    }
    curr = curr[part];
  }
  return typeof curr === "string" ? curr : undefined;
}

export const I18nProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [locale, setLocaleState] = useState<Locale>(() => {
    const saved = localStorage.getItem("lingo_locale");
    if (saved === "en" || saved === "th") {
      return saved as Locale;
    }
    // Default to Thai as requested
    return "th";
  });

  const setLocale = useCallback((newLocale: Locale) => {
    setLocaleState(newLocale);
    localStorage.setItem("lingo_locale", newLocale);
    document.documentElement.lang = newLocale;
  }, []);

  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);

  const t = useCallback(
    (key: string, params?: Record<string, string | number>, fallback?: string): string => {
      // 1. Try current locale
      let text = resolveKey(dictionaries[locale], key);

      // 2. Fallback to Thai (Master) if missing in current locale
      if (text === undefined && locale !== "th") {
        text = resolveKey(dictionaries["th"], key);
      }

      // 3. Fallback to English if missing in Thai
      if (text === undefined && locale !== "en") {
        text = resolveKey(dictionaries["en"], key);
      }

      // 4. Fallback to provided fallback or key
      if (text === undefined) {
        text = fallback !== undefined ? fallback : key;
      }

      // Interpolate parameters {name}, {count}, etc.
      if (params && typeof text === "string") {
        return text.replace(/\{([a-zA-Z0-9_]+)\}/g, (_, matchKey) => {
          if (params[matchKey] !== undefined) {
            return String(params[matchKey]);
          }
          return `{${matchKey}}`;
        });
      }

      return text;
    },
    [locale]
  );

  return (
    <I18nContext.Provider
      value={{
        locale,
        setLocale,
        t,
        supportedLocales: SUPPORTED_LOCALES,
      }}
    >
      {children}
    </I18nContext.Provider>
  );
};

export function useI18n() {
  const context = useContext(I18nContext);
  if (!context) {
    throw new Error("useI18n must be used within an I18nProvider");
  }
  return context;
}
