import React, { useState, useEffect, useRef, useMemo } from "react";
import { cn } from "@/lib/utils";
import { Button } from "./button";
import { Search, ChevronDown, Check, Sparkles, RefreshCw, Server } from "lucide-react";
import { toast } from "sonner";
import * as SettingsService from "@bindings/lingo-translate/cmd/lingo-desktop/settingsservice.js";
import type { ProviderInfo } from "@/lib/providers";

const KNOWN_PROVIDER_MODELS: Record<string, string[]> = {
  uchs: [
    "deepseek-v4.1-flash",
    "deepseek-v4-pro-0813",
    "deepseek-v4-flash-0731",
  ],
  gemini: [
    "gemini-3.8-flash",
    "gemini-3.5-flash-lite",
    "gemini-3.1-pro-preview",
    "gemini-2.5-flash",
    "gemini-2.5-pro",
    "gemini-2.0-flash",
  ],
  openai: [
    "gpt-6-sol",
    "gpt-6-luna",
    "gpt-6-astra",
    "o4-mini",
    "o3-mini",
    "gpt-4o",
    "gpt-4o-mini",
  ],
  claude: [
    "claude-opus-5-5-20260922",
    "claude-fable-5-1-20260901",
    "claude-3-7-sonnet-20250219",
    "claude-3-5-haiku-20241022",
  ],
  anthropic: [
    "claude-opus-5-5-20260922",
    "claude-fable-5-1-20260901",
    "claude-3-7-sonnet-20250219",
    "claude-3-5-haiku-20241022",
  ],
  deepseek: [
    "deepseek-v4.1-flash",
    "deepseek-chat",
    "deepseek-reasoner",
  ],
  groq: [
    "llama-3.3-70b-versatile",
    "llama-3.1-8b-instant",
    "deepseek-r1-distill-llama-70b",
    "mixtral-8x7b-32768",
    "gemma2-9b-it",
  ],
  ollama: [
    "deepseek-r1:8b",
    "llama3.3:latest",
    "qwen2.5:latest",
    "mistral:latest",
    "phi4:latest",
  ],
  openrouter: [
    "google/gemini-3.8-flash",
    "openai/gpt-6-sol",
    "anthropic/claude-opus-5-5-20260922",
    "deepseek/deepseek-v4.1-flash",
    "google/gemini-2.5-flash",
    "anthropic/claude-3.7-sonnet",
    "meta-llama/llama-3.3-70b-instruct",
  ],
};

const RECENT_MODELS_STORAGE_KEY = "lingo_recent_custom_models";

function getRecentModels(): string[] {
  try {
    const raw = localStorage.getItem(RECENT_MODELS_STORAGE_KEY);
    return raw ? JSON.parse(raw) : [];
  } catch {
    return [];
  }
}

function saveRecentModel(model: string) {
  if (!model || model.trim() === "") return;
  try {
    const existing = getRecentModels().filter((m) => m !== model);
    const updated = [model.trim(), ...existing].slice(0, 15);
    localStorage.setItem(RECENT_MODELS_STORAGE_KEY, JSON.stringify(updated));
  } catch {
    // ignore
  }
}

export interface ModelSelectProps {
  providerInfo?: ProviderInfo;
  value: string;
  onChange: (val: string) => void;
  disabled?: boolean;
  className?: string;
  placeholder?: string;
  showChips?: boolean;
  baseURL?: string;
  apiKey?: string;
}

