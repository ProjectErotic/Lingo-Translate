import React from "react";
import { Badge, Progress } from "@/ui";
import type { Project } from "@bindings/lingo-translate/pkg/model";
import type { WorkspaceStats } from "@bindings/lingo-translate/pkg/storage";

interface StatusbarProps {
  project: Project | null;
  stats: WorkspaceStats | null;
}

export const Statusbar: React.FC<StatusbarProps> = ({ project, stats }) => {
  const percent = stats?.percent || 0;

  return (
    <div className="h-6 bg-card border-t border-border flex items-center justify-between px-3 text-[11px] text-muted-foreground select-none z-40">
      {/* Left items: Project name & Engine */}
      <div className="flex items-center gap-2 truncate">
        {project ? (
          <>
            <span className="font-semibold text-foreground truncate max-w-[200px]">
              {project.name || "Untitled Project"}
            </span>
            <span className="text-muted-foreground/60">·</span>
            <Badge variant="default" className="text-[10px] px-1.5 py-0 h-4 uppercase">
              {project.engine || "Engine"}
            </Badge>
            <span className="text-muted-foreground/60">·</span>
            <span className="text-muted-foreground">
              {project.source_lang || "Source"} → {project.target_lang || "Target"}
            </span>
          </>
        ) : (
          <span className="italic text-muted-foreground/70">No project open</span>
        )}
      </div>

      {/* Right items: Stats & Progress Bar */}
      {project && stats && (
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-1 font-mono">
            <span className="text-emerald-400 font-medium">
              {stats.translated.toLocaleString()}
            </span>
            <span className="text-muted-foreground/60">/</span>
            <span className="text-foreground">
              {stats.total.toLocaleString()}
            </span>
            <span className="text-muted-foreground ml-1">
              ({percent.toFixed(1)}%)
            </span>
          </div>

          <div className="w-24">
            <Progress value={percent} className="h-1.5" />
          </div>
        </div>
      )}
    </div>
  );
};
