import React, { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  Button,
  Input,
} from "@/ui";
import {
  DeployService,
  ProjectService,
} from "@bindings/lingo-translate/cmd/lingo-desktop";
import type { MergeStats } from "@bindings/lingo-translate/pkg/merger";
import { toast } from "sonner";
import { GitMerge, FolderOpen, Loader2, CheckCircle2 } from "lucide-react";

interface UpdateVersionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export const UpdateVersionDialog: React.FC<UpdateVersionDialogProps> = ({
  open,
  onOpenChange,
  onSuccess,
}) => {
  const [newGamePath, setNewGamePath] = useState("");
  const [merging, setMerging] = useState(false);
  const [stats, setStats] = useState<MergeStats | null>(null);

  const handlePickDirectory = async () => {
    try {
      const selected = await ProjectService.PickDirectory("Select Updated Game Folder");
      if (selected) {
        setNewGamePath(selected);
      }
    } catch (err: any) {
      toast.error(`Dialog error: ${err?.message || err}`);
    }
  };

  const handleMerge = async () => {
    if (!newGamePath) {
      toast.error("Please choose the updated game folder");
      return;
    }

    try {
      setMerging(true);
      const res = await DeployService.Merge(newGamePath);
      setStats(res);
      toast.success("Game version merge completed!");
      onSuccess();
    } catch (err: any) {
      toast.error(`Merge failed: ${err?.message || err}`);
    } finally {
      setMerging(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-base">
            <GitMerge className="w-5 h-5 text-primary" />
            Smart Merge: Update Game Version
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2 text-sm">
          <p className="text-xs text-muted-foreground leading-relaxed">
            Reconciles your current translations with an updated version of the game (e.g. v1.0 → v1.1). Exact matches and fuzzy line relocations are automatically preserved.
          </p>

          <div>
            <label className="block text-xs font-semibold text-muted-foreground mb-1">
              New/Updated Game Folder
            </label>
            <div className="flex gap-2">
              <Input
                value={newGamePath}
                onChange={(e) => setNewGamePath(e.target.value)}
                placeholder="/path/to/updated_game_v1_1"
                className="flex-1 font-mono text-xs"
                disabled={merging}
              />
              <Button
                variant="outline"
                size="sm"
                onClick={handlePickDirectory}
                disabled={merging}
                className="gap-1"
              >
                <FolderOpen className="w-3.5 h-3.5" />
                Browse…
              </Button>
            </div>
          </div>

          {/* Merge Results Table */}
          {stats && (
            <div className="border border-border bg-card/60 p-3.5 rounded-md space-y-2">
              <div className="flex items-center gap-1.5 text-xs font-semibold text-emerald-400">
                <CheckCircle2 className="w-4 h-4" />
                Merge Summary Statistics
              </div>

              <div className="grid grid-cols-2 gap-2 text-xs pt-1">
                <div className="bg-card p-2 rounded border border-border">
                  <span className="text-muted-foreground block">Total In Incoming Version</span>
                  <span className="font-mono text-base font-bold text-foreground">
                    {stats.total_new}
                  </span>
                </div>
                <div className="bg-card p-2 rounded border border-border">
                  <span className="text-muted-foreground block">Exact Preserved</span>
                  <span className="font-mono text-base font-bold text-emerald-400">
                    {stats.exact_matches}
                  </span>
                </div>
                <div className="bg-card p-2 rounded border border-border">
                  <span className="text-muted-foreground block">Fuzzy Preserved</span>
                  <span className="font-mono text-base font-bold text-primary">
                    {stats.fuzzy_matches}
                  </span>
                </div>
                <div className="bg-card p-2 rounded border border-border">
                  <span className="text-muted-foreground block">New Lines To Translate</span>
                  <span className="font-mono text-base font-bold text-amber-400">
                    {stats.new_untranslated}
                  </span>
                </div>
              </div>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="secondary"
            onClick={() => onOpenChange(false)}
            disabled={merging}
          >
            {stats ? "Done" : "Cancel"}
          </Button>
          {!stats && (
            <Button
              onClick={handleMerge}
              disabled={merging || !newGamePath}
              className="gap-1.5"
            >
              {merging ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  Reconciling Version Differences…
                </>
              ) : (
                <>
                  <GitMerge className="w-4 h-4" />
                  Run Smart Merge
                </>
              )}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
