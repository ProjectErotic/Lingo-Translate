import React, { useEffect, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  Button,
  Input,
  ModelSelect,
} from "@/ui";
import {
  SettingsService,
  EntryService,
} from "@bindings/lingo-translate/cmd/lingo-desktop";
import type { Settings } from "@bindings/lingo-translate/cmd/lingo-desktop";
import { toast } from "sonner";
import {
  Save,
  Settings as SettingsIcon,
  Zap,
  Bot,
  Key,
  Sliders,
  Info,
  Eye,
  EyeOff,
  RefreshCw,
  Layers,
  ExternalLink,
  Sparkles,
  Cpu,
  Palette,
  UserCheck,
  Server,
  Database,
  ShieldAlert,
  Trash2,
  CheckCircle2,
  Code2,
  Plus,
  Play,
} from "lucide-react";
import { fetchProviders, BUILTIN_PROVIDERS, type ProviderInfo } from "@/lib/providers";
import { applyThemeSettings } from "@/lib/theme";
import type { Definition as CustomProviderDef } from "@bindings/lingo-translate/pkg/translator/custom/models.js";

interface SettingsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

type TabType =
  | "appearance"
  | "routing"
  | "accounts"
  | "keys"
  | "endpoints"
  | "memory_context"
  | "tools"
  | "translation"
  | "about";

