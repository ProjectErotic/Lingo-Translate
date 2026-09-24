import React, { useEffect, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  Button,
  Input,
  Progress,
  ModelSelect,
} from "@/ui";
import {
  TranslationService,
  SettingsService,
} from "@bindings/lingo-translate/cmd/lingo-desktop";
import type { TranslationDonePayload, Settings } from "@bindings/lingo-translate/cmd/lingo-desktop";
import type { TranslationProgress } from "@bindings/lingo-translate/pkg/model";
import { Events } from "@wailsio/runtime";
import { toast } from "sonner";
import { Languages, Loader2, Play, XCircle, Cpu, Zap, ShieldAlert, Database } from "lucide-react";
import { fetchProviders, BUILTIN_PROVIDERS, type ProviderInfo } from "@/lib/providers";

interface TranslateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  currentFile?: string;
  sourceLang: string;
  targetLang: string;
  onFinished?: () => void;
}

export const TranslateDialog: React.FC<TranslateDialogProps> = ({
  open,
  onOpenChange,
  currentFile,
  sourceLang,
  targetLang,
  onFinished,
}) => {
  const [providers, setProviders] = useState<ProviderInfo[]>(BUILTIN_PROVIDERS);
  const [provider, setProvider] = useState("mock");
  const [modelName, setModelName] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [baseURL, setBaseURL] = useState("");
  const [scope, setScope] = useState<"all" | "untranslated" | "file">("untranslated");
  const [batchSize, setBatchSize] = useState(10);
  const [concurrency, setConcurrency] = useState(4);

  // Fallback and Task routing states
  const [fallbackEnabled, setFallbackEnabled] = useState(false);
  const [fallbackProvider, setFallbackProvider] = useState("gemini");
  const [fallbackModel, setFallbackModel] = useState("gemini-3.8-flash");

  // Context & Memory states
  const [translationStyle, setTranslationStyle] = useState("standard");
  const [customPrompt, setCustomPrompt] = useState("");
  const [useMemoryCache, setUseMemoryCache] = useState(true);

  const [isRunning, setIsRunning] = useState(false);
  const [progress, setProgress] = useState<TranslationProgress | null>(null);
  const [settings, setSettings] = useState<Settings | null>(null);
  const [autoRouteShort, setAutoRouteShort] = useState(false);
  const [enableSystemOne, setEnableSystemOne] = useState(false);

  // Load available providers & settings defaults on dialog open
  useEffect(() => {
    if (open) {
      // 1. Fetch dynamic providers/plugins list
      fetchProviders().then((loadedProviders) => {
        setProviders(loadedProviders);
      });

      // 2. Load saved settings
      SettingsService.GetSettings().then((s) => {
        if (s) {
          setSettings(s);
          setAutoRouteShort(s.tasks?.auto_route_short_text ?? false);
          setEnableSystemOne(s.system_one?.enabled ?? false);

          const defaultProv = s.default_provider || "mock";
          setProvider(defaultProv);
          setModelName(s.default_model || "");
          setBatchSize(s.default_batch_size || 10);
          setConcurrency(s.default_concurrency || 4);

          // Fallback defaults from settings
          setFallbackEnabled((s.tasks as any)?.enable_fallback ?? false);
          setFallbackProvider((s.tasks as any)?.fallback_translation?.provider || "gemini");
          setFallbackModel((s.tasks as any)?.fallback_translation?.model || "gemini-3.8-flash");

          // Memory & Context defaults
          setUseMemoryCache((s as any).enable_memory_cache ?? true);
          setTranslationStyle((s as any).translation_style || "standard");
          setCustomPrompt((s as any).context_lore || "");

          // Resolve API key & baseURL for active provider
          if (defaultProv === "gemini") {
            setApiKey(s.gemini_api_key || "");
          } else if (defaultProv === "openai") {
            setApiKey(s.openai_api_key || "");
            setBaseURL(s.openai_base_url || "");
          } else if (defaultProv === "uchs") {
            setApiKey(s.uchs_api_key || "");
            setBaseURL("https://ilms.uchs-th.com/v1");
          } else if (defaultProv === "google") {
            setApiKey(s.google_api_key || "");
          } else if (s.plugin_keys && s.plugin_keys[defaultProv]) {
            setApiKey(s.plugin_keys[defaultProv]);
            setBaseURL(s.plugin_base_urls?.[defaultProv] || "");
          } else if ((s as any)[`${defaultProv}_api_key`]) {
            setApiKey((s as any)[`${defaultProv}_api_key`]);
            setBaseURL((s as any)[`${defaultProv}_base_url`] || "");
          }
        }
      });

      TranslationService.IsRunning().then(setIsRunning);
    }
  }, [open]);

  // Update apiKey and model options dynamically when provider changes
  const handleProviderChange = (newProvider: string) => {
    setProvider(newProvider);
    const pInfo = providers.find((p: ProviderInfo) => p.name === newProvider);

    SettingsService.GetSettings().then((s) => {
      let key = "";
      let base = pInfo?.base_url || "";
      let model = pInfo?.default_model || (pInfo?.available_models?.[0] || "");

      if (s) {
        if (newProvider === "gemini") {
          key = s.gemini_api_key || "";
        } else if (newProvider === "openai") {
          key = s.openai_api_key || "";
          base = s.openai_base_url || base;
        } else if (newProvider === "uchs") {
          key = s.uchs_api_key || "";
          base = "https://ilms.uchs-th.com/v1";
        } else if (newProvider === "google") {
          key = s.google_api_key || "";
        } else if (s.plugin_keys && s.plugin_keys[newProvider]) {
          key = s.plugin_keys[newProvider];
          base = s.plugin_base_urls?.[newProvider] || base;
        } else if ((s as any)[`${newProvider}_api_key`]) {
          key = (s as any)[`${newProvider}_api_key`];
          base = (s as any)[`${newProvider}_base_url`] || base;
        }
      }

      setApiKey(key);
      setBaseURL(base);
      if (model) {
        setModelName(model);
      }
    });
  };

  // Subscribe to translation events
  useEffect(() => {
    let unregProgress: (() => void) | undefined;
    let unregDone: (() => void) | undefined;

    try {
      unregProgress = Events.On("translation:progress", (event: any) => {
        const data = event.data?.[0] || event.data;
        if (data) {
          setProgress(data as TranslationProgress);
        }
      });

      unregDone = Events.On("translation:done", (event: any) => {
        const payload = (event.data?.[0] || event.data) as TranslationDonePayload;
        setIsRunning(false);
        if (payload?.success) {
          toast.success("Translation finished successfully!");
        } else {
          toast.error(`Translation finished with errors: ${payload?.error || "Unknown error"}`);
        }
        if (onFinished) onFinished();
      });
    } catch (e) {
      console.warn("Event registration error:", e);
    }

    return () => {
      if (unregProgress) unregProgress();
      if (unregDone) unregDone();
    };
  }, [onFinished]);

  const handleStart = async () => {
    try {
      setIsRunning(true);
      setProgress(null);

      let effectiveScope = scope === "file" ? currentFile || "all" : scope;

      const startOpts: any = {
        provider: {
          name: provider,
          api_key: apiKey,
          model: modelName,
          base_url: baseURL,
        },
        source_lang: sourceLang,
        target_lang: targetLang,
        batch_size: batchSize,
        concurrency: concurrency,
        scope: effectiveScope,
      };

      if (autoRouteShort && settings?.tasks?.fast_translation?.provider) {
        startOpts.auto_route_short = true;
        startOpts.max_short_len = settings.tasks.max_short_length || 60;
        startOpts.fast_provider = {
          name: settings.tasks.fast_translation.provider,
          model: settings.tasks.fast_translation.model || "",
          api_key: "",
          base_url: "",
        };
      } else {
        startOpts.auto_route_short = false;
      }

      if (fallbackEnabled && fallbackProvider) {
        const fbInfo = providers.find((p: ProviderInfo) => p.name === fallbackProvider);
        startOpts.fallback_provider = {
          name: fallbackProvider,
          model: fallbackModel,
          api_key: "",
          base_url: fbInfo?.base_url || "",
        };
      }

      startOpts.enable_memory_cache = useMemoryCache;
      startOpts.style = translationStyle;
      if (customPrompt.trim()) {
        startOpts.prompt = customPrompt.trim();
      }

      if (enableSystemOne && settings?.system_one) {
        startOpts.system_one = {
          enabled: true,
          provider: settings.system_one.provider || "heuristic",
          api_key: settings.system_one.api_key || "",
          base_url: settings.system_one.base_url || "",
          confidence_threshold: settings.system_one.confidence_threshold || 0.85,
          filter_ambiguous_code: settings.system_one.features?.filter_ambiguous_code ?? true,
          accept_ui_drafts: settings.system_one.features?.accept_ui_drafts ?? true,
          verify_qa: settings.system_one.features?.verify_qa ?? false,
        };
      } else {
        startOpts.system_one = {
          enabled: false,
          provider: "heuristic",
          api_key: "",
          filter_ambiguous_code: false,
          accept_ui_drafts: false,
          verify_qa: false,
        };
      }

      await TranslationService.Start(startOpts);

      toast.info("Translation pipeline started in background");
    } catch (err: any) {
      setIsRunning(false);
      toast.error(`Failed to start translation: ${err?.message || err}`);
    }
  };

  const handleCancel = async () => {
    try {
      await TranslationService.Cancel();
      toast.warning("Cancel request sent");
    } catch (err: any) {
      toast.error(`Failed to cancel: ${err?.message || err}`);
    }
  };

  const currentProviderInfo = providers.find((p: ProviderInfo) => p.name === provider);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-base">
            <Languages className="w-5 h-5 text-primary" />
            AI Translation Pipeline
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2 text-sm">
          {/* Dynamic Provider Selection */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-semibold text-muted-foreground mb-1">
                Translation Service
              </label>
              {/* @ui-allow-native */}
              <select
                value={provider}
                onChange={(e) => handleProviderChange(e.target.value)}
                disabled={isRunning}
                className="w-full h-8 rounded-md border border-input bg-card px-2 text-sm text-foreground focus:outline-none focus:border-primary disabled:opacity-50"
              >
                {providers.map((p: ProviderInfo) => (
                  <option key={p.name} value={p.name}>
                    {p.display_name} {p.is_custom ? "(Plugin)" : ""}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-xs font-semibold text-muted-foreground mb-1">
                Model Selection
              </label>
              <ModelSelect
                providerInfo={currentProviderInfo}
                value={modelName}
                onChange={setModelName}
                disabled={isRunning}
                placeholder={currentProviderInfo?.default_model || "e.g. gpt-6-sol, gemini-3.8-flash"}
                showChips={true}
                apiKey={apiKey}
                baseURL={baseURL || currentProviderInfo?.base_url}
              />
            </div>
          </div>

          {/* API Key (for non-mock providers) */}
          {provider !== "mock" && (
            <div>
              <label className="block text-xs font-semibold text-muted-foreground mb-1">
                API Key {currentProviderInfo?.is_custom ? `(${currentProviderInfo.display_name})` : ""}
              </label>
              <Input
                type="password"
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                disabled={isRunning}
                placeholder={`Enter API Key for ${currentProviderInfo?.display_name || provider}`}
              />
            </div>
          )}

          {/* Base URL (if custom plugin or openai) */}
          {(currentProviderInfo?.is_custom || provider === "openai") && (
            <div>
              <label className="block text-xs font-semibold text-muted-foreground mb-1">
                Base URL {currentProviderInfo?.is_custom ? "(Plugin Endpoint)" : "(Custom Endpoint)"}
              </label>
              <Input
                value={baseURL}
                onChange={(e) => setBaseURL(e.target.value)}
                disabled={isRunning}
                placeholder={currentProviderInfo?.base_url || "https://..."}
              />
            </div>
          )}

          {/* Scope */}
          <div>
            <label className="block text-xs font-semibold text-muted-foreground mb-1.5">
              Translation Scope
            </label>
            <div className="flex gap-2">
              <label className="flex items-center gap-1.5 text-xs text-foreground cursor-pointer">
                {/* @ui-allow-native */}
                <input
                  type="radio"
                  name="scope"
                  checked={scope === "untranslated"}
                  onChange={() => setScope("untranslated")}
                  disabled={isRunning}
                  className="accent-primary"
                />
                Untranslated Only
              </label>
              <label className="flex items-center gap-1.5 text-xs text-foreground cursor-pointer">
                {/* @ui-allow-native */}
                <input
                  type="radio"
                  name="scope"
                  checked={scope === "all"}
                  onChange={() => setScope("all")}
                  disabled={isRunning}
                  className="accent-primary"
                />
                All Text (Overwrite)
              </label>
              {currentFile && (
                <label className="flex items-center gap-1.5 text-xs text-foreground cursor-pointer">
                  {/* @ui-allow-native */}
                  <input
                    type="radio"
                    name="scope"
                    checked={scope === "file"}
                    onChange={() => setScope("file")}
                    disabled={isRunning}
                    className="accent-primary"
                  />
                  Current File ({currentFile})
                </label>
              )}
            </div>
          </div>

          {/* Batch Size & Concurrency */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-semibold text-muted-foreground mb-1">
                Batch Size (lines)
              </label>
              <Input
                type="number"
                min={1}
                max={1000}
                value={batchSize}
                onChange={(e) => setBatchSize(parseInt(e.target.value) || 10)}
                disabled={isRunning}
              />
            </div>
            <div>
              <label className="block text-xs font-semibold text-muted-foreground mb-1">
                Concurrency (workers)
              </label>
              <Input
                type="number"
                min={1}
                max={16}
                value={concurrency}
                onChange={(e) => setConcurrency(parseInt(e.target.value) || 4)}
                disabled={isRunning}
              />
            </div>
          </div>

          {/* Automatic Fallback Failover */}
          <div className="rounded-lg border border-border/70 bg-card/60 p-3 space-y-2.5">
            <div className="flex items-center justify-between">
              <label className="flex items-center gap-2 cursor-pointer select-none text-xs font-semibold text-foreground">
                {/* @ui-allow-native */}
                <input
                  type="checkbox"
                  checked={fallbackEnabled}
                  onChange={(e) => setFallbackEnabled(e.target.checked)}
                  disabled={isRunning}
                  className="rounded border-border text-primary focus:ring-primary"
                />
                <span className="flex items-center gap-1.5">
                  <ShieldAlert className="w-3.5 h-3.5 text-amber-400" />
                  Automatic Fallback Failover
                </span>
              </label>
              <span className="text-[10px] text-muted-foreground">
                Auto-switches provider on error (429, timeout)
              </span>
            </div>

            {fallbackEnabled && (
              <div className="grid grid-cols-2 gap-3 pt-1 border-t border-border/40">
                <div>
                  <label className="block text-[11px] font-medium text-muted-foreground mb-1">
                    Fallback Provider
                  </label>
                  {/* @ui-allow-native */}
                  <select
                    value={fallbackProvider}
                    onChange={(e) => {
                      const val = e.target.value;
                      setFallbackProvider(val);
                      const fInfo = providers.find((p: ProviderInfo) => p.name === val);
                      setFallbackModel(fInfo?.default_model || (fInfo?.available_models?.[0] || ""));
                    }}
                    disabled={isRunning}
                    className="w-full h-8 rounded-md border border-input bg-card px-2 text-xs text-foreground focus:outline-none focus:border-primary disabled:opacity-50"
                  >
                    {providers.filter((p: ProviderInfo) => p.name !== provider).map((p: ProviderInfo) => (
                      <option key={p.name} value={p.name}>
                        {p.display_name}
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-[11px] font-medium text-muted-foreground mb-1">
                    Fallback Model
                  </label>
                  <ModelSelect
                    providerInfo={providers.find((p: ProviderInfo) => p.name === fallbackProvider)}
                    value={fallbackModel}
                    onChange={setFallbackModel}
                    disabled={isRunning}
                    showChips={false}
                    apiKey={
                      settings
                        ? fallbackProvider === "uchs"
                          ? settings.uchs_api_key
                          : fallbackProvider === "gemini"
                          ? settings.gemini_api_key
                          : fallbackProvider === "openai"
                          ? settings.openai_api_key
                          : settings.plugin_keys?.[fallbackProvider] || ""
                        : ""
                    }
                    baseURL={providers.find((p: ProviderInfo) => p.name === fallbackProvider)?.base_url}
                  />
                </div>
              </div>
            )}
          </div>

          {/* Translation Memory Cache & Context */}
          <div className="rounded-lg border border-border/70 bg-card/60 p-3 space-y-2.5">
            <div className="flex items-center justify-between">
              <label className="flex items-center gap-2 cursor-pointer select-none text-xs font-semibold text-foreground">
                {/* @ui-allow-native */}
                <input
                  type="checkbox"
                  checked={useMemoryCache}
                  onChange={(e) => setUseMemoryCache(e.target.checked)}
                  disabled={isRunning}
                  className="rounded border-border text-primary focus:ring-primary"
                />
                <span className="flex items-center gap-1.5">
                  <Database className="w-3.5 h-3.5 text-emerald-400" />
                  Translation Memory (TM Cache)
                </span>
              </label>
              <span className="text-[10px] text-muted-foreground">
                Reuses exact match lines from SQLite
              </span>
            </div>

            <div className="space-y-2 pt-1 border-t border-border/40">
              <div>
                <label className="block text-[11px] font-medium text-muted-foreground mb-1">
                  Localization Tone &amp; Persona
                </label>
                {/* @ui-allow-native */}
                <select
                  value={translationStyle}
                  onChange={(e) => setTranslationStyle(e.target.value)}
                  disabled={isRunning}
                  className="w-full h-8 rounded-md border border-input bg-card px-2.5 text-xs text-foreground focus:outline-none focus:border-primary disabled:opacity-50"
                >
                  <option value="standard">Standard Game Localization (Balanced)</option>
                  <option value="nsfw">NSFW / Adult / Erotic (Uncensored 18+)</option>
                  <option value="vn_romance">Visual Novel &amp; Romance (Intimacy &amp; Dialogue)</option>
                  <option value="fantasy_rpg">Fantasy RPG &amp; Medieval</option>
                  <option value="comedy">Comedy &amp; Playful</option>
                  <option value="dan_uncensored">Anti-Censorship Jailbreak</option>
                </select>
              </div>

              <div>
                <label className="block text-[11px] font-medium text-muted-foreground mb-1">
                  Custom Game Lore &amp; Directives (Optional)
                </label>
                <Input
                  className="h-8 text-xs"
                  value={customPrompt}
                  onChange={(e) => setCustomPrompt(e.target.value)}
                  disabled={isRunning}
                  placeholder="e.g. Hero is named Ken, heroine is Yuna, fantasy academy setting"
                />
              </div>
            </div>
          </div>

          {/* Hermes Task Routing & System One Auxiliary Section */}
          <div className="rounded-lg border border-border/70 bg-card/60 p-3 space-y-2.5">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-foreground flex items-center gap-1.5">
                <Cpu className="w-3.5 h-3.5 text-primary" />
                AI Routing &amp; Decision System
              </span>
              <span className="text-[11px] text-muted-foreground font-mono">
                Hermes + Jev
              </span>
            </div>

            {/* Task Routing: Fast model for short text */}
            <div className="pt-1.5 border-t border-border/50 text-xs">
              <label className="flex items-start gap-2 cursor-pointer">
                {/* @ui-allow-native */}
                <input
                  type="checkbox"
                  checked={autoRouteShort}
                  onChange={(e) => setAutoRouteShort(e.target.checked)}
                  disabled={isRunning}
                  className="rounded border-input text-primary mt-0.5"
                />
                <div className="space-y-0.5">
                  <div className="font-medium text-foreground flex items-center gap-1.5">
                    Fast Model Auto-Route
                    {autoRouteShort && (
                      <span className="text-[10px] bg-primary/15 text-primary px-1.5 py-0.5 rounded font-mono">
                        &lt;{settings?.tasks?.max_short_length || 60} chars
                      </span>
                    )}
                  </div>
                  <div className="text-[11px] text-muted-foreground">
                    Bulk short lines/UI routed to{" "}
                    <strong className="text-foreground">
                      {settings?.tasks?.fast_translation?.provider || "fast provider"}
                    </strong>{" "}
                    ({settings?.tasks?.fast_translation?.model || "default"})
                  </div>
                </div>
              </label>
            </div>

            {/* System One Auxiliary Decision Engine */}
            <div className="pt-2 border-t border-border/50 text-xs">
              <label className="flex items-start gap-2 cursor-pointer">
                {/* @ui-allow-native */}
                <input
                  type="checkbox"
                  checked={enableSystemOne}
                  onChange={(e) => setEnableSystemOne(e.target.checked)}
                  disabled={isRunning}
                  className="rounded border-input text-primary mt-0.5"
                />
                <div className="space-y-0.5">
                  <div className="font-medium text-foreground flex items-center gap-1.5">
                    System One Decision Engine
                    {enableSystemOne ? (
                      <span className="text-[10px] bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 px-1.5 py-0.5 rounded flex items-center gap-1 font-mono">
                        <Zap className="w-2.5 h-2.5" />
                        {settings?.system_one?.provider === "typesafe_jev" ? "TypeSafe Jev" : "Heuristic"}
                      </span>
                    ) : (
                      <span className="text-[10px] text-muted-foreground font-mono">
                        (Disabled)
                      </span>
                    )}
                  </div>
                  <div className="text-[11px] text-muted-foreground">
                    Filters ambiguous code &amp; speculatively accepts high-confidence drafts
                  </div>
                </div>
              </label>
            </div>
          </div>

          {/* Languages info indicator */}
          <div className="flex items-center justify-between text-xs text-muted-foreground bg-background p-2 rounded-md border border-border">
            <span>
              Source: <strong className="text-foreground">{sourceLang}</strong>
            </span>
            <span>➔</span>
            <span>
              Target: <strong className="text-primary">{targetLang}</strong>
            </span>
          </div>

          {/* Progress Tracker */}
          {isRunning && (
            <div className="space-y-2 border-t border-border pt-3">
              <div className="flex justify-between text-xs text-muted-foreground">
                <span>
                  Translated: {progress ? progress.completed : 0} /{" "}
                  {progress ? progress.total : 0}
                </span>
                <span>
                  {progress ? Math.round(progress.percent) : 0}%
                </span>
              </div>
              <Progress
                value={progress ? progress.percent : 0}
              />
              <div className="text-[11px] text-muted-foreground truncate flex items-center gap-1.5">
                <Loader2 className="w-3 h-3 animate-spin text-primary" />
                {progress?.current_file ? `Translating ${progress.current_file}...` : "Working..."}
              </div>
            </div>
          )}
        </div>

        <DialogFooter className="gap-2 sm:gap-0">
          {isRunning ? (
            <Button
              variant="destructive"
              onClick={handleCancel}
              className="gap-1.5"
            >
              <XCircle className="w-4 h-4" />
              Cancel Translation
            </Button>
          ) : (
            <>
              <Button
                variant="secondary"
                onClick={() => onOpenChange(false)}
              >
                Close
              </Button>
              <Button
                onClick={handleStart}
                className="gap-1.5"
              >
                <Play className="w-4 h-4 fill-current" />
                Start Translation
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
