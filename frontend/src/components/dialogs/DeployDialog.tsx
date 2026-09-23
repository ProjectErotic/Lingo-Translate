import React, { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  Button,
  Input,
  Badge,
} from "@/ui";
import {
  DeployService,
  ProjectService,
} from "@bindings/lingo-translate/cmd/lingo-desktop";
import { toast } from "sonner";
import { Rocket, FolderOpen, Loader2 } from "lucide-react";

interface DeployDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  engine: string;
  defaultGamePath: string;
  defaultTargetLang: string;
}

export const DeployDialog: React.FC<DeployDialogProps> = ({
  open,
  onOpenChange,
  engine,
  defaultGamePath,
  defaultTargetLang,
}) => {
  const isRPGM = engine === "rpgm" || engine === "rpgm-mv" || engine === "rpgm-mz";
  const [mode, setMode] = useState<"layer" | "copy">(isRPGM ? "layer" : "copy");
  const gamePath = defaultGamePath;
  const [destPath, setDestPath] = useState(`${defaultGamePath}_Translated`);
  const [langName, setLangName] = useState(defaultTargetLang || "Thai");
  const [deploying, setDeploying] = useState(false);

  const handlePickDestDir = async () => {
    try {
      const selected = await ProjectService.PickDirectory("Select Destination Folder");
      if (selected) {
        setDestPath(selected);
      }
    } catch (err: any) {
      toast.error(`Dialog error: ${err?.message || err}`);
    }
  };

  const handleDeploy = async () => {
    try {
      setDeploying(true);

      if (mode === "layer") {
        if (!isRPGM) {
          toast.error("Non-destructive JS layer is only supported for RPG Maker MV/MZ");
          return;
        }
        await DeployService.DeployLayer(gamePath, langName);
        toast.success(
          "Translation layer deployed! Original data files were preserved untouched."
        );
      } else {
        if (!destPath) {
          toast.error("Destination folder is required for copy mode");
          return;
        }
        await DeployService.ExportCopy(gamePath, destPath);
        toast.success(`Export copy completed! Patched game written to: ${destPath}`);
      }

      onOpenChange(false);
    } catch (err: any) {
      toast.error(`Deployment failed: ${err?.message || err}`);
    } finally {
      setDeploying(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-base">
            <Rocket className="w-5 h-5 text-primary" />
            Deploy Translations to Game
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2 text-sm">
          {/* Deployment Mode Selection */}
          <div className="space-y-2">
            <label className="block text-xs font-semibold text-muted-foreground">
              Deployment Method
            </label>

            {/* Non-destructive Layer (RPGM Only) */}
            <div
              onClick={() => isRPGM && setMode("layer")}
              className={`p-3 rounded-md border transition-all cursor-pointer ${
                mode === "layer"
                  ? "border-primary bg-primary/10"
                  : "border-border bg-card hover:bg-muted"
              } ${!isRPGM ? "opacity-40 cursor-not-allowed" : ""}`}
            >
              <div className="flex items-center justify-between mb-1">
                <span className="font-semibold text-xs text-foreground">
                  Non-Destructive JS Plugin Layer (Drop-in)
                </span>
                <Badge variant={isRPGM ? "default" : "secondary"}>
                  RPG Maker MV/MZ
                </Badge>
              </div>
              <p className="text-[11px] text-muted-foreground">
                Zero file overwrite. Installs <code className="text-primary font-mono">Lingo_TranslationLayer.js</code> and loads translations on the fly.
              </p>
            </div>

            {/* Export Copy */}
            <div
              onClick={() => setMode("copy")}
              className={`p-3 rounded-md border transition-all cursor-pointer ${
                mode === "copy"
                  ? "border-primary bg-primary/10"
                  : "border-border bg-card hover:bg-muted"
              }`}
            >
              <div className="flex items-center justify-between mb-1">
                <span className="font-semibold text-xs text-foreground">
                  Export Copy (Patched Game Folder)
                </span>
                <Badge variant="outline">All Engines</Badge>
              </div>
              <p className="text-[11px] text-muted-foreground">
                Clones game assets and patches translated strings directly into a new target folder.
              </p>
            </div>
          </div>

          {/* Destination path for copy mode */}
          {mode === "copy" && (
            <div>
              <label className="block text-xs font-semibold text-muted-foreground mb-1">
                Destination Folder
              </label>
              <div className="flex gap-2">
                <Input
                  value={destPath}
                  onChange={(e) => setDestPath(e.target.value)}
                  className="flex-1 font-mono text-xs"
                />
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handlePickDestDir}
                  className="gap-1"
                >
                  <FolderOpen className="w-3.5 h-3.5" />
                  Browse…
                </Button>
              </div>
            </div>
          )}

          {/* Language display name */}
          <div>
            <label className="block text-xs font-semibold text-muted-foreground mb-1">
              Language Display Name
            </label>
            <Input
              value={langName}
              onChange={(e) => setLangName(e.target.value)}
              placeholder="e.g. Thai"
            />
          </div>
        </div>

        <DialogFooter>
          <Button
            variant="secondary"
            onClick={() => onOpenChange(false)}
            disabled={deploying}
          >
            Cancel
          </Button>
          <Button onClick={handleDeploy} disabled={deploying} className="gap-1.5">
            {deploying ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                Deploying…
              </>
            ) : (
              <>
                <Rocket className="w-4 h-4" />
                Deploy Now
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