export const SettingsDialog: React.FC<SettingsDialogProps> = ({
  open,
  onOpenChange,
}) => {
  const [activeTab, setActiveTab] = useState<TabType>("appearance");
  const [showKeys, setShowKeys] = useState<{ [key: string]: boolean }>({});
  const [providers, setProviders] = useState<ProviderInfo[]>(BUILTIN_PROVIDERS);
  const [isRefreshingPlugins, setIsRefreshingPlugins] = useState(false);
  const [clearingCache, setClearingCache] = useState(false);

  const [settings, setSettings] = useState<Settings>({
    default_provider: "mock",
    default_model: "gemini-3.8-flash",
    gemini_api_key: "",
    openai_api_key: "",
    openai_base_url: "",
    google_api_key: "",
    chanomhub_token: "",
    uchs_api_key: "",
    plugin_keys: {},
    plugin_base_urls: {},
    default_source_lang: "Japanese",
    default_target_lang: "Thai",
    default_batch_size: 10,
    default_concurrency: 4,
    theme: "dark",
    density: "comfortable",
    font_size: "medium",
    context_lore: "",
    translation_style: "standard",
    enable_memory_cache: true,
    tasks: {
      primary_translation: { provider: "gemini", model: "gemini-3.8-flash" },
      fast_translation: { provider: "google", model: "google-translate" },
      fallback_translation: { provider: "gemini", model: "gemini-3.8-flash" },
      auto_route_short_text: false,
      max_short_length: 60,
      enable_fallback: false,
    },
    system_one: {
      enabled: false,
      provider: "typesafe_jev",
      api_key: "",
      confidence_threshold: 0.85,
      features: {
        filter_ambiguous_code: true,
        accept_ui_drafts: true,
        verify_qa: false,
      },
    },
  });
  const [saving, setSaving] = useState(false);
  const [requestingUchsKey, setRequestingUchsKey] = useState(false);
  const [openingUchsPortal, setOpeningUchsPortal] = useState(false);

  const [savedPlugins, setSavedPlugins] = useState<CustomProviderDef[]>([]);
  const [newPlugin, setNewPlugin] = useState({
    name: "",
    display_name: "",
    base_url: "",
    api_key: "",
    default_model: "",
    available_models: "",
  });
  const [isFetchingPluginModels, setIsFetchingPluginModels] = useState(false);
  const [isTestingPlugin, setIsTestingPlugin] = useState(false);
  const [testResult, setTestResult] = useState<string | null>(null);
  const [isSavingPlugin, setIsSavingPlugin] = useState(false);

  const loadProvidersData = async () => {
    try {
      const list = await fetchProviders();
      setProviders(list);
    } catch (err) {
      console.error("Failed to load providers:", err);
    }
  };

  const loadSavedPlugins = async () => {
    try {
      const list = await SettingsService.ListCustomProviders();
      setSavedPlugins(list || []);
    } catch (err) {
      console.error("Failed to load custom provider plugins:", err);
    }
  };

  useEffect(() => {
    if (open) {
      loadProvidersData();
      loadSavedPlugins();
      SettingsService.GetSettings()
        .then((s) => {
          if (s) {
            setSettings({
              ...s,
              uchs_api_key: s.uchs_api_key || "",
              plugin_keys: s.plugin_keys || {},
              plugin_base_urls: s.plugin_base_urls || {},
              theme: s.theme || "dark",
              density: (s as any).density || "comfortable",
              font_size: (s as any).font_size || "medium",
              context_lore: (s as any).context_lore || "",
              translation_style: (s as any).translation_style || "standard",
              enable_memory_cache: (s as any).enable_memory_cache ?? true,
              tasks: {
                primary_translation: s.tasks?.primary_translation || {
                  provider: s.default_provider || "gemini",
                  model: s.default_model || "gemini-3.8-flash",
                },
                fast_translation: s.tasks?.fast_translation || {
                  provider: "google",
                  model: "google-translate",
                },
                fallback_translation: s.tasks?.fallback_translation || {
                  provider: "gemini",
                  model: "gemini-3.8-flash",
                },
                auto_route_short_text: s.tasks?.auto_route_short_text ?? false,
                max_short_length: s.tasks?.max_short_length || 60,
                enable_fallback: s.tasks?.enable_fallback ?? false,
              },
              system_one: s.system_one || {
                enabled: false,
                provider: "typesafe_jev",
                api_key: "",
                confidence_threshold: 0.85,
                features: {
                  filter_ambiguous_code: true,
                  accept_ui_drafts: true,
                  verify_qa: false,
                },
              },
            });
          }
        })
        .catch((err) => {
          console.error("Failed to load settings:", err);
        });
    }
  }, [open]);

  const handleRequestUchsKey = async () => {
    if (!settings.chanomhub_token) {
      toast.error("จำเป็นต้องระบุ Chanomhub Token ก่อนเพื่อขอรับ Key จากระบบ");
      return;
    }
    try {
      setRequestingUchsKey(true);
      const key = await SettingsService.RequestUchsKey(settings.chanomhub_token);
      if (key) {
        setSettings((prev) => ({ ...prev, uchs_api_key: key }));
        toast.success("ออก UCHS API Key ผ่าน Chanomhub เรียบร้อยแล้ว!");
      }
    } catch (err: any) {
      toast.error(err.message || "ไม่สามารถขอรับ UCHS Key ได้");
    } finally {
      setRequestingUchsKey(false);
    }
  };

  const handleOpenUchsPortal = async () => {
    if (!settings.chanomhub_token) {
      toast.error("จำเป็นต้องระบุ Chanomhub Token ก่อนเพื่อเข้าสู่ระบบ");
      return;
    }
    try {
      setOpeningUchsPortal(true);
      await SettingsService.OpenUchsPortal(settings.chanomhub_token);
      toast.success("เปิด UCHS Portal ในเว็บเบราว์เซอร์แล้ว");
    } catch (err: any) {
      toast.error(err.message || "ไม่สามารถเปิด UCHS Portal ได้");
    } finally {
      setOpeningUchsPortal(false);
    }
  };

  const toggleShowKey = (key: string) => {
    setShowKeys((prev) => ({ ...prev, [key]: !prev[key] }));
  };

  const handlePluginKeyChange = (providerName: string, value: string) => {
    setSettings((prev) => ({
      ...prev,
      plugin_keys: {
        ...(prev.plugin_keys || {}),
        [providerName]: value,
      },
    }));
  };

  const handleRefreshPlugins = async () => {
    try {
      setIsRefreshingPlugins(true);
      const list = await fetchProviders();
      setProviders(list);
      toast.success(`Discovered ${list.length} translation providers`);
    } catch (err: any) {
      toast.error(`Plugin reload failed: ${err?.message || err}`);
    } finally {
      setIsRefreshingPlugins(false);
    }
  };

  const handleClearCache = async () => {
    try {
      setClearingCache(true);
      await EntryService.ClearCache();
      toast.success("Translation Memory (TM Cache) cleared successfully!");
    } catch (err: any) {
      toast.info(`Cache reset: ${err?.message || "No active workspace cache to clear"}`);
    } finally {
      setClearingCache(false);
    }
  };

  const handleThemeChange = (newTheme: string) => {
    setSettings((prev) => ({ ...prev, theme: newTheme }));
    applyThemeSettings(newTheme, (settings as any).density, (settings as any).font_size);
  };

  const handleDensityChange = (newDensity: string) => {
    setSettings((prev) => ({ ...prev, density: newDensity }));
    applyThemeSettings(settings.theme, newDensity, (settings as any).font_size);
  };

  const handleFontSizeChange = (newFontSize: string) => {
    setSettings((prev) => ({ ...prev, font_size: newFontSize }));
    applyThemeSettings(settings.theme, (settings as any).density, newFontSize);
  };

  const handleFetchNewPluginModels = async () => {
    if (!newPlugin.base_url.trim()) {
      toast.error("Please enter a Base URL first.");
      return;
    }
    try {
      setIsFetchingPluginModels(true);
      toast.info(`Querying models from ${newPlugin.base_url}...`);
      const models = await SettingsService.FetchRemoteModels(newPlugin.base_url, newPlugin.api_key);
      if (models && models.length > 0) {
        setNewPlugin((prev) => ({
          ...prev,
          available_models: models.join(", "),
          default_model: prev.default_model || models[0],
        }));
        toast.success(`Successfully fetched ${models.length} models from server!`);
      } else {
        toast.warning("Server responded with 0 models.");
      }
    } catch (err: any) {
      toast.error(`Fetch models failed: ${err?.message || err}`);
    } finally {
      setIsFetchingPluginModels(false);
    }
  };

  const handleTestNewPlugin = async () => {
    if (!newPlugin.base_url.trim()) {
      toast.error("Please enter a Base URL.");
      return;
    }
    try {
      setIsTestingPlugin(true);
      setTestResult(null);
      toast.info("Testing endpoint connection...");
      const result = await SettingsService.TestCustomProvider({
        name: newPlugin.name.trim() || "test_endpoint",
        display_name: newPlugin.display_name.trim() || "Test Endpoint",
        base_url: newPlugin.base_url.trim(),
        default_model: newPlugin.default_model.trim() || "default",
        api_key: newPlugin.api_key.trim(),
      });
      setTestResult(result);
      toast.success(`Connection verified! Response: "${result}"`);
    } catch (err: any) {
      setTestResult(`Error: ${err?.message || err}`);
      toast.error(`Connection failed: ${err?.message || err}`);
    } finally {
      setIsTestingPlugin(false);
    }
  };

  const handleSaveCustomPlugin = async () => {
    const rawName = newPlugin.name.trim().toLowerCase().replace(/[^a-z0-9_-]/g, "");
    if (!rawName) {
      toast.error("Please provide a valid plugin identifier name (e.g. ollama-local, vllm-deepseek).");
      return;
    }
    if (!newPlugin.base_url.trim()) {
      toast.error("Base URL is required.");
      return;
    }

    try {
      setIsSavingPlugin(true);
      const modelsList = newPlugin.available_models
        .split(",")
        .map((m) => m.trim())
        .filter(Boolean);

      const def: CustomProviderDef = {
        name: rawName,
        display_name: newPlugin.display_name.trim() || rawName,
        base_url: newPlugin.base_url.trim(),
        default_model: newPlugin.default_model.trim() || (modelsList[0] || "default"),
        available_models: modelsList,
        api_key: newPlugin.api_key.trim(),
      };

      await SettingsService.SaveCustomProvider(def);
      toast.success(`Saved plugin "${rawName}" to ~/.lingo/providers/${rawName}.json!`);
      setNewPlugin({
        name: "",
        display_name: "",
        base_url: "",
        api_key: "",
        default_model: "",
        available_models: "",
      });
      setTestResult(null);
      await loadSavedPlugins();
      await loadProvidersData();
    } catch (err: any) {
      toast.error(`Failed to save plugin: ${err?.message || err}`);
    } finally {
      setIsSavingPlugin(false);
    }
  };

  const handleDeleteCustomPlugin = async (name: string) => {
    try {
      await SettingsService.DeleteCustomProvider(name);
      toast.success(`Removed provider plugin "${name}"`);
      await loadSavedPlugins();
      await loadProvidersData();
    } catch (err: any) {
      toast.error(`Failed to delete plugin: ${err?.message || err}`);
    }
  };

  const handleSave = async () => {
    try {
      setSaving(true);
      await SettingsService.SaveSettings(settings);
      applyThemeSettings(settings.theme, (settings as any).density, (settings as any).font_size);
      toast.success("Settings saved successfully!");
      onOpenChange(false);
    } catch (err: any) {
      toast.error(`Failed to save settings: ${err?.message || err}`);
    } finally {
      setSaving(false);
    }
  };

  const customPlugins = providers.filter((p) => p.is_custom);

  const navItems = [
    { id: "appearance" as TabType, label: "Appearance", icon: Palette },
    { id: "routing" as TabType, label: "Task Routing & Fallback", icon: Bot },
    { id: "accounts" as TabType, label: "Accounts & SSO", icon: UserCheck },
    { id: "keys" as TabType, label: "API Keys", icon: Key },
    { id: "endpoints" as TabType, label: "Custom Endpoints", icon: Server },
    { id: "memory_context" as TabType, label: "Memory & Context", icon: Database },
    { id: "tools" as TabType, label: "Tools & Auxiliary", icon: Cpu },
    { id: "translation" as TabType, label: "Pipeline Defaults", icon: Sliders },
    { id: "about" as TabType, label: "About", icon: Info },
  ];

  const primaryProviderInfo = providers.find(
    (p) => p.name === (settings.tasks?.primary_translation?.provider || settings.default_provider)
  );
  const fastProviderInfo = providers.find(
    (p) => p.name === (settings.tasks?.fast_translation?.provider || "google")
  );
  const fallbackProviderInfo = providers.find(
    (p) => p.name === (settings.tasks?.fallback_translation?.provider || "gemini")
  );

  const resolveAuth = (provName?: string) => {
    if (!provName) return { apiKey: "", baseURL: "" };
    if (provName === "gemini") return { apiKey: settings.gemini_api_key || "", baseURL: "https://generativelanguage.googleapis.com" };
    if (provName === "openai") return { apiKey: settings.openai_api_key || "", baseURL: settings.openai_base_url || "https://api.openai.com/v1" };
    if (provName === "uchs") return { apiKey: settings.uchs_api_key || "", baseURL: "https://ilms.uchs-th.com/v1" };
    if (provName === "google") return { apiKey: settings.google_api_key || "", baseURL: "" };
    if (provName === "ollama") return { apiKey: settings.plugin_keys?.ollama || "ollama", baseURL: settings.plugin_base_urls?.ollama || "http://localhost:11434/v1" };
    const key = settings.plugin_keys?.[provName] || "";
    const base = settings.plugin_base_urls?.[provName] || "";
    return { apiKey: key, baseURL: base };
  };

  const primaryAuth = resolveAuth(settings.tasks?.primary_translation?.provider || settings.default_provider);
  const fastAuth = resolveAuth(settings.tasks?.fast_translation?.provider || "google");
  const fallbackAuth = resolveAuth(settings.tasks?.fallback_translation?.provider || "gemini");

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl h-[620px] p-0 flex flex-col overflow-hidden bg-background border-border">
        <DialogHeader className="p-4 pb-3 border-b border-border">
          <DialogTitle className="flex items-center gap-2 text-base text-foreground">
            <SettingsIcon className="w-5 h-5 text-primary" />
            Preferences &amp; Settings
          </DialogTitle>
        </DialogHeader>

        <div className="flex flex-1 overflow-hidden">
          {/* Left Sidebar Navigation */}
          <div className="w-52 bg-card/60 border-r border-border p-2 space-y-1 overflow-y-auto">
            {navItems.map((item) => {
              const Icon = item.icon;
              const isActive = activeTab === item.id;
              return (
                // @ui-allow-native
                <button
                  key={item.id}
                  onClick={() => setActiveTab(item.id)}
                  className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-md text-xs font-medium transition-colors text-left ${
                    isActive
                      ? "bg-secondary text-foreground font-semibold border-l-2 border-primary"
                      : "text-muted-foreground hover:text-foreground hover:bg-muted"
                  }`}
                >
                  <Icon className={`w-4 h-4 shrink-0 ${isActive ? "text-primary" : "text-muted-foreground"}`} />
                  <span className="truncate">{item.label}</span>
                </button>
              );
            })}
          </div>

          {/* Right Content Area */}
          <div className="flex-1 p-5 overflow-y-auto bg-background text-sm text-foreground">
            {/* 1. General & Appearance Tab */}
            {activeTab === "appearance" && (
              <div className="space-y-5">
                <div>
                  <h3 className="text-sm font-semibold text-foreground">Appearance &amp; Layout</h3>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    Customize user interface theme, window layout density, and typography scaling.
                  </p>
                </div>

                {/* Theme Selection */}
                <div className="space-y-2">
                  <label className="block text-xs font-semibold text-muted-foreground">
                    Color Theme
                  </label>
                  <div className="grid grid-cols-2 gap-2.5">
                    {[
                      { id: "dark", label: "Dark (Default)", desc: "Balanced dark slate background" },
                      { id: "light", label: "Light", desc: "Crisp light background for daytime use" },
                      { id: "midnight", label: "Midnight OLED", desc: "Pure #000000 black background" },
                      { id: "system", label: "System Preference", desc: "Automatically match OS color scheme" },
                    ].map((t) => {
                      const isSelected = settings.theme === t.id;
                      return (
                        // @ui-allow-native
                        <button
                          key={t.id}
                          type="button"
                          onClick={() => handleThemeChange(t.id)}
                          className={`p-3 rounded-lg border text-left transition-all ${
                            isSelected
                              ? "border-primary bg-primary/10 shadow-sm"
                              : "border-border bg-card/60 hover:bg-card"
                          }`}
                        >
                          <div className="flex items-center justify-between">
                            <span className="text-xs font-semibold text-foreground">{t.label}</span>
                            {isSelected && <CheckCircle2 className="w-4 h-4 text-primary" />}
                          </div>
                          <p className="text-[11px] text-muted-foreground mt-1">{t.desc}</p>
                        </button>
                      );
                    })}
                  </div>
                </div>

                {/* Window & Layout Density */}
                <div className="space-y-2 pt-2 border-t border-border/50">
                  <label className="block text-xs font-semibold text-muted-foreground">
                    Table &amp; Window Layout Density
                  </label>
                  <div className="grid grid-cols-2 gap-2.5">
                    {[
                      {
                        id: "comfortable",
                        label: "Comfortable (Default)",
                        desc: "Standard padding and generous row height",
                      },
                      {
                        id: "compact",
                        label: "Compact Mode",
                        desc: "Dense row spacing for viewing thousands of translation lines",
                      },
                    ].map((d) => {
                      const isSelected = (settings as any).density === d.id || (!settings.density && d.id === "comfortable");
                      return (
                        // @ui-allow-native
                        <button
                          key={d.id}
                          type="button"
                          onClick={() => handleDensityChange(d.id)}
                          className={`p-3 rounded-lg border text-left transition-all ${
                            isSelected
                              ? "border-primary bg-primary/10 shadow-sm"
                              : "border-border bg-card/60 hover:bg-card"
                          }`}
                        >
                          <div className="flex items-center justify-between">
                            <span className="text-xs font-semibold text-foreground">{d.label}</span>
                            {isSelected && <CheckCircle2 className="w-4 h-4 text-primary" />}
                          </div>
                          <p className="text-[11px] text-muted-foreground mt-1">{d.desc}</p>
                        </button>
                      );
                    })}
                  </div>
                </div>

                {/* Font Size Scaling */}
                <div className="space-y-2 pt-2 border-t border-border/50">
                  <label className="block text-xs font-semibold text-muted-foreground">
                    UI Font Size Scale
                  </label>
                  <div className="grid grid-cols-3 gap-2">
                    {[
                      { id: "small", label: "Small (13px)" },
                      { id: "medium", label: "Medium (14px - Default)" },
                      { id: "large", label: "Large (15px)" },
                    ].map((f) => {
                      const isSelected = (settings as any).font_size === f.id || (!settings.font_size && f.id === "medium");
                      return (
                        // @ui-allow-native
                        <button
                          key={f.id}
                          type="button"
                          onClick={() => handleFontSizeChange(f.id)}
                          className={`py-2 px-3 rounded-md border text-center text-xs transition-colors ${
                            isSelected
                              ? "border-primary bg-primary/20 text-primary font-semibold"
                              : "border-border bg-card text-muted-foreground hover:text-foreground"
                          }`}
                        >
                          {f.label}
                        </button>
                      );
                    })}
                  </div>
                </div>
              </div>
            )}

            {/* 2. Task Routing & Fallback Tab */}
            {activeTab === "routing" && (
              <div className="space-y-4">
                <div>
                  <div className="flex items-center gap-2">
                    <h3 className="text-sm font-semibold text-foreground">Task Routing &amp; Fallback Engine</h3>
                    <span className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-primary/20 text-primary">
                      HERMES ARCHITECTURE
                    </span>
                  </div>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    Delegate localization workload to specialized models and configure automated failover.
                  </p>
                </div>

                {/* Primary Narrative Role */}
                <div className="p-3 bg-card/70 border border-border rounded-lg space-y-2.5">
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="text-xs font-semibold text-foreground flex items-center gap-1.5">
                        <Sparkles className="w-3.5 h-3.5 text-primary" />
                        Primary Narrative Translation
                      </div>
                      <p className="text-[11px] text-muted-foreground mt-0.5">
                        High-context reasoning model for dialogue, character speech, and emotional storytelling.
                      </p>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-3 pt-1">
                    <div>
                      <label className="block text-[11px] font-medium text-muted-foreground mb-1">
                        Provider
                      </label>
                      {/* @ui-allow-native */}
                      <select
                        value={settings.tasks?.primary_translation?.provider || settings.default_provider}
                        onChange={(e) => {
                          const val = e.target.value;
                          const found = providers.find((p) => p.name === val);
                          const nextModel = found?.default_model || (found?.available_models?.[0] || settings.tasks!.primary_translation.model);
                          setSettings({
                            ...settings,
                            default_provider: val,
                            default_model: nextModel,
                            tasks: {
                              ...settings.tasks!,
                              primary_translation: {
                                ...settings.tasks!.primary_translation,
                                provider: val,
                                model: nextModel,
                              },
                            },
                          });
                        }}
                        className="w-full h-8 rounded-md border border-input bg-card px-2.5 text-xs text-foreground focus:outline-none focus:border-primary"
                      >
                        {providers.map((p) => (
                          <option key={p.name} value={p.name}>
                            {p.display_name}
                          </option>
                        ))}
                      </select>
                    </div>

                    <div>
                      <label className="block text-[11px] font-medium text-muted-foreground mb-1">
                        Model Selection
                      </label>
                      <ModelSelect
                        providerInfo={primaryProviderInfo}
                        value={settings.tasks?.primary_translation?.model || settings.default_model}
                        onChange={(val) =>
                          setSettings({
                            ...settings,
                            default_model: val,
                            tasks: {
                              ...settings.tasks!,
                              primary_translation: {
                                ...settings.tasks!.primary_translation,
                                model: val,
                              },
                            },
                          })
                        }
                        apiKey={primaryAuth.apiKey}
                        baseURL={primaryAuth.baseURL}
                      />
                    </div>
                  </div>
                </div>

                {/* Fast / Bulk UI Translation Role */}
                <div className="p-3 bg-card/70 border border-border rounded-lg space-y-2.5">
                  <div>
                    <div className="text-xs font-semibold text-foreground flex items-center gap-1.5">
                      <Zap className="w-3.5 h-3.5 text-amber-400" />
                      Fast / Economy Translation
                    </div>
                    <p className="text-[11px] text-muted-foreground mt-0.5">
                      High-throughput, cost-effective engine for UI menus, item names, skills, and combat logs.
                    </p>
                  </div>

                  <div className="grid grid-cols-2 gap-3 pt-1">
                    <div>
                      <label className="block text-[11px] font-medium text-muted-foreground mb-1">
                        Provider
                      </label>
                      {/* @ui-allow-native */}
                      <select
                        value={settings.tasks?.fast_translation?.provider || "google"}
                        onChange={(e) => {
                          const val = e.target.value;
                          const found = providers.find((p) => p.name === val);
                          const nextModel = found?.default_model || (found?.available_models?.[0] || "google-translate");
                          setSettings({
                            ...settings,
                            tasks: {
                              ...settings.tasks!,
                              fast_translation: {
                                ...settings.tasks!.fast_translation,
                                provider: val,
                                model: nextModel,
                              },
                            },
                          });
                        }}
                        className="w-full h-8 rounded-md border border-input bg-card px-2.5 text-xs text-foreground focus:outline-none focus:border-primary"
                      >
                        {providers.map((p) => (
                          <option key={p.name} value={p.name}>
                            {p.display_name}
                          </option>
                        ))}
                      </select>
                    </div>

                    <div>
                      <label className="block text-[11px] font-medium text-muted-foreground mb-1">
                        Model Selection
                      </label>
                      <ModelSelect
                        providerInfo={fastProviderInfo}
                        value={settings.tasks?.fast_translation?.model || "google-translate"}
                        onChange={(val) =>
                          setSettings({
                            ...settings,
                            tasks: {
                              ...settings.tasks!,
                              fast_translation: {
                                ...settings.tasks!.fast_translation,
                                model: val,
                              },
                            },
                          })
                        }
                        apiKey={fastAuth.apiKey}
                        baseURL={fastAuth.baseURL}
                      />
                    </div>
                  </div>

                  <div className="flex items-center justify-between pt-2 border-t border-border/50 text-xs">
                    <label className="flex items-center gap-2 cursor-pointer select-none text-muted-foreground hover:text-foreground">
                      {/* @ui-allow-native */}
                      <input
                        type="checkbox"
                        checked={settings.tasks?.auto_route_short_text ?? false}
                        onChange={(e) =>
                          setSettings({
                            ...settings,
                            tasks: {
                              ...settings.tasks!,
                              auto_route_short_text: e.target.checked,
                            },
                          })
                        }
                        className="rounded border-border text-primary focus:ring-primary"
                      />
                      <span>Auto-route short texts to Fast Model</span>
                    </label>

                    <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                      <span>Max length:</span>
                      <Input
                        type="number"
                        min={10}
                        max={300}
                        className="w-16 h-7 text-xs px-2 py-0 text-center font-mono"
                        value={settings.tasks?.max_short_length || 60}
                        onChange={(e) =>
                          setSettings({
                            ...settings,
                            tasks: {
                              ...settings.tasks!,
                              max_short_length: parseInt(e.target.value) || 60,
                            },
                          })
                        }
                      />
                      <span>chars</span>
                    </div>
                  </div>
                </div>

                {/* Automated Fallback Failover Role */}
                <div className="p-3 bg-card/70 border border-border rounded-lg space-y-2.5">
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="text-xs font-semibold text-foreground flex items-center gap-1.5">
                        <ShieldAlert className="w-3.5 h-3.5 text-amber-400" />
                        Automatic API Failover (Fallback Engine)
                      </div>
                      <p className="text-[11px] text-muted-foreground mt-0.5">
                        If primary translation provider fails due to rate limits (429), quota exhaustion, or timeout, automatically retry with fallback.
                      </p>
                    </div>

                    <label className="flex items-center gap-1.5 text-xs cursor-pointer select-none font-semibold text-foreground">
                      {/* @ui-allow-native */}
                      <input
                        type="checkbox"
                        checked={settings.tasks?.enable_fallback ?? false}
                        onChange={(e) =>
                          setSettings({
                            ...settings,
                            tasks: {
                              ...settings.tasks!,
                              enable_fallback: e.target.checked,
                            },
                          })
                        }
                        className="rounded border-border text-primary focus:ring-primary"
                      />
                      <span>Enable Fallback</span>
                    </label>
                  </div>

                  {settings.tasks?.enable_fallback && (
                    <div className="grid grid-cols-2 gap-3 pt-1 border-t border-border/50">
                      <div>
                        <label className="block text-[11px] font-medium text-muted-foreground mb-1">
                          Fallback Provider
                        </label>
                        {/* @ui-allow-native */}
                        <select
                          value={settings.tasks?.fallback_translation?.provider || "gemini"}
                          onChange={(e) => {
                            const val = e.target.value;
                            const found = providers.find((p) => p.name === val);
                            const nextModel = found?.default_model || (found?.available_models?.[0] || "gemini-3.8-flash");
                            setSettings({
                              ...settings,
                              tasks: {
                                ...settings.tasks!,
                                fallback_translation: {
                                  ...settings.tasks!.fallback_translation,
                                  provider: val,
                                  model: nextModel,
                                },
                              },
                            });
                          }}
                          className="w-full h-8 rounded-md border border-input bg-card px-2.5 text-xs text-foreground focus:outline-none focus:border-primary"
                        >
                          {providers.map((p) => (
                            <option key={p.name} value={p.name}>
                              {p.display_name}
                            </option>
                          ))}
                        </select>
                      </div>

                      <div>
                        <label className="block text-[11px] font-medium text-muted-foreground mb-1">
                          Fallback Model Selection
                        </label>
                        <ModelSelect
                          providerInfo={fallbackProviderInfo}
                          value={settings.tasks?.fallback_translation?.model || "gemini-3.8-flash"}
                          onChange={(val) =>
                            setSettings({
                              ...settings,
                              tasks: {
                                ...settings.tasks!,
                                fallback_translation: {
                                  ...settings.tasks!.fallback_translation,
                                  model: val,
                                },
                              },
                            })
                          }
                          showChips={false}
                          apiKey={fallbackAuth.apiKey}
                          baseURL={fallbackAuth.baseURL}
                        />
                      </div>
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* 3. Accounts & SSO Tab */}
            {activeTab === "accounts" && (
              <div className="space-y-4">
                <div>
                  <h3 className="text-sm font-semibold text-foreground">Accounts &amp; SSO Integrations</h3>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    Manage community authentication and cloud gateway access keys.
                  </p>
                </div>

                {/* Chanomhub Account */}
                <div className="p-3.5 bg-card/70 border border-border rounded-lg space-y-3">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <UserCheck className="w-4 h-4 text-primary" />
                      <span className="text-xs font-semibold text-foreground">
                        Chanomhub Community Account
                      </span>
                    </div>
                    <span
                      className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                        settings.chanomhub_token
                          ? "bg-emerald-950/60 text-emerald-400 border border-emerald-500/30"
                          : "bg-muted text-muted-foreground"
                      }`}
                    >
                      {settings.chanomhub_token ? "CONNECTED" : "NOT CONNECTED"}
                    </span>
                  </div>

                  <p className="text-[11px] text-muted-foreground leading-relaxed">
                    Required for publishing game translation mods to the community hub and authorizing UCHS AI virtual keys.
                  </p>

                  <div className="space-y-1">
                    <label className="block text-[11px] font-medium text-muted-foreground">
                      Chanomhub JWT Token
                    </label>
                    <div className="flex gap-2">
                      <Input
                        type={showKeys["chanomhub"] ? "text" : "password"}
                        value={settings.chanomhub_token}
                        onChange={(e) =>
                          setSettings({ ...settings, chanomhub_token: e.target.value })
                        }
                        placeholder="eyJhbGciOi..."
                        className="font-mono text-xs"
                      />
                      <Button
                        type="button"
                        variant="secondary"
                        size="icon"
                        className="h-8 w-8 shrink-0"
                        onClick={() => toggleShowKey("chanomhub")}
                      >
                        {showKeys["chanomhub"] ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                      </Button>
                    </div>
                  </div>
                </div>

                {/* UCHS AI Infrastructure Gateway */}
                <div className="p-3.5 bg-emerald-950/20 border border-emerald-500/30 rounded-lg space-y-3">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <Cpu className="w-4 h-4 text-emerald-400" />
                      <span className="text-xs font-semibold text-emerald-300">
                        UCHS AI Infrastructure Gateway (LiteLLM Cluster)
                      </span>
                    </div>
                    <span className="text-[10px] text-emerald-400/80 font-mono">
                      ilms.uchs-th.com
                    </span>
                  </div>

                  <p className="text-[11px] text-muted-foreground leading-relaxed">
                    Access high-throughput DeepSeek V4.1 Flash and Pro translation clusters with zero per-token cost using your Chanomhub virtual key.
                  </p>

                  <div className="space-y-1">
                    <label className="block text-[11px] font-medium text-emerald-300/80">
                      UCHS Virtual API Key
                    </label>
                    <div className="flex gap-2">
                      <Input
                        type={showKeys["uchs"] ? "text" : "password"}
                        value={settings.uchs_api_key || ""}
                        onChange={(e) =>
                          setSettings({ ...settings, uchs_api_key: e.target.value })
                        }
                        placeholder="sk-uchs-..."
                        className="font-mono text-xs border-emerald-500/30"
                      />
                      <Button
                        type="button"
                        variant="secondary"
                        size="icon"
                        className="h-8 w-8 shrink-0"
                        onClick={() => toggleShowKey("uchs")}
                      >
                        {showKeys["uchs"] ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                      </Button>
                    </div>
                  </div>

                  <div className="flex flex-wrap gap-2 pt-1">
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      disabled={requestingUchsKey || !settings.chanomhub_token}
                      onClick={handleRequestUchsKey}
                      className="text-xs h-7 gap-1.5 border-emerald-500/40 text-emerald-400 hover:bg-emerald-500/10"
                    >
                      <Sparkles className="w-3 h-3 text-emerald-400" />
                      {requestingUchsKey ? "กำลังขอ Key..." : "ขอรับ Key จาก Chanomhub"}
                    </Button>

                    <Button
                      type="button"
                      size="sm"
                      variant="ghost"
                      disabled={openingUchsPortal || !settings.chanomhub_token}
                      onClick={handleOpenUchsPortal}
                      className="text-xs h-7 gap-1.5 text-zinc-300 hover:text-white"
                    >
                      <ExternalLink className="w-3 h-3 text-emerald-400" />
                      {openingUchsPortal ? "กำลังเปิด..." : "เปิดหน้าบัญชี UCHS (Portal)"}
                    </Button>
                  </div>
                </div>
              </div>
            )}

            {/* 4. API Keys Tab */}
            {activeTab === "keys" && (
              <div className="space-y-4">
                <div>
                  <h3 className="text-sm font-semibold text-foreground">API Keys &amp; Authentication</h3>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    Credentials are saved securely in your local user config.
                  </p>
                </div>

                <div className="space-y-3 pt-1">
                  <div>
                    <label className="block text-xs font-semibold text-muted-foreground mb-1">
                      Google Gemini API Key
                    </label>
                    <div className="flex gap-2">
                      <Input
                        type={showKeys["gemini"] ? "text" : "password"}
                        value={settings.gemini_api_key}
                        onChange={(e) =>
                          setSettings({ ...settings, gemini_api_key: e.target.value })
                        }
                        placeholder="AIzaSy..."
                      />
                      <Button
                        type="button"
                        variant="secondary"
                        size="icon"
                        className="h-8 w-8 shrink-0"
                        onClick={() => toggleShowKey("gemini")}
                      >
                        {showKeys["gemini"] ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                      </Button>
                    </div>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-muted-foreground mb-1">
                      OpenAI API Key
                    </label>
                    <div className="flex gap-2">
                      <Input
                        type={showKeys["openai"] ? "text" : "password"}
                        value={settings.openai_api_key}
                        onChange={(e) =>
                          setSettings({ ...settings, openai_api_key: e.target.value })
                        }
                        placeholder="sk-..."
                      />
                      <Button
                        type="button"
                        variant="secondary"
                        size="icon"
                        className="h-8 w-8 shrink-0"
                        onClick={() => toggleShowKey("openai")}
                      >
                        {showKeys["openai"] ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                      </Button>
                    </div>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-muted-foreground mb-1">
                      Google Translate Cloud API Key (Optional)
                    </label>
                    <div className="flex gap-2">
                      <Input
                        type={showKeys["google"] ? "text" : "password"}
                        value={settings.google_api_key}
                        onChange={(e) =>
                          setSettings({ ...settings, google_api_key: e.target.value })
                        }
                        placeholder="Cloud Translation API Key"
                      />
                      <Button
                        type="button"
                        variant="secondary"
                        size="icon"
                        className="h-8 w-8 shrink-0"
                        onClick={() => toggleShowKey("google")}
                      >
                        {showKeys["google"] ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                      </Button>
                    </div>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-muted-foreground mb-1">
                      TypeSafe AI API Key (System One - Jev)
                    </label>
                    <div className="flex gap-2">
                      <Input
                        type={showKeys["jev"] ? "text" : "password"}
                        value={settings.system_one?.api_key || ""}
                        onChange={(e) =>
                          setSettings({
                            ...settings,
                            system_one: {
                              ...settings.system_one!,
                              api_key: e.target.value,
                            },
                          })
                        }
                        placeholder="ts_live_..."
                      />
                      <Button
                        type="button"
                        variant="secondary"
                        size="icon"
                        className="h-8 w-8 shrink-0"
                        onClick={() => toggleShowKey("jev")}
                      >
                        {showKeys["jev"] ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                      </Button>
                    </div>
                  </div>

                  {/* Dynamic External Plugin Credentials */}
                  {customPlugins.length > 0 && (
                    <div className="border-t border-border pt-3 space-y-3">
                      <div className="text-xs font-semibold text-amber-400 uppercase tracking-wider flex items-center gap-1.5">
                        <Layers className="w-3.5 h-3.5" />
                        External Plugins ({customPlugins.length})
                      </div>

                      {customPlugins.map((plugin) => (
                        <div key={plugin.name} className="p-3 bg-card border border-border rounded-md space-y-2">
                          <div className="flex justify-between items-center">
                            <label className="block text-xs font-semibold text-foreground">
                              {plugin.display_name} API Key
                            </label>
                            <span className="text-[10px] text-amber-400 font-mono">
                              {plugin.name}
                            </span>
                          </div>
                          <div className="flex gap-2">
                            <Input
                              type={showKeys[plugin.name] ? "text" : "password"}
                              value={
                                settings.plugin_keys?.[plugin.name] ||
                                (settings as any)[`${plugin.name}_api_key`] ||
                                ""
                              }
                              onChange={(e) => handlePluginKeyChange(plugin.name, e.target.value)}
                              placeholder={`Enter API Key for ${plugin.display_name}`}
                            />
                            <Button
                              type="button"
                              variant="secondary"
                              size="icon"
                              className="h-8 w-8 shrink-0"
                              onClick={() => toggleShowKey(plugin.name)}
                            >
                              {showKeys[plugin.name] ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                            </Button>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* 5. Custom Endpoints Tab */}
            {activeTab === "endpoints" && (
              <div className="space-y-5">
                <div>
                  <h3 className="text-sm font-semibold text-foreground">Custom Endpoints &amp; Local LLMs</h3>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    Connect local offline engines (Ollama, LM Studio, vLLM) or OpenAI-compatible gateways, and save them as reusable provider plugins.
                  </p>
                </div>

                {/* 1. Global OpenAI Base URL Override */}
                <div className="p-3.5 bg-card/70 border border-border rounded-lg space-y-3">
                  <div>
                    <label className="block text-xs font-semibold text-foreground mb-1">
                      OpenAI API Base URL Override
                    </label>
                    <Input
                      value={settings.openai_base_url}
                      onChange={(e) => setSettings({ ...settings, openai_base_url: e.target.value })}
                      placeholder="https://api.openai.com/v1"
                    />
                  </div>

                  <div>
                    <label className="block text-[11px] font-medium text-muted-foreground mb-1.5">
                      Quick Server Presets
                    </label>
                    <div className="flex flex-wrap gap-2">
                      {[
                        { label: "Ollama (11434)", url: "http://localhost:11434/v1", defaultName: "ollama-local", defaultLabel: "Ollama Local" },
                        { label: "LM Studio (1234)", url: "http://localhost:1234/v1", defaultName: "lm-studio", defaultLabel: "LM Studio" },
                        { label: "vLLM (8000)", url: "http://localhost:8000/v1", defaultName: "vllm-server", defaultLabel: "vLLM Server" },
                        { label: "OpenRouter", url: "https://openrouter.ai/api/v1", defaultName: "openrouter-custom", defaultLabel: "OpenRouter Custom" },
                        { label: "OpenAI Official", url: "https://api.openai.com/v1", defaultName: "openai-custom", defaultLabel: "OpenAI Custom" },
                      ].map((preset) => (
                        // @ui-allow-native
                        <button
                          key={preset.url}
                          type="button"
                          onClick={() => {
                            setSettings({ ...settings, openai_base_url: preset.url });
                            setNewPlugin((prev) => ({
                              ...prev,
                              name: prev.name || preset.defaultName,
                              display_name: prev.display_name || preset.defaultLabel,
                              base_url: preset.url,
                            }));
                          }}
                          className={`px-2.5 py-1 rounded text-xs border font-mono transition-colors ${
                            settings.openai_base_url === preset.url
                              ? "bg-primary/20 text-primary border-primary/50 font-semibold"
                              : "bg-muted text-muted-foreground border-border hover:text-foreground hover:bg-secondary"
                          }`}
                        >
                          {preset.label}
                        </button>
                      ))}
                    </div>
                  </div>
                </div>

                {/* 2. Save Custom Endpoint as Provider Plugin */}
                <div className="p-4 bg-card/80 border border-primary/30 rounded-lg space-y-3.5 shadow-sm">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <div className="w-6 h-6 rounded bg-primary/20 flex items-center justify-center text-primary">
                        <Plus className="w-3.5 h-3.5" />
                      </div>
                      <div>
                        <span className="text-xs font-bold text-foreground">
                          Save Custom Endpoint as Provider Plugin
                        </span>
                        <p className="text-[10px] text-muted-foreground">
                          Saves to <code className="font-mono text-primary/80">~/.lingo/providers/&lt;name&gt;.json</code> for 1st-class selection across GUI &amp; CLI.
                        </p>
                      </div>
                    </div>
                    <span className="px-1.5 py-0.5 rounded text-[10px] font-mono bg-secondary text-secondary-foreground border border-border">
                      PLUGIN GENERATOR
                    </span>
                  </div>

                  <div className="grid grid-cols-2 gap-3 pt-1">
                    <div>
                      <label className="block text-[11px] font-medium text-muted-foreground mb-1">
                        Plugin Identifier ID (e.g. ollama-local)
                      </label>
                      <Input
                        value={newPlugin.name}
                        onChange={(e) =>
                          setNewPlugin({
                            ...newPlugin,
                            name: e.target.value.toLowerCase().replace(/[^a-z0-9_-]/g, ""),
                          })
                        }
                        placeholder="e.g. ollama-local"
                        className="font-mono text-xs"
                      />
                    </div>
                    <div>
                      <label className="block text-[11px] font-medium text-muted-foreground mb-1">
                        Display Label (e.g. Ollama Local)
                      </label>
                      <Input
                        value={newPlugin.display_name}
                        onChange={(e) => setNewPlugin({ ...newPlugin, display_name: e.target.value })}
                        placeholder="e.g. Ollama Local Qwen"
                        className="text-xs"
                      />
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="block text-[11px] font-medium text-muted-foreground mb-1">
                        Endpoint Base URL
                      </label>
                      <Input
                        value={newPlugin.base_url}
                        onChange={(e) => setNewPlugin({ ...newPlugin, base_url: e.target.value })}
                        placeholder="http://localhost:11434/v1"
                        className="font-mono text-xs"
                      />
                    </div>
                    <div>
                      <label className="block text-[11px] font-medium text-muted-foreground mb-1">
                        API Key (Optional / if required)
                      </label>
                      <Input
                        type="password"
                        value={newPlugin.api_key}
                        onChange={(e) => setNewPlugin({ ...newPlugin, api_key: e.target.value })}
                        placeholder="sk-... or ollama"
                        className="font-mono text-xs"
                      />
                    </div>
                  </div>

                  {/* Models list and live query */}
                  <div className="space-y-1.5">
                    <div className="flex justify-between items-center">
                      <label className="block text-[11px] font-medium text-muted-foreground">
                        Available Models (comma-separated list)
                      </label>
                      <Button
                        type="button"
                        variant="secondary"
                        size="sm"
                        onClick={handleFetchNewPluginModels}
                        disabled={isFetchingPluginModels || !newPlugin.base_url.trim()}
                        className="h-6 text-[10px] gap-1 px-2 border border-border"
                      >
                        <RefreshCw className={`w-3 h-3 text-primary ${isFetchingPluginModels ? "animate-spin" : ""}`} />
                        <span>⚡ Fetch Models from Server</span>
                      </Button>
                    </div>
                    <Input
                      value={newPlugin.available_models}
                      onChange={(e) => setNewPlugin({ ...newPlugin, available_models: e.target.value })}
                      placeholder="qwen2.5:latest, llama3.2:latest, deepseek-r1:8b"
                      className="font-mono text-xs"
                    />
                  </div>

                  <div>
                    <label className="block text-[11px] font-medium text-muted-foreground mb-1">
                      Default Model
                    </label>
                    <Input
                      value={newPlugin.default_model}
                      onChange={(e) => setNewPlugin({ ...newPlugin, default_model: e.target.value })}
                      placeholder="e.g. qwen2.5:latest"
                      className="font-mono text-xs"
                    />
                  </div>

                  {/* Test Result Banner */}
                  {testResult && (
                    <div
                      className={`p-2.5 rounded-md text-xs font-mono border ${
                        testResult.startsWith("Error:")
                          ? "bg-destructive/15 text-destructive border-destructive/30"
                          : "bg-emerald-950/40 text-emerald-300 border-emerald-500/30"
                      }`}
                    >
                      {testResult}
                    </div>
                  )}

                  {/* Plugin Actions */}
                  <div className="flex justify-end gap-2 pt-2 border-t border-border/60">
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={handleTestNewPlugin}
                      disabled={isTestingPlugin || !newPlugin.base_url.trim()}
                      className="gap-1.5 text-xs h-8"
                    >
                      <Play className="w-3.5 h-3.5 text-primary" />
                      <span>{isTestingPlugin ? "Testing..." : "Test Connection"}</span>
                    </Button>
                    <Button
                      type="button"
                      variant="default"
                      size="sm"
                      onClick={handleSaveCustomPlugin}
                      disabled={isSavingPlugin || !newPlugin.name.trim() || !newPlugin.base_url.trim()}
                      className="gap-1.5 text-xs h-8 font-semibold"
                    >
                      <Save className="w-3.5 h-3.5" />
                      <span>{isSavingPlugin ? "Saving..." : "Save as Provider Plugin"}</span>
                    </Button>
                  </div>
                </div>

                {/* 3. Registered Custom Provider Plugins List */}
                <div className="space-y-3 pt-2">
                  <div className="flex items-center justify-between">
                    <div className="text-xs font-semibold text-foreground flex items-center gap-1.5">
                      <Layers className="w-3.5 h-3.5 text-primary" />
                      Registered Provider Plugins ({savedPlugins.length || customPlugins.length})
                    </div>
                    <span className="text-[10px] text-muted-foreground font-mono">
                      ~/.lingo/providers/
                    </span>
                  </div>

                  {(savedPlugins.length > 0 ? savedPlugins : customPlugins).length > 0 ? (
                    <div className="space-y-2">
                      {(savedPlugins.length > 0 ? savedPlugins : customPlugins).map((plugin: any) => (
                        <div
                          key={plugin.name}
                          className="p-3 bg-card border border-border rounded-lg space-y-2.5 transition-colors hover:border-primary/40"
                        >
                          <div className="flex items-center justify-between">
                            <div className="flex items-center gap-2">
                              <span className="text-xs font-bold text-foreground">
                                {plugin.display_name || plugin.name}
                              </span>
                              <span className="px-1.5 py-0.5 rounded text-[10px] font-mono bg-primary/10 text-primary border border-primary/30">
                                {plugin.name}
                              </span>
                            </div>
                            <Button
                              type="button"
                              variant="ghost"
                              size="sm"
                              onClick={() => handleDeleteCustomPlugin(plugin.name)}
                              className="h-7 w-7 p-0 text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                              title="Delete provider plugin"
                            >
                              <Trash2 className="w-3.5 h-3.5" />
                            </Button>
                          </div>

                          <div className="grid grid-cols-2 gap-2 text-[11px] font-mono text-muted-foreground">
                            <div className="truncate">
                              <span className="text-foreground/70">Base URL: </span>
                              <span className="text-primary/90">{plugin.base_url}</span>
                            </div>
                            <div className="truncate">
                              <span className="text-foreground/70">Default Model: </span>
                              <span className="text-foreground">{plugin.default_model || "None"}</span>
                            </div>
                          </div>

                          {plugin.available_models && plugin.available_models.length > 0 && (
                            <div className="flex flex-wrap gap-1 pt-1">
                              {plugin.available_models.map((m: string) => (
                                <span
                                  key={m}
                                  className="px-1.5 py-0.5 rounded text-[10px] font-mono bg-muted text-muted-foreground border border-border"
                                >
                                  {m}
                                </span>
                              ))}
                            </div>
                          )}
                        </div>
                      ))}
                    </div>
                  ) : (
                    <div className="p-4 rounded-lg border border-dashed border-border text-center text-xs text-muted-foreground">
                      No custom provider plugins saved yet. Use the generator above to save your local Ollama, vLLM, or LM Studio endpoint!
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* 6. Memory & Context Tab */}
            {activeTab === "memory_context" && (
              <div className="space-y-4">
                <div>
                  <h3 className="text-sm font-semibold text-foreground">Translation Memory &amp; Context</h3>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    Manage translation reuse cache, persona framing, and custom game directives.
                  </p>
                </div>

                {/* Translation Memory (TM Cache) */}
                <div className="p-3.5 bg-card/70 border border-border rounded-lg space-y-3">
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="text-xs font-semibold text-foreground flex items-center gap-1.5">
                        <Database className="w-3.5 h-3.5 text-emerald-400" />
                        Translation Memory (TM Cache)
                      </div>
                      <p className="text-[11px] text-muted-foreground mt-0.5">
                        Automatically reuses identical translations instantly from local SQLite, avoiding duplicate AI costs.
                      </p>
                    </div>

                    <label className="flex items-center gap-2 text-xs font-semibold cursor-pointer select-none">
                      {/* @ui-allow-native */}
                      <input
                        type="checkbox"
                        checked={(settings as any).enable_memory_cache ?? true}
                        onChange={(e) =>
                          setSettings({
                            ...settings,
                            enable_memory_cache: e.target.checked,
                          })
                        }
                        className="rounded border-border text-primary focus:ring-primary"
                      />
                      <span>Active</span>
                    </label>
                  </div>

                  <div className="flex justify-between items-center pt-2 border-t border-border/50 text-xs text-muted-foreground">
                    <span>Cache Storage: Embedded in project .nst database</span>
                    <Button
                      type="button"
                      variant="destructive"
                      size="sm"
                      disabled={clearingCache}
                      onClick={handleClearCache}
                      className="h-7 text-xs gap-1.5"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                      {clearingCache ? "Clearing..." : "Clear Translation Cache"}
                    </Button>
                  </div>
                </div>

                {/* Tone & Persona Preset */}
                <div className="p-3.5 bg-card/70 border border-border rounded-lg space-y-2.5">
                  <label className="block text-xs font-semibold text-foreground">
                    Default Localization Tone &amp; Persona
                  </label>
                  {/* @ui-allow-native */}
                  <select
                    value={(settings as any).translation_style || "standard"}
                    onChange={(e) =>
                      setSettings({
                        ...settings,
                        translation_style: e.target.value,
                      })
                    }
                    className="w-full h-8 rounded-md border border-input bg-card px-2.5 text-xs text-foreground focus:outline-none focus:border-primary"
                  >
                    <option value="standard">Standard Game Localization (Balanced natural dialogue)</option>
                    <option value="nsfw">NSFW / Adult / Erotic (Uncensored 18+)</option>
                    <option value="vn_romance">Visual Novel &amp; Romance (Intimacy &amp; Dialogue particles)</option>
                    <option value="fantasy_rpg">Fantasy RPG &amp; Medieval (Immersive fantasy terminology)</option>
                    <option value="comedy">Comedy &amp; Playful (Humorous, witty timing)</option>
                    <option value="dan_uncensored">Anti-Censorship Jailbreak (Anti-refusal directive)</option>
                  </select>
                </div>

                {/* Custom Game Lore & Directives */}
                <div className="p-3.5 bg-card/70 border border-border rounded-lg space-y-2">
                  <label className="block text-xs font-semibold text-foreground">
                    Global Game Lore &amp; Context Directives
                  </label>
                  <p className="text-[11px] text-muted-foreground">
                    Define persistent game background, character relationships, pronoun rules, or terms injected into AI prompts.
                  </p>
                  {/* @ui-allow-native */}
                  <textarea
                    rows={4}
                    value={(settings as any).context_lore || ""}
                    onChange={(e) =>
                      setSettings({
                        ...settings,
                        context_lore: e.target.value,
                      })
                    }
                    placeholder="e.g. พระเอกชื่อ เคน, นางเอกชื่อ ยูนะ, เซตติ้งโรงเรียนเวทมนตร์, ห้ามแปลชื่อเวทมนตร์"
                    className="w-full rounded-md border border-input bg-card p-2 text-xs text-foreground focus:outline-none focus:border-primary resize-none"
                  />
                </div>
              </div>
            )}

            {/* 7. Tools & Auxiliary Tab */}
            {activeTab === "tools" && (
              <div className="space-y-4">
                <div>
                  <h3 className="text-sm font-semibold text-foreground">Tools &amp; Auxiliary Services</h3>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    Non-autoregressive decision engine, Model Context Protocol (MCP), and external plugin drivers.
                  </p>
                </div>

                {/* System One (Jev) */}
                <div className="p-3.5 bg-card/70 border border-border rounded-lg space-y-3">
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="flex items-center gap-2">
                        <Cpu className="w-4 h-4 text-primary" />
                        <span className="text-xs font-semibold text-foreground">
                          System One Decision Engine (TypeSafe AI Jev)
                        </span>
                      </div>
                      <p className="text-[11px] text-muted-foreground mt-0.5">
                        Ultra-low latency non-autoregressive decision layer for code filtering and speculative draft acceptance.
                      </p>
                    </div>

                    <Button
                      type="button"
                      variant={settings.system_one?.enabled ? "default" : "secondary"}
                      size="sm"
                      className="h-7 text-xs"
                      onClick={() =>
                        setSettings({
                          ...settings,
                          system_one: {
                            ...settings.system_one!,
                            enabled: !settings.system_one?.enabled,
                          },
                        })
                      }
                    >
                      {settings.system_one?.enabled ? "Enabled" : "Enable Jev"}
                    </Button>
                  </div>

                  {settings.system_one?.enabled && (
                    <div className="grid grid-cols-2 gap-3 pt-2 border-t border-border/50">
                      <div>
                        <label className="block text-[11px] font-medium text-muted-foreground mb-1">
                          Engine Provider
                        </label>
                        {/* @ui-allow-native */}
                        <select
                          value={settings.system_one?.provider || "typesafe_jev"}
                          onChange={(e) =>
                            setSettings({
                              ...settings,
                              system_one: {
                                ...settings.system_one!,
                                provider: e.target.value,
                              },
                            })
                          }
                          className="w-full h-8 rounded-md border border-input bg-card px-2.5 text-xs text-foreground focus:outline-none focus:border-primary"
                        >
                          <option value="typesafe_jev">TypeSafe AI (Jev Cloud)</option>
                          <option value="heuristic">Built-in Deterministic Heuristic</option>
                        </select>
                      </div>

                      <div>
                        <label className="block text-[11px] font-medium text-muted-foreground mb-1">
                          Confidence Threshold
                        </label>
                        <Input
                          type="number"
                          step="0.05"
                          min="0.5"
                          max="1.0"
                          value={settings.system_one?.confidence_threshold || 0.85}
                          onChange={(e) =>
                            setSettings({
                              ...settings,
                              system_one: {
                                ...settings.system_one!,
                                confidence_threshold: parseFloat(e.target.value) || 0.85,
                              },
                            })
                          }
                          className="h-8 text-xs font-mono"
                        />
                      </div>
                    </div>
                  )}
                </div>

                {/* MCP Server Overview */}
                <div className="p-3.5 bg-card/70 border border-border rounded-lg space-y-2">
                  <div className="flex items-center gap-2">
                    <Code2 className="w-4 h-4 text-primary" />
                    <span className="text-xs font-semibold text-foreground">
                      Model Context Protocol (MCP) Server
                    </span>
                  </div>
                  <p className="text-[11px] text-muted-foreground leading-relaxed">
                    Exposes translation workspace tools to external AI IDEs (Antigravity, Claude Desktop, Cursor) over JSON-RPC stdio.
                  </p>
                  <div className="flex flex-wrap gap-1.5 pt-1">
                    {[
                      "list_files",
                      "query_entries",
                      "translate_entries",
                      "export_patch",
                      "inspect_game",
                    ].map((tool) => (
                      <span
                        key={tool}
                        className="px-2 py-0.5 rounded text-[10px] font-mono bg-muted text-muted-foreground border border-border"
                      >
                        {tool}
                      </span>
                    ))}
                  </div>
                </div>

                {/* External Provider Plugins list */}
                <div className="p-3.5 bg-card/70 border border-border rounded-lg space-y-2.5">
                  <div className="flex justify-between items-center">
                    <span className="text-xs font-semibold text-foreground">
                      Discovered Plugins ({customPlugins.length})
                    </span>
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={handleRefreshPlugins}
                      disabled={isRefreshingPlugins}
                      className="h-7 text-xs gap-1.5"
                    >
                      <RefreshCw className={`w-3 h-3 ${isRefreshingPlugins ? "animate-spin" : ""}`} />
                      Refresh
                    </Button>
                  </div>
                  <p className="text-[11px] text-muted-foreground">
                    Drop declarative <code>.json</code> provider files into <code>~/.config/lingo/providers/</code> to extend engines.
                  </p>
                </div>
              </div>
            )}

            {/* 8. Pipeline Defaults Tab */}
            {activeTab === "translation" && (
              <div className="space-y-4">
                <div>
                  <h3 className="text-sm font-semibold text-foreground">Pipeline Defaults</h3>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    Configure default languages, batch sizing, and concurrency workers.
                  </p>
                </div>

                <div className="space-y-3 pt-2">
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="block text-xs font-semibold text-muted-foreground mb-1">
                        Default Source Language
                      </label>
                      <Input
                        value={settings.default_source_lang}
                        onChange={(e) =>
                          setSettings({ ...settings, default_source_lang: e.target.value })
                        }
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-semibold text-muted-foreground mb-1">
                        Default Target Language
                      </label>
                      <Input
                        value={settings.default_target_lang}
                        onChange={(e) =>
                          setSettings({ ...settings, default_target_lang: e.target.value })
                        }
                      />
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="block text-xs font-semibold text-muted-foreground mb-1">
                        Batch Size (lines per request)
                      </label>
                      <Input
                        type="number"
                        min={1}
                        max={1000}
                        value={settings.default_batch_size}
                        onChange={(e) =>
                          setSettings({
                            ...settings,
                            default_batch_size: parseInt(e.target.value) || 10,
                          })
                        }
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
                        value={settings.default_concurrency}
                        onChange={(e) =>
                          setSettings({
                            ...settings,
                            default_concurrency: parseInt(e.target.value) || 4,
                          })
                        }
                      />
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* 9. About Tab */}
            {activeTab === "about" && (
              <div className="space-y-4">
                <div className="flex items-center gap-3 border-b border-border pb-4">
                  <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-primary to-primary/70 flex items-center justify-center font-bold text-xl text-primary-foreground shadow-lg">
                    L
                  </div>
                  <div>
                    <h2 className="font-bold text-base text-foreground">Lingo Translate - Game Translation Suite</h2>
                    <p className="text-xs text-muted-foreground">Version {__APP_VERSION__} (Pure Go + React)</p>
                  </div>
                </div>

                <div className="space-y-2 text-xs text-muted-foreground leading-relaxed">
                  <p>
                    Modern, high-performance visual novel and game translation suite.
                    Designed for fast scanning, parallel AI translation pipelines, and zero-dependency packaging.
                  </p>
                  <div className="p-3 bg-card rounded-md border border-border space-y-1">
                    <div className="text-foreground font-semibold">Engine Features:</div>
                    <div>• Supported Engines: RPG Maker (MV/MZ), Ren&apos;Py, Godot, Unity, LibGDX</div>
                    <div>• AI Architecture: Hermes task-based model delegation with automatic fallback</div>
                    <div>• Translation Memory: Instant SQLite match cache with character lore injection</div>
                    <div>• Providers: Official Gemini, OpenAI, UCHS AI Cluster, and custom JSON drivers</div>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>

        <DialogFooter className="p-3 bg-card/60 border-t border-border flex justify-end gap-2">
          <Button variant="secondary" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleSave}
            disabled={saving}
            className="gap-1.5"
          >
            <Save className="w-4 h-4" />
            {saving ? "Saving..." : "Save Settings"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