export const ModelSelect: React.FC<ModelSelectProps> = ({
  providerInfo,
  value,
  onChange,
  disabled = false,
  className,
  placeholder,
  showChips = true,
  baseURL,
  apiKey,
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [remoteModels, setRemoteModels] = useState<string[]>([]);
  const [isFetchingRemote, setIsFetchingRemote] = useState(false);
  const [recentModels, setRecentModels] = useState<string[]>([]);
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setRecentModels(getRecentModels());
  }, []);

  // Close dropdown on outside click
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // Merge models strictly from real data
  const allModels = useMemo(() => {
    const providerKey = (providerInfo?.name || "").toLowerCase();
    const available = providerInfo?.available_models || [];
    const known = KNOWN_PROVIDER_MODELS[providerKey] || [];

    const set = new Set<string>();

    // 1. Live server models fetched dynamically
    remoteModels.forEach((m) => set.add(m));

    // 2. Real models provided by backend provider info
    available.forEach((m) => set.add(m));

    // 3. If no models provided from backend/server, fallback to verified official models
    if (set.size === 0) {
      known.forEach((m) => set.add(m));
    }

    // 4. Custom models recently used by user
    recentModels.forEach((m) => set.add(m));

    if (value && value.trim()) {
      set.add(value.trim());
    }

    return Array.from(set);
  }, [providerInfo, remoteModels, recentModels, value]);

  // Filtered models
  const filteredModels = useMemo(() => {
    if (!searchQuery.trim()) return allModels;
    const q = searchQuery.toLowerCase();
    return allModels.filter((m) => m.toLowerCase().includes(q));
  }, [allModels, searchQuery]);

  const handleSelectModel = (model: string) => {
    onChange(model);
    saveRecentModel(model);
    setRecentModels(getRecentModels());
    setIsOpen(false);
    setSearchQuery("");
  };

  const resolveTargetBaseURL = () => {
    if (baseURL) return baseURL;
    if (providerInfo?.base_url) return providerInfo.base_url;
    if (providerInfo?.name === "ollama") return "http://localhost:11434/v1";
    if (providerInfo?.name === "gemini") return "https://generativelanguage.googleapis.com";
    if (providerInfo?.name === "uchs") return "https://ilms.uchs-th.com/v1";
    if (providerInfo?.name === "openai") return "https://api.openai.com/v1";
    return "";
  };

  const handleFetchRemote = async (e?: React.MouseEvent) => {
    if (e) e.stopPropagation();
    const targetBaseURL = resolveTargetBaseURL();
    if (!targetBaseURL) {
      toast.error("Please configure a Base URL first to fetch live models.");
      return;
    }

    // Resolve API key from prop or fallback to saved settings
    let targetAPIKey = apiKey || "";
    if (!targetAPIKey) {
      try {
        const s = await SettingsService.GetSettings();
        if (providerInfo?.name === "uchs") targetAPIKey = s.uchs_api_key || "";
        else if (providerInfo?.name === "gemini") targetAPIKey = s.gemini_api_key || "";
        else if (providerInfo?.name === "openai") targetAPIKey = s.openai_api_key || "";
        else if (providerInfo?.name && s.plugin_keys) targetAPIKey = s.plugin_keys[providerInfo.name] || "";
      } catch {
        // ignore
      }
    }

    const isLocal = targetBaseURL.includes("localhost") || targetBaseURL.includes("127.0.0.1");
    if (!targetAPIKey && !isLocal && providerInfo?.name !== "mock") {
      toast.warning(`Please configure an API Key for ${providerInfo?.display_name || "this provider"} in Settings.`);
      return;
    }

    try {
      setIsFetchingRemote(true);
      toast.info(`Fetching live models from ${providerInfo?.display_name || targetBaseURL}...`);
      const fetched = await SettingsService.FetchRemoteModels(targetBaseURL, targetAPIKey);
      if (fetched && fetched.length > 0) {
        setRemoteModels(fetched);
        toast.success(`Found ${fetched.length} models on server!`);
        setIsOpen(true);
      } else {
        toast.warning("Server responded but returned no models.");
      }
    } catch (err: any) {
      toast.error(`Failed to fetch models: ${err?.message || err}`);
    } finally {
      setIsFetchingRemote(false);
    }
  };

  const effectiveBaseURL = resolveTargetBaseURL();
  const canFetchLive = Boolean(effectiveBaseURL || providerInfo?.is_custom);

  // Top quick chips (first 4 items)
  const quickChips = useMemo(() => {
    return allModels.slice(0, 4);
  }, [allModels]);

  return (
    <div ref={dropdownRef} className={cn("relative space-y-1.5 w-full", className)}>
      {/* Searchable Select Input Trigger */}
      <div className="flex gap-1.5 items-center">
        <div
          onClick={() => !disabled && setIsOpen(!isOpen)}
          className={cn(
            "flex-1 h-8 px-2.5 rounded-md border border-input bg-card flex items-center justify-between cursor-pointer transition-colors text-xs text-foreground",
            isOpen && "border-primary ring-1 ring-primary/30",
            disabled && "opacity-50 cursor-not-allowed"
          )}
        >
          <span className={cn("font-mono truncate", !value && "text-muted-foreground")}>
            {value || placeholder || providerInfo?.default_model || "Select or enter model..."}
          </span>
          <div className="flex items-center gap-1 shrink-0 text-muted-foreground ml-1">
            <ChevronDown className={cn("w-3.5 h-3.5 transition-transform", isOpen && "rotate-180")} />
          </div>
        </div>

        {canFetchLive && (
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleFetchRemote}
            disabled={disabled || isFetchingRemote}
            className="h-8 px-2 text-[11px] shrink-0 gap-1 border-border bg-card/70 hover:bg-card"
            title="Fetch live models directly from server"
          >
            <RefreshCw className={cn("w-3 h-3 text-primary", isFetchingRemote && "animate-spin")} />
            <span className="hidden sm:inline">Fetch</span>
          </Button>
        )}
      </div>

      {/* Dropdown Menu */}
      {isOpen && (
        <div className="absolute top-full left-0 right-0 z-50 mt-1 max-h-64 overflow-hidden rounded-md border border-border bg-popover text-popover-foreground shadow-lg flex flex-col">
          {/* Search box inside dropdown */}
          <div className="p-2 border-b border-border/70 flex items-center gap-2 bg-muted/40">
            <Search className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
            {/* @ui-allow-native */}
            <input
              type="text"
              autoFocus
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search all models or type custom..."
              className="w-full bg-transparent text-xs text-foreground placeholder:text-muted-foreground outline-none font-mono"
            />
            {canFetchLive && (
              // @ui-allow-native
              <button
                type="button"
                onClick={handleFetchRemote}
                disabled={isFetchingRemote}
                className="text-[10px] text-primary hover:underline shrink-0 flex items-center gap-1"
              >
                <Server className="w-3 h-3" />
                Live
              </button>
            )}
          </div>

          {/* Model Options List */}
          <div className="overflow-y-auto max-h-52 p-1 space-y-0.5 text-xs">
            {/* Custom option when user types something unique */}
            {searchQuery.trim() && !allModels.includes(searchQuery.trim()) && (
              // @ui-allow-native
              <button
                type="button"
                onClick={() => handleSelectModel(searchQuery.trim())}
                className="w-full px-2.5 py-1.5 rounded text-left flex items-center justify-between text-primary hover:bg-primary/10 transition-colors"
              >
                <div className="flex items-center gap-1.5 truncate">
                  <Sparkles className="w-3.5 h-3.5 shrink-0" />
                  <span className="font-mono text-xs truncate">Use custom: "{searchQuery.trim()}"</span>
                </div>
              </button>
            )}

            {filteredModels.length > 0 ? (
              filteredModels.map((m) => {
                const isSelected = value === m;
                const isServerModel = remoteModels.includes(m);
                return (
                  // @ui-allow-native
                  <button
                    key={m}
                    type="button"
                    onClick={() => handleSelectModel(m)}
                    className={cn(
                      "w-full px-2.5 py-1.5 rounded text-left flex items-center justify-between font-mono text-xs transition-colors",
                      isSelected
                        ? "bg-primary text-primary-foreground font-semibold"
                        : "hover:bg-muted text-foreground"
                    )}
                  >
                    <span className="truncate">{m}</span>
                    <div className="flex items-center gap-1.5 shrink-0 ml-2">
                      {isServerModel && (
                        <span className="px-1 py-0.2 rounded text-[9px] bg-emerald-500/20 text-emerald-400 font-sans">
                          Server
                        </span>
                      )}
                      {isSelected && <Check className="w-3.5 h-3.5" />}
                    </div>
                  </button>
                );
              })
            ) : (
              <div className="px-3 py-3 text-center text-xs text-muted-foreground">
                No matching models. Press Enter or click above to use "{searchQuery.trim()}".
              </div>
            )}
          </div>
        </div>
      )}

      {/* Quick Select Chips */}
      {showChips && quickChips.length > 1 && (
        <div className="flex flex-wrap gap-1 pt-0.5">
          {quickChips.map((m) => {
            const isActive = value === m;
            return (
              // @ui-allow-native
              <button
                key={m}
                type="button"
                disabled={disabled}
                onClick={() => handleSelectModel(m)}
                className={cn(
                  "px-1.5 py-0.5 text-[10px] rounded font-mono transition-colors border",
                  isActive
                    ? "bg-primary/20 text-primary border-primary/40 font-semibold"
                    : "bg-muted text-muted-foreground border-transparent hover:text-foreground hover:bg-secondary"
                )}
              >
                {m}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
};
