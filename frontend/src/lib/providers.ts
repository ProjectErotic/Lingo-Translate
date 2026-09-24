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
    description: "Official Google Gemini models (Gemini 3.8 Flash, 3.5 Flash Lite, 2.5 Flash/Pro)",
    is_custom: false,
    default_model: "gemini-3.8-flash",
    available_models: [
      "gemini-3.8-flash",
      "gemini-3.5-flash-lite",
      "gemini-3.1-pro-preview",
      "gemini-2.5-flash",
      "gemini-2.5-pro",
      "gemini-2.0-flash",
    ],
  },
  {
    name: "openai",
    display_name: "OpenAI / Compatible",
    description: "Official OpenAI API (GPT-6 Sol/Luna/Astra, o4-mini, o3-mini)",
    is_custom: false,
    base_url: "https://api.openai.com/v1",
    default_model: "gpt-6-sol",
    available_models: [
      "gpt-6-sol",
      "gpt-6-luna",
      "gpt-6-astra",
      "o4-mini",
      "o3-mini",
      "gpt-4o",
      "gpt-4o-mini",
    ],
  },
  {
    name: "uchs",
    display_name: "UCHS AI Infrastructure",
    description: "High-throughput cluster gateway powered by LiteLLM (DeepSeek V4.1 Flash, DeepSeek Pro)",
    is_custom: false,
    base_url: "https://ilms.uchs-th.com/v1",
    default_model: "deepseek-v4.1-flash",
    available_models: [
      "deepseek-v4.1-flash",
      "deepseek-v4-pro-0813",
      "deepseek-v4-flash-0731",
    ],
  },
  {
    name: "deepseek",
    display_name: "DeepSeek Official",
    description: "Direct DeepSeek API (V4.1 Flash, Chat V3, Reasoner R1)",
    is_custom: false,
    base_url: "https://api.deepseek.com/v1",
    default_model: "deepseek-v4.1-flash",
    available_models: [
      "deepseek-v4.1-flash",
      "deepseek-chat",
      "deepseek-reasoner",
    ],
  },
  {
    name: "groq",
    display_name: "Groq Cloud (Ultra-Fast)",
    description: "High-speed LPU inference engine",
    is_custom: false,
    base_url: "https://api.groq.com/openai/v1",
    default_model: "llama-3.3-70b-versatile",
    available_models: [
      "llama-3.3-70b-versatile",
      "llama-3.1-8b-instant",
      "deepseek-r1-distill-llama-70b",
      "mixtral-8x7b-32768",
      "gemma2-9b-it",
    ],
  },
  {
    name: "ollama",
    display_name: "Ollama (Local LLM)",
    description: "Local LLM server running on your machine",
    is_custom: false,
    base_url: "http://localhost:11434/v1",
    default_model: "deepseek-r1:8b",
    available_models: [
      "deepseek-r1:8b",
      "llama3.3:latest",
      "qwen2.5:latest",
      "mistral:latest",
      "phi4:latest",
      "gemma2:latest",
    ],
  },
  {
    name: "openrouter",
    display_name: "OpenRouter Multi-Provider",
    description: "Unified gateway to hundreds of models",
    is_custom: false,
    base_url: "https://openrouter.ai/api/v1",
    default_model: "google/gemini-3.8-flash",
    available_models: [
      "google/gemini-3.8-flash",
      "openai/gpt-6-sol",
      "anthropic/claude-opus-5-5-20260922",
      "deepseek/deepseek-v4.1-flash",
      "google/gemini-2.5-flash",
      "anthropic/claude-3.7-sonnet",
      "meta-llama/llama-3.3-70b-instruct",
    ],
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
