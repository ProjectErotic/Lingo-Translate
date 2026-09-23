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
import { ProjectService } from "@bindings/lingo-translate/cmd/lingo-desktop";
import { toast } from "sonner";
import { FolderOpen, Sparkles, Loader2 } from "lucide-react";

interface OpenGameDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export const OpenGameDialog: React.FC<OpenGameDialogProps> = ({
  open,
  onOpenChange,
  onSuccess,
}) => {
  const [gamePath, setGamePath] = useState("");
  const [wsPath, setWsPath] = useState("");
  const [detectedEngine, setDetectedEngine] = useState<string | null>(null);
  const [srcLang, setSrcLang] = useState("Japanese");
  const [tgtLang, setTgtLang] = useState("Thai");
  const [extracting, setExtracting] = useState(false);

  const handlePickDirectory = async () => {
    try {
      const selected = await ProjectService.PickDirectory("Select Game Folder");
      if (selected) {
        setGamePath(selected);
        const detected = await ProjectService.DetectEngine(selected).catch(() => "");
        setDetectedEngine(detected || "unknown");
        setWsPath(`${selected}/workspace.nst`);
      }
    } catch (err: any) {
      toast.error(`Dialog error: ${err?.message || err}`);
    }
  };

  const handlePickWorkspace = async () => {
    try {
      const selected = await ProjectService.PickWorkspace(true);
      if (selected) {
        setWsPath(selected);
      }
    } catch (err: any) {
      toast.error(`Dialog error: ${err?.message || err}`);
    }
  };

  const handleExtract = async () => {
    if (!gamePath) {
      toast.error("Please select a game folder");
      return;
    }

    try {
      setExtracting(true);
      const stats = await ProjectService.CreateFromGame(
        gamePath,
        wsPath,
        srcLang,
        tgtLang
      );
      toast.success(
        `Extracted ${stats?.total_entries || 0} entries across ${stats?.files_scanned || 0} files!`
      );
      onOpenChange(false);
      onSuccess();
    } catch (err: any) {
      toast.error(`Extraction failed: ${err?.message || err}`);
    } finally {
      setExtracting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-base">
            <Sparkles className="w-5 h-5 text-primary" />
            Extract New Game Project
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2 text-sm">
          {/* Game Folder Selection */}
          <div>
            <label className="block text-xs font-semibold text-muted-foreground mb-1">
              Game Root Directory
            </label>
            <div className="flex gap-2">
              <Input
                value={gamePath}
                onChange={(e) => setGamePath(e.target.value)}
                placeholder="/path/to/game"
                className="flex-1 font-mono text-xs"
              />
              <Button
                variant="outline"
                size="sm"
                onClick={handlePickDirectory}
                className="gap-1"
              >
                <FolderOpen className="w-3.5 h-3.5" />
                Browse…
              </Button>
            </div>
            {detectedEngine && (
              <div className="mt-1.5 flex items-center gap-2">
                <span className="text-xs text-muted-foreground">Detected Engine:</span>
                <Badge variant={detectedEngine === "unknown" ? "warning" : "default"}>
                  {detectedEngine.toUpperCase()}
                </Badge>
              </div>
            )}
          </div>

          {/* Workspace Destination */}
          <div>
            <label className="block text-xs font-semibold text-muted-foreground mb-1">
              Save Workspace File (.nst)
            </label>
            <div className="flex gap-2">
              <Input
                value={wsPath}
                onChange={(e) => setWsPath(e.target.value)}
                placeholder="/path/to/workspace.nst"
                className="flex-1 font-mono text-xs"
              />
              <Button
                variant="outline"
                size="sm"
                onClick={handlePickWorkspace}
                className="gap-1"
              >
                <FolderOpen className="w-3.5 h-3.5" />
                Browse…
              </Button>
            </div>
          </div>

          {/* Languages */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-semibold text-muted-foreground mb-1">
                Source Language
              </label>
              <Input
                value={srcLang}
                onChange={(e) => setSrcLang(e.target.value)}
              />
            </div>
            <div>
              <label className="block text-xs font-semibold text-muted-foreground mb-1">
                Target Language
              </label>
              <Input
                value={tgtLang}
                onChange={(e) => setTgtLang(e.target.value)}
              />
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button
            variant="secondary"
            onClick={() => onOpenChange(false)}
            disabled={extracting}
          >
            Cancel
          </Button>
          <Button
            onClick={handleExtract}
            disabled={extracting || !gamePath}
            className="gap-1.5"
          >
            {extracting ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                Extracting Game Texts…
              </>
            ) : (
              <>
                <Sparkles className="w-4 h-4" />
                Start Extraction
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
