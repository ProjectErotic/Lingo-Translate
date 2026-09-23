import { useEffect, useState, useCallback } from "react";
import { HashRouter, Routes, Route, useNavigate, useLocation } from "react-router-dom";
import {
  ProjectService,
  EntryService,
} from "@bindings/lingo-translate/cmd/lingo-desktop";
import type { Project } from "@bindings/lingo-translate/pkg/model";
import type { WorkspaceStats } from "@bindings/lingo-translate/pkg/storage";
import { Menubar } from "@/components/layout/Menubar";
import { Statusbar } from "@/components/layout/Statusbar";
import { ProjectsPage } from "@/pages/ProjectsPage";
import { EditorPage } from "@/pages/EditorPage";
import { OpenGameDialog } from "@/components/dialogs/OpenGameDialog";
import { TranslateDialog } from "@/components/dialogs/TranslateDialog";
import { DeployDialog } from "@/components/dialogs/DeployDialog";
import { UpdateVersionDialog } from "@/components/dialogs/UpdateVersionDialog";
import { PublishDialog } from "@/components/dialogs/PublishDialog";
import { SettingsDialog } from "@/components/dialogs/SettingsDialog";
import { AboutDialog } from "@/components/dialogs/AboutDialog";
import { Toaster, toast } from "sonner";
import { CommandProvider, useCommands } from "@/lib/commands";

