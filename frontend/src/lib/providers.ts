export interface ProviderInfo {
  name: string;
  display_name: string;
  description?: string;
  is_custom: boolean;
  base_url?: string;
  default_model?: string;
  available_models?: string[];
}

export const BUILTIN_PROVIDERS: ProviderInfo[] = [
  {
    name: "mock",
    display_name: "Mock (Debug / Offline)",
    description: "Fast offline mock provider for previewing translation workflows without API costs",
    is_custom: false,
  },
  {
    name: "gemini",
    display_name: "Google Gemini AI",
    description: "Official Google Gemini models (Gemini 2.5 Flash, 1.5 Pro)",
    is_custom: false,
    default_model: "gemini-2.5-flash",
    available_models: ["gemini-2.5-flash", "gemini-2.5-pro", "gemini-1.5-flash", "gemini-1.5-pro"],
  },
  {
    name: "openai",
    display_name: "OpenAI / Compatible",
    description: "Standard OpenAI API or local LLM server (Ollama, LM Studio)",
    is_custom: false,
    base_url: "https://api.openai.com/v1",
    default_model: "gpt-4o-mini",
    available_models: ["gpt-4o-mini", "gpt-4o", "o3-mini"],
  },
  {
    name: "uchs",
    display_name: "UCHS AI Infrastructure",
    description: "High-throughput cluster gateway powered by LiteLLM (DeepSeek V4.1 Flash, DeepSeek Pro)",
    is_custom: false,
    base_url: "https://ilms.uchs-th.com/v1",
    default_model: "deepseek-v4.1-flash",
    available_models: ["deepseek-v4.1-flash", "deepseek-v4-pro-0813", "deepseek-v4-flash-0731"],
  },
  {
    name: "google",
    display_name: "Google Translate API",
    description: "Google Cloud Translation API v2",
    is_custom: false,
  },
];

/**
 * Dynamically fetches all available providers (both built-in and external JSON plugins)
 */
export async function fetchProviders(): Promise<ProviderInfo[]> {
  // 1. Try fetching from Wails desktop runtime via Call.ByName
  try {
    const runtime = await import("@wailsio/runtime");
    if (runtime && runtime.Call && typeof runtime.Call.ByName === "function") {
      const list = await runtime.Call.ByName("main.SettingsService.GetProviders");
      if (Array.isArray(list) && list.length > 0) {
        return list;
      }
    }
  } catch {
    // Wails runtime call not available or method not bound
  }

  // 2. Try HTTP endpoint (Web UI dashboard or dev server)
  try {
    const res = await fetch("/api/providers");
    if (res.ok) {
      const list = await res.json();
      if (Array.isArray(list) && list.length > 0) {
        return list;
      }
    }
  } catch {
    // HTTP fetch not available
  }

  // 3. Fallback to builtin providers
  return BUILTIN_PROVIDERS;
}

/**
 * Save a new external provider definition
 */
export async function saveCustomProvider(def: {
  name: string;
  display_name?: string;
  description?: string;
  base_url: string;
  default_model: string;
  available_models?: string[];
}): Promise<boolean> {
  try {
    const res = await fetch("/api/providers/custom", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(def),
    });
    return res.ok;
  } catch {
    return false;
  }
}
