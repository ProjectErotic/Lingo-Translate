import React, { useEffect, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  Button,
  Input,
} from "@/ui";
import { SettingsService } from "@bindings/lingo-translate/cmd/lingo-desktop";
import type { Settings } from "@bindings/lingo-translate/cmd/lingo-desktop";
import { toast } from "sonner";
import {
  Save,
  Settings as SettingsIcon,
  Zap,
  Bot,
  Key,
  Puzzle,
  Sliders,
  Info,
  Eye,
  EyeOff,
  RefreshCw,
  Layers,
  ExternalLink,
  Sparkles,
  Cpu,
} from "lucide-react";
import { fetchProviders, BUILTIN_PROVIDERS, type ProviderInfo } from "@/lib/providers";

interface SettingsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

type TabType = "providers" | "models" | "system_one" | "keys" | "plugins" | "translation" | "about";

export const SettingsDialog: React.FC<SettingsDialogProps> = ({
  open,
  onOpenChange,
}) => {
  const [activeTab, setActiveTab] = useState<TabType>("providers");
  const [showKeys, setShowKeys] = useState<{ [key: string]: boolean }>({});
  const [providers, setProviders] = useState<ProviderInfo[]>(BUILTIN_PROVIDERS);
  const [isRefreshingPlugins, setIsRefreshingPlugins] = useState(false);

  const [settings, setSettings] = useState<Settings>({
    default_provider: "mock",
    default_model: "gemini-2.5-flash",
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
    tasks: {
      primary_translation: { provider: "gemini", model: "gemini-2.5-flash" },
      fast_translation: { provider: "google", model: "google-translate" },
      auto_route_short_text: false,
      max_short_length: 60,
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

  const loadProvidersData = async () => {
    try {
      const list = await fetchProviders();
      setProviders(list);
    } catch (err) {
      console.error("Failed to load providers:", err);
    }
  };

  useEffect(() => {
    if (open) {
      loadProvidersData();
      SettingsService.GetSettings()
        .then((s) => {
          if (s) {
            setSettings({
              ...s,
              uchs_api_key: s.uchs_api_key || "",
              plugin_keys: s.plugin_keys || {},
              plugin_base_urls: s.plugin_base_urls || {},
              tasks: s.tasks || {
                primary_translation: { provider: s.default_provider || "gemini", model: s.default_model || "gemini-2.5-flash" },
                fast_translation: { provider: "google", model: "google-translate" },
                auto_route_short_text: false,
                max_short_length: 60,
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
      let key = "";
      try {
        key = await SettingsService.RequestUchsKey();
      } catch {
        const res = await fetch("https://api.chanomhub.com/api/uchs/keys/generate", {
          method: "POST",
          headers: {
            "Authorization": `Bearer ${settings.chanomhub_token}`,
            "Content-Type": "application/json",
          },
        });
        if (!res.ok) throw new Error(`Server returned HTTP ${res.status}`);
        const data = await res.json();
        key = data.key;
      }
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
      try {
        await SettingsService.OpenUchsPortal();
      } catch {
        const res = await fetch("https://api.chanomhub.com/api/uchs/sso-url", {
          method: "POST",
          headers: {
            "Authorization": `Bearer ${settings.chanomhub_token}`,
            "Content-Type": "application/json",
          },
        });
        if (!res.ok) throw new Error(`Server returned HTTP ${res.status}`);
        const data = await res.json();
        if (data.redirectUrl) {
          window.open(data.redirectUrl, "_blank", "noopener,noreferrer");
        }
      }
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

  const handlePluginBaseURLChange = (providerName: string, value: string) => {
    setSettings((prev) => ({
      ...prev,
      plugin_base_urls: {
        ...(prev.plugin_base_urls || {}),
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

  const handleSave = async () => {
    try {
      setSaving(true);
      await SettingsService.SaveSettings(settings);
      toast.success("Settings saved successfully!");
      onOpenChange(false);
    } catch (err: any) {
      toast.error(`Failed to save settings: ${err?.message || err}`);
    } finally {
      setSaving(false);
    }
  };

  const activeProviderInfo = providers.find((p) => p.name === settings.default_provider);
  const customPlugins = providers.filter((p) => p.is_custom);

  const navItems = [
    { id: "providers" as TabType, label: "Providers", icon: Zap },
    { id: "models" as TabType, label: "Task Routing", icon: Bot },
    { id: "system_one" as TabType, label: "System One (Jev)", icon: Cpu },
    { id: "keys" as TabType, label: "API Keys", icon: Key },
    {
      id: "plugins" as TabType,
      label: `Plugins (${customPlugins.length})`,
      icon: Puzzle,
    },
    { id: "translation" as TabType, label: "Translation", icon: Sliders },
    { id: "about" as TabType, label: "About", icon: Info },
  ];

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl h-[560px] p-0 flex flex-col overflow-hidden bg-background border-border">
        <DialogHeader className="p-4 pb-3 border-b border-border">
          <DialogTitle className="flex items-center gap-2 text-base text-foreground">
            <SettingsIcon className="w-5 h-5 text-primary" />
            Preferences & Settings
          </DialogTitle>
        </DialogHeader>

        <div className="flex flex-1 overflow-hidden">
          {/* Left Sidebar */}
          <div className="w-48 bg-card/60 border-r border-border p-2 space-y-1">
            {navItems.map((item) => {
              const Icon = item.icon;
              const isActive = activeTab === item.id;
              return (
                // @ui-allow-native
                <button
                  key={item.id}
                  onClick={() => setActiveTab(item.id)}
                  className={`w-full flex items-center gap-3 px-3 py-2 rounded-md text-xs font-medium transition-colors text-left ${
                    isActive
                      ? "bg-secondary text-foreground font-semibold border-l-2 border-primary"
                      : "text-muted-foreground hover:text-foreground hover:bg-muted"
                  }`}
                >
                  <Icon className={`w-4 h-4 ${isActive ? "text-primary" : "text-muted-foreground"}`} />
                  {item.label}
                </button>
              );
            })}
          </div>

          {/* Right Content Area */}
          <div className="flex-1 p-5 overflow-y-auto bg-background text-sm text-foreground">
            {/* 1. Providers Tab */}
            {activeTab === "providers" && (
              <div className="space-y-4">
                <div>
                  <h3 className="text-sm font-semibold text-foreground">Translation Providers</h3>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    Select default translation module and custom endpoint routes.
                  </p>
                </div>

                <div className="space-y-3 pt-2">
                  <div>
                    <label className="block text-xs font-semibold text-muted-foreground mb-1">
                      Active Default Provider
                    </label>
                    <select
                      value={settings.default_provider}
                      onChange={(e) => {
                        const val = e.target.value;
                        const found = providers.find((p) => p.name === val);
                        setSettings({
                          ...settings,
                          default_provider: val,
                          default_model: found?.default_model || (found?.available_models?.[0] || settings.default_model),
                        });
                      }}
                      className="w-full h-9 rounded-md border border-input bg-card px-3 text-sm text-foreground focus:outline-none focus:border-primary"
                    >
                      {providers.map((p) => (
                        <option key={p.name} value={p.name}>
                          {p.display_name} {p.is_custom ? "[External Plugin]" : "[Built-in]"}
                        </option>
                      ))}
                    </select>
                  </div>

                  {/* Provider Info Card */}
                  {activeProviderInfo && (
                    <div className="p-3 bg-card border border-border rounded-md space-y-1.5 text-xs">
                      <div className="flex justify-between items-center">
                        <span className="font-semibold text-foreground">
                          {activeProviderInfo.display_name}
                        </span>
                        <span
                          className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                            activeProviderInfo.is_custom
                              ? "bg-amber-950/60 text-amber-400"
                              : "bg-primary/20 text-primary"
                          }`}
                        >
                          {activeProviderInfo.is_custom ? "PLUGIN DRIVER" : "BUILT-IN"}
                        </span>
                      </div>
                      <p className="text-muted-foreground text-[11px]">
                        {activeProviderInfo.description || "Declarative translation driver."}
                      </p>
                      {activeProviderInfo.base_url && (
                        <div className="text-[11px] text-muted-foreground flex items-center gap-1 font-mono">
                          <span>Endpoint:</span>
                          <code className="text-foreground">{activeProviderInfo.base_url}</code>
                        </div>
                      )}
                    </div>
                  )}

                  {/* Custom Base URL (if openai or custom plugin) */}
                  {(activeProviderInfo?.is_custom || settings.default_provider === "openai") && (
                    <div>
                      <label className="block text-xs font-semibold text-muted-foreground mb-1">
                        Endpoint Base URL Override (Optional)
                      </label>
                      <Input
                        value={
                          activeProviderInfo?.is_custom
                            ? settings.plugin_base_urls?.[settings.default_provider] || ""
                            : settings.openai_base_url
                        }
                        onChange={(e) => {
                          if (activeProviderInfo?.is_custom) {
                            handlePluginBaseURLChange(settings.default_provider, e.target.value);
                          } else {
                            setSettings({ ...settings, openai_base_url: e.target.value });
                          }
                        }}
                        placeholder={activeProviderInfo?.base_url || "https://api.openai.com/v1"}
                      />
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* 2. Task Routing Tab (Hermes-Style Delegation) */}
            {activeTab === "models" && (
              <div className="space-y-4">
                <div>
                  <div className="flex items-center gap-2">
                    <h3 className="text-sm font-semibold text-foreground">Task-Based Model Routing</h3>
                    <span className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-primary/20 text-primary">
                      HERMES ARCHITECTURE
                    </span>
                  </div>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    Delegate localization workload to specialized models based on complexity and cost.
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
                        High-context reasoning model for dialogue, character speech, and emotional narrative.
                      </p>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-3 pt-1">
                    <div>
                      <label className="block text-[11px] font-medium text-muted-foreground mb-1">
                        Provider
                      </label>
                      <select
                        value={settings.tasks?.primary_translation?.provider || settings.default_provider}
                        onChange={(e) => {
                          const val = e.target.value;
                          const found = providers.find((p) => p.name === val);
                          setSettings({
                            ...settings,
                            default_provider: val,
                            tasks: {
                              ...settings.tasks!,
                              primary_translation: {
                                ...settings.tasks!.primary_translation,
                                provider: val,
                                model: found?.default_model || (found?.available_models?.[0] || settings.tasks!.primary_translation.model),
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
                        Model String
                      </label>
                      <Input
                        className="h-8 text-xs font-mono"
                        value={settings.tasks?.primary_translation?.model || settings.default_model}
                        onChange={(e) =>
                          setSettings({
                            ...settings,
                            default_model: e.target.value,
                            tasks: {
                              ...settings.tasks!,
                              primary_translation: {
                                ...settings.tasks!.primary_translation,
                                model: e.target.value,
                              },
                            },
                          })
                        }
                        placeholder="e.g. gemini-2.5-pro, claude-3-7-sonnet"
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
                      <select
                        value={settings.tasks?.fast_translation?.provider || "google"}
                        onChange={(e) => {
                          const val = e.target.value;
                          const found = providers.find((p) => p.name === val);
                          setSettings({
                            ...settings,
                            tasks: {
                              ...settings.tasks!,
                              fast_translation: {
                                ...settings.tasks!.fast_translation,
                                provider: val,
                                model: found?.default_model || (found?.available_models?.[0] || "google-translate"),
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
                        Model String
                      </label>
                      <Input
                        className="h-8 text-xs font-mono"
                        value={settings.tasks?.fast_translation?.model || "google-translate"}
                        onChange={(e) =>
                          setSettings({
                            ...settings,
                            tasks: {
                              ...settings.tasks!,
                              fast_translation: {
                                ...settings.tasks!.fast_translation,
                                model: e.target.value,
                              },
                            },
                          })
                        }
                        placeholder="e.g. google-translate, gemini-2.5-flash"
                      />
                    </div>
                  </div>

                  <div className="flex items-center justify-between pt-2 border-t border-border/50 text-xs">
                    <label className="flex items-center gap-2 cursor-pointer select-none text-muted-foreground hover:text-foreground">
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

                {/* Legacy Fallback Default */}
                <div className="pt-1 text-xs text-muted-foreground flex items-center justify-between">
                  <span>Fallback Model: <code className="text-foreground font-mono">{settings.default_model}</code></span>
                  <span className="text-[11px]">Primary Provider: <code className="text-foreground font-mono">{settings.default_provider}</code></span>
                </div>
              </div>
            )}

            {/* 3. System One (Jev) Tab */}
            {activeTab === "system_one" && (
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <div className="flex items-center gap-2">
                      <h3 className="text-sm font-semibold text-foreground">System One Decision Engine</h3>
                      <span
                        className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                          settings.system_one?.enabled
                            ? "bg-emerald-950/60 text-emerald-400 border border-emerald-500/30"
                            : "bg-muted text-muted-foreground"
                        }`}
                      >
                        {settings.system_one?.enabled ? "ACTIVE (Jev)" : "DISABLED"}
                      </span>
                    </div>
                    <p className="text-xs text-muted-foreground mt-0.5">
                      Ultra-low latency non-autoregressive decision layer (TypeSafe AI Jev) for code filtering & draft acceptance.
                    </p>
                  </div>

                  <Button
                    type="button"
                    variant={settings.system_one?.enabled ? "default" : "secondary"}
                    size="sm"
                    className="h-8 gap-1.5 text-xs"
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
                    <Cpu className="w-3.5 h-3.5" />
                    {settings.system_one?.enabled ? "Enabled" : "Enable Jev"}
                  </Button>
                </div>

                <div className="p-3 bg-card/70 border border-border rounded-lg space-y-3">
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="block text-xs font-semibold text-muted-foreground mb-1">
                        System One Provider
                      </label>
                      <select
                        disabled={!settings.system_one?.enabled}
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
                        className="w-full h-8 rounded-md border border-input bg-card px-2.5 text-xs text-foreground focus:outline-none focus:border-primary disabled:opacity-50"
                      >
                        <option value="typesafe_jev">TypeSafe AI (Jev Decision Engine)</option>
                        <option value="heuristic">Built-in Deterministic Heuristic</option>
                      </select>
                    </div>

                    <div>
                      <label className="block text-xs font-semibold text-muted-foreground mb-1">
                        Confidence Threshold
                      </label>
                      <Input
                        type="number"
                        step="0.05"
                        min="0.5"
                        max="1.0"
                        disabled={!settings.system_one?.enabled}
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
                        className="h-8 text-xs font-mono disabled:opacity-50"
                      />
                    </div>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-muted-foreground mb-1">
                      TypeSafe AI API Key
                    </label>
                    <div className="flex gap-2">
                      <Input
                        type={showKeys["jev"] ? "text" : "password"}
                        disabled={!settings.system_one?.enabled}
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
                        className="h-8 text-xs font-mono disabled:opacity-50"
                      />
                      <Button
                        type="button"
                        variant="secondary"
                        size="icon"
                        disabled={!settings.system_one?.enabled}
                        className="h-8 w-8 shrink-0"
                        onClick={() => toggleShowKey("jev")}
                      >
                        {showKeys["jev"] ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                      </Button>
                    </div>
                  </div>
                </div>

                {/* Delegations / Capabilities */}
                <div className="p-3 bg-card/70 border border-border rounded-lg space-y-2.5">
                  <div className="text-xs font-semibold text-foreground">
                    Active Decision Delegations
                  </div>
                  <p className="text-[11px] text-muted-foreground">
                    Select which tasks should be routed to System One instead of using traditional heuristics or expensive frontier LLMs:
                  </p>

                  <div className="space-y-2 pt-1 text-xs">
                    <label className="flex items-start gap-2.5 cursor-pointer select-none">
                      <input
                        type="checkbox"
                        disabled={!settings.system_one?.enabled}
                        checked={settings.system_one?.features?.filter_ambiguous_code ?? true}
                        onChange={(e) =>
                          setSettings({
                            ...settings,
                            system_one: {
                              ...settings.system_one!,
                              features: {
                                ...settings.system_one!.features,
                                filter_ambiguous_code: e.target.checked,
                              },
                            },
                          })
                        }
                        className="mt-0.5 rounded border-border text-primary focus:ring-primary disabled:opacity-50"
                      />
                      <div>
                        <span className="font-medium text-foreground">Ambiguous Code Gatekeeper</span>
                        <p className="text-[11px] text-muted-foreground">
                          Call Jev only when heuristic regex is uncertain about engine scripts vs player-facing text.
                        </p>
                      </div>
                    </label>

                    <label className="flex items-start gap-2.5 cursor-pointer select-none">
                      <input
                        type="checkbox"
                        disabled={!settings.system_one?.enabled}
                        checked={settings.system_one?.features?.accept_ui_drafts ?? true}
                        onChange={(e) =>
                          setSettings({
                            ...settings,
                            system_one: {
                              ...settings.system_one!,
                              features: {
                                ...settings.system_one!.features,
                                accept_ui_drafts: e.target.checked,
                              },
                            },
                          })
                        }
                        className="mt-0.5 rounded border-border text-primary focus:ring-primary disabled:opacity-50"
                      />
                      <div>
                        <span className="font-medium text-foreground">Speculative Draft Acceptor (Cost Saver)</span>
                        <p className="text-[11px] text-muted-foreground">
                          Instantly approve fluent MT drafts for short UI & menu strings without calling Frontier LLMs.
                        </p>
                      </div>
                    </label>

                    <label className="flex items-start gap-2.5 cursor-pointer select-none">
                      <input
                        type="checkbox"
                        disabled={!settings.system_one?.enabled}
                        checked={settings.system_one?.features?.verify_qa ?? false}
                        onChange={(e) =>
                          setSettings({
                            ...settings,
                            system_one: {
                              ...settings.system_one!,
                              features: {
                                ...settings.system_one!.features,
                                verify_qa: e.target.checked,
                              },
                            },
                          })
                        }
                        className="mt-0.5 rounded border-border text-primary focus:ring-primary disabled:opacity-50"
                      />
                      <div>
                        <span className="font-medium text-foreground">Automated QA & Refusal Sentinel</span>
                        <p className="text-[11px] text-muted-foreground">
                          Score translation quality and auto-flag safety refusals or hallucinations before database commit.
                        </p>
                      </div>
                    </label>
                  </div>
                </div>
              </div>
            )}

            {/* 3. API Keys Tab */}
            {activeTab === "keys" && (
              <div className="space-y-4">
                <div>
                  <h3 className="text-sm font-semibold text-foreground">API Keys & Authentication</h3>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    Credentials are saved securely in your local user config.
                  </p>
                </div>

                <div className="space-y-4 pt-2">
                  {/* Built-in Providers */}
                  <div className="space-y-3">
                    <div className="text-xs font-semibold text-primary uppercase tracking-wider">
                      Core Drivers
                    </div>

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
                          className="h-8 w-8"
                          onClick={() => toggleShowKey("gemini")}
                        >
                          {showKeys["gemini"] ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                        </Button>
                      </div>
                    </div>

                    <div>
                      <label className="block text-xs font-semibold text-muted-foreground mb-1">
                        OpenAI API Key (or Local LLM)
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
                          className="h-8 w-8"
                          onClick={() => toggleShowKey("openai")}
                        >
                          {showKeys["openai"] ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                        </Button>
                      </div>
                    </div>

                    <div>
                      <label className="block text-xs font-semibold text-muted-foreground mb-1">
                        Google Translate API Key (Optional)
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
                          className="h-8 w-8"
                          onClick={() => toggleShowKey("google")}
                        >
                          {showKeys["google"] ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                        </Button>
                      </div>
                    </div>

                    <div>
                      <label className="block text-xs font-semibold text-muted-foreground mb-1">
                        Chanomhub Token (Publishing)
                      </label>
                      <div className="flex gap-2">
                        <Input
                          type={showKeys["chanomhub"] ? "text" : "password"}
                          value={settings.chanomhub_token}
                          onChange={(e) =>
                            setSettings({ ...settings, chanomhub_token: e.target.value })
                          }
                          placeholder="JWT Token"
                        />
                        <Button
                          type="button"
                          variant="secondary"
                          size="icon"
                          className="h-8 w-8"
                          onClick={() => toggleShowKey("chanomhub")}
                        >
                          {showKeys["chanomhub"] ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                        </Button>
                      </div>
                    </div>

                    {/* UCHS AI Infrastructure Section */}
                    <div className="p-3 bg-emerald-950/20 border border-emerald-500/30 rounded-lg space-y-2.5">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <Cpu className="w-4 h-4 text-emerald-400" />
                          <label className="block text-xs font-semibold text-emerald-300">
                            UCHS AI Gateway (LiteLLM Cluster)
                          </label>
                        </div>
                        <span className="text-[10px] text-emerald-400/80 font-mono">
                          ilms.uchs-th.com
                        </span>
                      </div>

                      <p className="text-[11px] text-muted-foreground leading-relaxed">
                        เข้าถึงโมเดล DeepSeek V4.1 Flash, DeepSeek Pro และ Translation Engine ด้วย Virtual Key ที่ออกผ่านบัญชี Chanomhub
                      </p>

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

                      <div className="flex flex-wrap gap-2 pt-1">
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          disabled={requestingUchsKey || !settings.chanomhub_token}
                          onClick={handleRequestUchsKey}
                          className="text-xs h-7 gap-1 border-emerald-500/40 text-emerald-400 hover:bg-emerald-500/10"
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
                          className="text-xs h-7 gap-1 text-zinc-300 hover:text-white"
                        >
                          <ExternalLink className="w-3 h-3 text-emerald-400" />
                          {openingUchsPortal ? "กำลังเปิด..." : "เปิดหน้าบัญชี UCHS (Portal)"}
                        </Button>
                      </div>
                    </div>

                    {/* TypeSafe AI (Jev) Section */}
                    <div className="p-3 bg-card border border-border rounded-lg space-y-2">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <Cpu className="w-4 h-4 text-primary" />
                          <label className="block text-xs font-semibold text-foreground">
                            TypeSafe AI API Key (System One - Jev)
                          </label>
                        </div>
                        <span className="text-[10px] text-muted-foreground font-mono">
                          typesafe.ai
                        </span>
                      </div>
                      <p className="text-[11px] text-muted-foreground leading-relaxed">
                        Used for fast non-autoregressive decision tasks (ambiguous code filtering and speculative draft acceptance).
                      </p>
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
                          className="font-mono text-xs"
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
                              plugin: {plugin.name}
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
                              className="h-8 w-8"
                              onClick={() => toggleShowKey(plugin.name)}
                            >
                              {showKeys[plugin.name] ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                            </Button>
                          </div>
                          {plugin.base_url && (
                            <div className="text-[11px] text-muted-foreground flex items-center gap-1">
                              <span>Default Endpoint:</span>
                              <code className="text-foreground bg-muted px-1 py-0.5 rounded">{plugin.base_url}</code>
                            </div>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* 4. Plugins Tab */}
            {activeTab === "plugins" && (
              <div className="space-y-4">
                <div className="flex justify-between items-start">
                  <div>
                    <h3 className="text-sm font-semibold text-foreground">External Provider Plugins</h3>
                    <p className="text-xs text-muted-foreground mt-0.5">
                      Auto-discovered declarative plugins from <code>providers/</code> and <code>~/.config/lingo/providers/</code>.
                    </p>
                  </div>
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={handleRefreshPlugins}
                    disabled={isRefreshingPlugins}
                    className="h-7 text-xs gap-1.5"
                  >
                    <RefreshCw className={`w-3.5 h-3.5 ${isRefreshingPlugins ? "animate-spin" : ""}`} />
                    Refresh
                  </Button>
                </div>

                <div className="space-y-2 pt-1">
                  {customPlugins.length === 0 ? (
                    <div className="p-4 bg-card border border-border rounded-md text-xs text-muted-foreground text-center">
                      No external JSON plugins currently detected.
                    </div>
                  ) : (
                    customPlugins.map((cp) => (
                      <div key={cp.name} className="p-3 bg-card border border-border rounded-md space-y-1.5">
                        <div className="flex justify-between items-center">
                          <div>
                            <span className="font-semibold text-xs text-foreground">{cp.display_name}</span>
                            <span className="ml-2 text-[10px] text-muted-foreground font-mono">({cp.name})</span>
                          </div>
                          <span className="px-2 py-0.5 text-[10px] font-bold bg-amber-950/60 text-amber-400 rounded">
                            REGISTERED
                          </span>
                        </div>
                        <p className="text-[11px] text-muted-foreground">
                          {cp.description || "Declarative external provider"}
                        </p>
                        <div className="flex flex-wrap gap-2 text-[11px] text-muted-foreground pt-1">
                          {cp.base_url && (
                            <div>Endpoint: <code className="text-foreground">{cp.base_url}</code></div>
                          )}
                          {cp.default_model && (
                            <div>Default: <code className="text-primary font-mono">{cp.default_model}</code></div>
                          )}
                        </div>
                        {cp.available_models && cp.available_models.length > 0 && (
                          <div className="flex flex-wrap gap-1 pt-1">
                            {cp.available_models.map((m) => (
                              <span key={m} className="px-1.5 py-0.5 rounded text-[10px] bg-background text-muted-foreground border border-border">
                                {m}
                              </span>
                            ))}
                          </div>
                        )}
                      </div>
                    ))
                  )}
                </div>

                <div className="text-xs text-muted-foreground bg-card/60 p-3 rounded border border-border mt-3">
                  💡 Drop any new <code>.json</code> provider definition in <code>providers/</code> or <code>~/.config/lingo/providers/</code> to expand translation models dynamically without rebuilding the app.
                </div>
              </div>
            )}

            {/* 5. Translation Tab */}
            {activeTab === "translation" && (
              <div className="space-y-4">
                <div>
                  <h3 className="text-sm font-semibold text-foreground">Translation Pipeline Defaults</h3>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    Configure default languages, batch sizing, and concurrency.
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

            {/* 6. About Tab */}
            {activeTab === "about" && (
              <div className="space-y-4">
                <div className="flex items-center gap-3 border-b border-border pb-4">
                  <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-primary to-primary/70 flex items-center justify-center font-bold text-xl text-primary-foreground shadow-lg">
                    NST
                  </div>
                  <div>
                    <h2 className="font-bold text-base text-foreground">Lingo Translate - Game Translation Suite</h2>
                    <p className="text-xs text-muted-foreground">Version {__APP_VERSION__} (Go Edition)</p>
                  </div>
                </div>

                <div className="space-y-2 text-xs text-muted-foreground leading-relaxed">
                  <p>
                    Modern, high-performance visual novel and game translation suite.
                    Designed for fast scanning, parallel AI translation pipelines, and zero-dependency packaging.
                  </p>
                  <div className="p-3 bg-card rounded-md border border-border space-y-1">
                    <div className="text-foreground font-semibold">Engine Features:</div>
                    <div>• Supported Engines: RPG Maker (MV/MZ), Tyrano, Wolf RPG, BGI / Ethornell, Ren&apos;Py</div>
                    <div>• AI Architecture: Parallel chunk batching with context preservation & glossary injection</div>
                    <div>• Plugin System: External JSON provider drivers with OpenAI-compatible routing</div>
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
