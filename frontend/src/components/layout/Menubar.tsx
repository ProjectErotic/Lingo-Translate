import React from "react";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  Button,
} from "@/ui";
import {
  FolderOpen,
  FileCode,
  XSquare,
  Settings,
  Languages,
  Rocket,
  GitMerge,
  Share2,
  HelpCircle,
  LayoutGrid,
  Edit3,
  BookOpen,
  Globe,
  Check,
} from "lucide-react";
import { useI18n } from "@/lib/i18n";

interface MenubarProps {
  hasOpenProject: boolean;
  onOpenGame: () => void;
  onOpenWorkspace: () => void;
  onCloseProject: () => void;
  onOpenSettings: () => void;
  onOpenTranslate: () => void;
  onOpenDeploy: () => void;
  onOpenMerge: () => void;
  onOpenPublish: () => void;
  onOpenAbout: () => void;
  onOpenUserGuide: () => void;
  currentRoute: string;
  onNavigate: (route: string) => void;
}

export const Menubar: React.FC<MenubarProps> = ({
  hasOpenProject,
  onOpenGame,
  onOpenWorkspace,
  onCloseProject,
  onOpenSettings,
  onOpenTranslate,
  onOpenDeploy,
  onOpenMerge,
  onOpenPublish,
  onOpenAbout,
  onOpenUserGuide,
  currentRoute,
  onNavigate,
}) => {
  const { t, locale, setLocale, supportedLocales } = useI18n();

  return (
    <div className="h-8 bg-popover border-b border-border flex items-center px-2 text-xs select-none gap-0.5 z-40">
      {/* Brand logo / tag */}
      <div className="flex items-center gap-1.5 px-2 mr-1 text-primary font-bold tracking-wide">
        <span className="w-2 h-2 rounded-full bg-primary" />
        Lingo
      </div>

      {/* File Menu */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="sm" className="h-6 px-2 text-xs font-normal">
            {t("menubar.file")}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start">
          <DropdownMenuItem onClick={onOpenGame}>
            <FolderOpen className="w-3.5 h-3.5 mr-2 text-primary" />
            {t("menubar.open_game")}
            <DropdownMenuShortcut>Ctrl+O</DropdownMenuShortcut>
          </DropdownMenuItem>
          <DropdownMenuItem onClick={onOpenWorkspace}>
            <FileCode className="w-3.5 h-3.5 mr-2 text-primary" />
            {t("menubar.open_workspace")}
            <DropdownMenuShortcut>Ctrl+Shift+O</DropdownMenuShortcut>
          </DropdownMenuItem>
          {hasOpenProject && (
            <DropdownMenuItem onClick={onCloseProject}>
              <XSquare className="w-3.5 h-3.5 mr-2 text-rose-400" />
              {t("menubar.close_project")}
              <DropdownMenuShortcut>Ctrl+W</DropdownMenuShortcut>
            </DropdownMenuItem>
          )}
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={onOpenSettings}>
            <Settings className="w-3.5 h-3.5 mr-2 text-muted-foreground" />
            {t("menubar.settings")}
            <DropdownMenuShortcut>Ctrl+,</DropdownMenuShortcut>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      {/* Translate Menu */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="sm"
            disabled={!hasOpenProject}
            className="h-6 px-2 text-xs font-normal disabled:opacity-40"
          >
            {t("menubar.translate")}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start">
          <DropdownMenuItem onClick={onOpenTranslate}>
            <Languages className="w-3.5 h-3.5 mr-2 text-primary" />
            {t("menubar.batch_translate")}
            <DropdownMenuShortcut>Ctrl+T</DropdownMenuShortcut>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      {/* Deploy Menu */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="sm"
            disabled={!hasOpenProject}
            className="h-6 px-2 text-xs font-normal disabled:opacity-40"
          >
            {t("menubar.deploy")}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start">
          <DropdownMenuItem onClick={onOpenDeploy}>
            <Rocket className="w-3.5 h-3.5 mr-2 text-amber-400" />
            {t("menubar.deploy_to_game")}
            <DropdownMenuShortcut>Ctrl+D</DropdownMenuShortcut>
          </DropdownMenuItem>
          <DropdownMenuItem onClick={onOpenMerge}>
            <GitMerge className="w-3.5 h-3.5 mr-2 text-emerald-400" />
            {t("menubar.smart_merge")}
            <DropdownMenuShortcut>Ctrl+M</DropdownMenuShortcut>
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={onOpenPublish}>
            <Share2 className="w-3.5 h-3.5 mr-2 text-primary" />
            {t("menubar.publish")}
            <DropdownMenuShortcut>Ctrl+P</DropdownMenuShortcut>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      {/* View Menu */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="sm" className="h-6 px-2 text-xs font-normal">
            {t("menubar.tools")}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start">
          <DropdownMenuItem
            onClick={() => onNavigate("/")}
            className={currentRoute === "/" ? "bg-muted font-semibold" : ""}
          >
            <LayoutGrid className="w-3.5 h-3.5 mr-2 text-primary" />
            {t("menubar.nav_projects")}
          </DropdownMenuItem>
          {hasOpenProject && (
            <DropdownMenuItem
              onClick={() => onNavigate("/editor")}
              className={currentRoute === "/editor" ? "bg-muted font-semibold" : ""}
            >
              <Edit3 className="w-3.5 h-3.5 mr-2 text-emerald-400" />
              {t("menubar.nav_editor")}
            </DropdownMenuItem>
          )}
        </DropdownMenuContent>
      </DropdownMenu>

      {/* Help Menu */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="sm" className="h-6 px-2 text-xs font-normal">
            Help
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start">
          <DropdownMenuItem onClick={onOpenUserGuide}>
            <BookOpen className="w-3.5 h-3.5 mr-2 text-primary" />
            {t("menubar.user_guide")}
            <DropdownMenuShortcut>F1</DropdownMenuShortcut>
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={onOpenAbout}>
            <HelpCircle className="w-3.5 h-3.5 mr-2 text-muted-foreground" />
            {t("menubar.about")}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      {/* Right Quick Actions */}
      <div className="ml-auto flex items-center gap-1">
        {/* Language Switcher */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="sm"
              className="h-6 px-2 text-xs font-normal text-muted-foreground hover:text-foreground gap-1.5"
              title="เปลี่ยนภาษา / Switch Language"
            >
              <Globe className="w-3.5 h-3.5 text-primary" />
              <span>{locale === "th" ? "ภาษาไทย" : "English"}</span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {supportedLocales.map((loc) => (
              <DropdownMenuItem
                key={loc.code}
                onClick={() => setLocale(loc.code)}
                className="flex items-center justify-between gap-4"
              >
                <div className="flex items-center gap-2">
                  <span>{loc.code === "th" ? "🇹🇭" : "🇺🇸"}</span>
                  <span className={locale === loc.code ? "font-semibold text-primary" : ""}>
                    {loc.nativeName} ({loc.name})
                  </span>
                </div>
                {locale === loc.code && <Check className="w-3.5 h-3.5 text-primary" />}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>

        {/* User Guide Button */}
        <Button
          variant="ghost"
          size="sm"
          onClick={onOpenUserGuide}
          className="h-6 px-2 text-xs font-normal text-muted-foreground hover:text-foreground gap-1.5"
          title="User Guide / คู่มือการใช้งาน (F1)"
        >
          <BookOpen className="w-3.5 h-3.5 text-primary" />
          <span className="hidden sm:inline">{t("projects.btn_user_guide")}</span>
        </Button>
      </div>
    </div>
  );
};
