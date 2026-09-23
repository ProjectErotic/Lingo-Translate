import React from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  Button,
  Badge,
} from "@/ui";
import { Info, Cpu, Layers, ShieldCheck, Zap } from "lucide-react";

interface AboutDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export const AboutDialog: React.FC<AboutDialogProps> = ({
  open,
  onOpenChange,
}) => {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-base">
            <Info className="w-5 h-5 text-primary" />
            About Lingo Translate
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2 text-xs text-muted-foreground">
          <div className="flex items-center gap-3 border-b border-border pb-3">
            <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-primary to-primary/70 flex items-center justify-center font-bold text-lg text-primary-foreground shadow-md">
              NST
            </div>
            <div>
              <h2 className="font-bold text-sm text-foreground">
                Game Translation Suite (Go Edition)
              </h2>
              <p className="text-[11px] text-muted-foreground">Version {__APP_VERSION__} · Wails v3 Desktop</p>
            </div>
          </div>

          <div className="space-y-2.5">
            <div className="flex items-start gap-2">
              <Zap className="w-4 h-4 text-amber-400 shrink-0 mt-0.5" />
              <div>
                <span className="font-semibold text-foreground block">Multi-Engine Extraction</span>
                <span>Native support for RPG Maker MV/MZ, Ren'Py, Godot, and Unity games.</span>
              </div>
            </div>

            <div className="flex items-start gap-2">
              <ShieldCheck className="w-4 h-4 text-emerald-400 shrink-0 mt-0.5" />
              <div>
                <span className="font-semibold text-foreground block">Safe Non-Destructive Layer</span>
                <span>Zero-risk translation layer for RPG Maker that never modifies original game files.</span>
              </div>
            </div>

            <div className="flex items-start gap-2">
              <Layers className="w-4 h-4 text-primary shrink-0 mt-0.5" />
              <div>
                <span className="font-semibold text-foreground block">Pure Go SQLite Storage</span>
                <span>Handles 50k+ translation entries with instant search, filtering, and paging.</span>
              </div>
            </div>

            <div className="flex items-start gap-2">
              <Cpu className="w-4 h-4 text-purple-400 shrink-0 mt-0.5" />
              <div>
                <span className="font-semibold text-foreground block">AI Translation & Memory</span>
                <span>Pluggable AI providers (Gemini, OpenAI, Google) with persistent TM cache.</span>
              </div>
            </div>
          </div>

          <div className="border-t border-border pt-3 flex flex-wrap gap-1.5">
            <Badge variant="outline">RPG Maker MV/MZ</Badge>
            <Badge variant="outline">Ren'Py</Badge>
            <Badge variant="outline">Godot</Badge>
            <Badge variant="outline">Unity</Badge>
            <Badge variant="secondary">Go 1.26</Badge>
            <Badge variant="secondary">Wails v3</Badge>
            <Badge variant="secondary">React 18 + TS</Badge>
          </div>
        </div>

        <DialogFooter>
          <Button onClick={() => onOpenChange(false)}>OK</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