function AppContent() {
  const navigate = useNavigate();
  const location = useLocation();

  const [currentProject, setCurrentProject] = useState<Project | null>(null);
  const [stats, setStats] = useState<WorkspaceStats | null>(null);
  const [selectedFile, setSelectedFile] = useState<string>("all");

  // Dialog open states
  const [openGameDialog, setOpenGameDialog] = useState(false);
  const [translateDialog, setTranslateDialog] = useState(false);
  const [deployDialog, setDeployDialog] = useState(false);
  const [mergeDialog, setMergeDialog] = useState(false);
  const [publishDialog, setPublishDialog] = useState(false);
  const [settingsDialog, setSettingsDialog] = useState(false);
  const [aboutDialog, setAboutDialog] = useState(false);

  // Refresh current project and stats
  const refreshCurrentProject = useCallback(async () => {
    try {
      const proj = await ProjectService.Current();
      setCurrentProject(proj);
      if (proj) {
        const s = await EntryService.Stats();
        setStats(s);
      } else {
        setStats(null);
      }
    } catch (err) {
      console.warn("Could not fetch current project:", err);
    }
  }, []);

  useEffect(() => {
    refreshCurrentProject();
  }, [refreshCurrentProject]);

  // Open a workspace from file path
  const handleOpenWorkspacePath = async (wsPath: string) => {
    try {
      const proj = await ProjectService.OpenWorkspace(wsPath);
      setCurrentProject(proj);
      const s = await EntryService.Stats();
      setStats(s);
      setSelectedFile("all");
      toast.success(`Opened project: ${proj?.name || wsPath}`);
      navigate("/editor");
    } catch (err: any) {
      toast.error(`Failed to open project: ${err?.message || err}`);
    }
  };

  // Browse dialog for opening .nst file
  const handleBrowseWorkspace = async () => {
    try {
      const selected = await ProjectService.PickWorkspace(false);
      if (selected) {
        await handleOpenWorkspacePath(selected);
      }
    } catch (err: any) {
      toast.error(`Picker error: ${err?.message || err}`);
    }
  };

  // Close active project
  const handleCloseProject = async () => {
    try {
      await ProjectService.Close();
      setCurrentProject(null);
      setStats(null);
      toast.info("Project closed");
      navigate("/");
    } catch (err: any) {
      toast.error(`Failed to close project: ${err?.message || err}`);
    }
  };

  // Register global shortcuts across the application
  useCommands(
    [
      {
        id: "file.open-game",
        title: "Open Game Folder",
        category: "File",
        keybinding: "Ctrl+O",
        run: () => setOpenGameDialog(true),
      },
      {
        id: "file.open-workspace",
        title: "Open Workspace File (.nst)",
        category: "File",
        keybinding: "Ctrl+Shift+O",
        run: handleBrowseWorkspace,
      },
      {
        id: "file.close-project",
        title: "Close Project",
        category: "File",
        keybinding: "Ctrl+W",
        when: () => currentProject !== null,
        preventInInput: true,
        run: handleCloseProject,
      },
      {
        id: "app.settings",
        title: "Settings",
        category: "Preferences",
        keybinding: "Ctrl+,",
        run: () => setSettingsDialog(true),
      },
      {
        id: "translate.open",
        title: "AI Translate",
        category: "Translation",
        keybinding: "Ctrl+T",
        when: () => currentProject !== null,
        run: () => setTranslateDialog(true),
      },
      {
        id: "deploy.open",
        title: "Deploy Mod",
        category: "Export",
        keybinding: "Ctrl+D",
        when: () => currentProject !== null,
        run: () => setDeployDialog(true),
      },
      {
        id: "merge.open",
        title: "Update Version / Merge",
        category: "Project",
        keybinding: "Ctrl+M",
        run: () => setMergeDialog(true),
      },
      {
        id: "publish.open",
        title: "Publish to Chanomhub",
        category: "Publish",
        keybinding: "Ctrl+P",
        when: () => currentProject !== null,
        run: () => setPublishDialog(true),
      },
      {
        id: "app.about",
        title: "About Lingo Translate",
        category: "Help",
        keybinding: "F1",
        run: () => setAboutDialog(true),
      },
    ],
    [currentProject]
  );

  return (
    <div className="h-screen w-screen flex flex-col overflow-hidden bg-background text-foreground">
      <Toaster position="top-right" theme="dark" richColors />

      {/* Top Application Menubar */}
      <Menubar
        hasOpenProject={currentProject !== null}
        onOpenGame={() => setOpenGameDialog(true)}
        onOpenWorkspace={handleBrowseWorkspace}
        onCloseProject={handleCloseProject}
        onOpenSettings={() => setSettingsDialog(true)}
        onOpenTranslate={() => setTranslateDialog(true)}
        onOpenDeploy={() => setDeployDialog(true)}
        onOpenMerge={() => setMergeDialog(true)}
        onOpenPublish={() => setPublishDialog(true)}
        onOpenAbout={() => setAboutDialog(true)}
        currentRoute={location.pathname}
        onNavigate={(route) => navigate(route)}
      />

      {/* Main Screen Views */}
      <div className="flex-1 flex overflow-hidden">
        <Routes>
          <Route
            path="/"
            element={
              <ProjectsPage
                onOpenProject={handleOpenWorkspacePath}
                onExtractNew={() => setOpenGameDialog(true)}
                onBrowseWorkspace={handleBrowseWorkspace}
              />
            }
          />
          <Route
            path="/editor"
            element={
              <EditorPage
                onOpenTranslate={() => setTranslateDialog(true)}
                onOpenDeploy={() => setDeployDialog(true)}
                selectedFile={selectedFile}
                onSelectFile={setSelectedFile}
                onStatsUpdated={refreshCurrentProject}
              />
            }
          />
        </Routes>
      </div>

      {/* Bottom Statusbar */}
      <Statusbar project={currentProject} stats={stats} />

      {/* Dialog Modals */}
      <OpenGameDialog
        open={openGameDialog}
        onOpenChange={setOpenGameDialog}
        onSuccess={() => {
          refreshCurrentProject();
          navigate("/editor");
        }}
      />

      <TranslateDialog
        open={translateDialog}
        onOpenChange={setTranslateDialog}
        currentFile={selectedFile !== "all" ? selectedFile : undefined}
        sourceLang={currentProject?.source_lang || "Japanese"}
        targetLang={currentProject?.target_lang || "Thai"}
        onFinished={refreshCurrentProject}
      />

      <DeployDialog
        open={deployDialog}
        onOpenChange={setDeployDialog}
        engine={currentProject?.engine || ""}
        defaultGamePath={currentProject?.source_path || ""}
        defaultTargetLang={currentProject?.target_lang || "Thai"}
      />

      <UpdateVersionDialog
        open={mergeDialog}
        onOpenChange={setMergeDialog}
        onSuccess={refreshCurrentProject}
      />

      <PublishDialog
        open={publishDialog}
        onOpenChange={setPublishDialog}
        gameDir={currentProject?.source_path || ""}
      />

      <SettingsDialog
        open={settingsDialog}
        onOpenChange={setSettingsDialog}
      />

      <AboutDialog
        open={aboutDialog}
        onOpenChange={setAboutDialog}
      />
    </div>
  );
}

export function App() {
  return (
    <HashRouter>
      <CommandProvider>
        <AppContent />
      </CommandProvider>
    </HashRouter>
  );
}

export default App;
