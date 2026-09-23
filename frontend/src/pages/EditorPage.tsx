import React, { useEffect, useState, useRef, useCallback, useMemo } from "react";
import {
  EntryService,
} from "@bindings/lingo-translate/cmd/lingo-desktop";
import { TranslationStatus, type TextEntry } from "@bindings/lingo-translate/pkg/model";
import type { FileSummary } from "@bindings/lingo-translate/pkg/storage";
import { Button, Input, TokenizedText } from "@/ui";
import { useCommands, useActiveScope } from "@/lib/commands";
import { useVirtualizer } from "@tanstack/react-virtual";
import { toast } from "sonner";
import {
  Search,
  FileText,
  ChevronLeft,
  ChevronRight,
  Sparkles,
  Rocket,
  Loader2,
} from "lucide-react";

interface EditorPageProps {
  onOpenTranslate: () => void;
  onOpenDeploy: () => void;
  selectedFile: string;
  onSelectFile: (file: string) => void;
  onStatsUpdated: () => void;
}

export const EditorPage: React.FC<EditorPageProps> = ({
  onOpenTranslate,
  onOpenDeploy,
  selectedFile,
  onSelectFile,
  onStatsUpdated,
}) => {
  // 1. Files state
  const [files, setFiles] = useState<FileSummary[]>([]);
  const [fileFilterSearch, setFileFilterSearch] = useState("");
  const [leftPanelWidth, setLeftPanelWidth] = useState(240);
  const [isResizingLeft, setIsResizingLeft] = useState(false);
  const [bottomPanelHeight, setBottomPanelHeight] = useState(190);
  const [isResizingBottom, setIsResizingBottom] = useState(false);

  // 2. Query filter state
  const [statusFilter, setStatusFilter] = useState<"all" | "untranslated" | "translated">("all");
  const [searchQuery, setSearchQuery] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const searchInputRef = useRef<HTMLInputElement>(null);

  // 3. Entries & Pagination state
  const [entries, setEntries] = useState<TextEntry[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(0);
  const pageSize = 100;

  // 4. Active selected entry & inline editing
  const [selectedIndex, setSelectedIndex] = useState<number | null>(null);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editValue, setEditValue] = useState("");
  const [originalValue, setOriginalValue] = useState("");

  // Debounce search input (300ms)
  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedSearch(searchQuery);
      setPage(0);
    }, 300);
    return () => clearTimeout(handler);
  }, [searchQuery]);

  // Set active command scope to 'editor'
  useActiveScope("editor");

  // Register editor-specific commands
  useCommands([
    {
      id: "editor.find",
      title: "Find in Current File",
      category: "Editor",
      keybinding: "Ctrl+F",
      scope: "editor",
      preventInInput: false,
      run: () => {
        searchInputRef.current?.focus();
        searchInputRef.current?.select();
      },
    },
  ]);

  // Load files list
  const loadFiles = useCallback(async () => {
    try {
      const fList = await EntryService.Files();
      setFiles(fList || []);
    } catch (err) {
      console.error("Failed to load files list:", err);
    }
  }, []);

  useEffect(() => {
    loadFiles();
  }, [loadFiles]);

  // Load entries query
  const loadEntries = useCallback(async () => {
    try {
      setLoading(true);
      const res = await EntryService.Query({
        file: selectedFile || "all",
        status: statusFilter,
        search: debouncedSearch,
        limit: pageSize,
        offset: page * pageSize,
      });

      setEntries(res.entries || []);
      setTotalCount(res.total || 0);

      // Default select first item if none selected or index out of range
      if (res.entries && res.entries.length > 0) {
        setSelectedIndex((prev) =>
          prev !== null && prev < res.entries.length ? prev : 0
        );
      } else {
        setSelectedIndex(null);
      }
    } catch (err: any) {
      toast.error(`Failed to load entries: ${err?.message || err}`);
    } finally {
      setLoading(false);
    }
  }, [selectedFile, statusFilter, debouncedSearch, page]);

  useEffect(() => {
    loadEntries();
  }, [loadEntries]);

  // Active entry selected for bottom detail pane
  const activeEntry =
    selectedIndex !== null && selectedIndex < entries.length
      ? entries[selectedIndex]
      : null;

  // Virtualizer for the table list
  const parentRef = useRef<HTMLDivElement>(null);
  const rowVirtualizer = useVirtualizer({
    count: entries.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 36, // fixed 36px row height
    overscan: 10,
  });

  // Handle saving an edit for a single entry
  const commitEdit = async (id: string, newTarget: string) => {
    if (newTarget === originalValue) {
      setEditingId(null);
      return;
    }

    try {
      await EntryService.Update(id, newTarget);

      // Local update without refetching entire list
      setEntries((prev) =>
        prev.map((item) =>
          item.id === id
            ? {
                ...item,
                target: newTarget,
                status:
                  newTarget.trim() !== ""
                    ? TranslationStatus.StatusTranslated
                    : TranslationStatus.StatusUntranslated,
              }
            : item
        )
      );

      toast.success("Saved");
      loadFiles();
      onStatsUpdated();
    } catch (err: any) {
      toast.error(`Failed to save: ${err?.message || err}`);
    } finally {
      setEditingId(null);
    }
  };

  // Start editing a row
  const startEditing = (entry: TextEntry) => {
    setEditingId(entry.id);
    setEditValue(entry.target || "");
    setOriginalValue(entry.target || "");
  };

  // Cancel editing
  const cancelEdit = () => {
    setEditingId(null);
    setEditValue(originalValue);
  };

  // Left Panel resize dragging
  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizingLeft) return;
      const newWidth = Math.max(160, Math.min(420, e.clientX));
      setLeftPanelWidth(newWidth);
    };

    const handleMouseUp = () => {
      setIsResizingLeft(false);
    };

    if (isResizingLeft) {
      window.addEventListener("mousemove", handleMouseMove);
      window.addEventListener("mouseup", handleMouseUp);
    }

    return () => {
      window.removeEventListener("mousemove", handleMouseMove);
      window.removeEventListener("mouseup", handleMouseUp);
    };
  }, [isResizingLeft]);

  // Bottom panel vertical resizer mouse listener
  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      const newHeight = window.innerHeight - e.clientY;
      if (newHeight >= 110 && newHeight <= 500) {
        setBottomPanelHeight(newHeight);
      }
    };

    const handleMouseUp = () => {
      setIsResizingBottom(false);
    };

    if (isResizingBottom) {
      window.addEventListener("mousemove", handleMouseMove);
      window.addEventListener("mouseup", handleMouseUp);
    }

    return () => {
      window.removeEventListener("mousemove", handleMouseMove);
      window.removeEventListener("mouseup", handleMouseUp);
    };
  }, [isResizingBottom]);

  // Filtered files list (memoized)
  const filteredFiles = useMemo(
    () =>
      files.filter((f) =>
        f.path.toLowerCase().includes(fileFilterSearch.toLowerCase())
      ),
    [files, fileFilterSearch]
  );

  const { totalAllFiles, translatedAllFiles } = useMemo(() => {
    return files.reduce(
      (acc, f) => ({
        totalAllFiles: acc.totalAllFiles + f.total,
        translatedAllFiles: acc.translatedAllFiles + f.translated,
      }),
      { totalAllFiles: 0, translatedAllFiles: 0 }
    );
  }, [files]);

  return (
    <div className="flex-1 flex flex-row overflow-hidden bg-background">
      {/* 1. Left Panel: File List */}
      <div
        style={{ width: `${leftPanelWidth}px` }}
        className="shrink-0 bg-card border-r border-border flex flex-col select-none relative"
      >
        {/* Panel Header */}
        <div className="p-2 border-b border-border space-y-1.5">
          <div className="flex items-center justify-between">
            <span className="text-xs font-bold text-foreground flex items-center gap-1.5">
              <FileText className="w-3.5 h-3.5 text-primary" />
              Files ({files.length})
            </span>
          </div>
          <Input
            value={fileFilterSearch}
            onChange={(e) => setFileFilterSearch(e.target.value)}
            placeholder="Filter files…"
            className="h-6 text-xs px-2 bg-background"
          />
        </div>

        {/* File items list */}
        <div className="flex-1 overflow-y-auto divide-y divide-border text-xs">
          {/* All Files item */}
          <div
            onClick={() => {
              onSelectFile("all");
              setPage(0);
            }}
            className={`px-3 py-2 cursor-pointer flex items-center justify-between transition-colors ${
              selectedFile === "all" || !selectedFile
                ? "bg-primary/20 text-primary font-semibold"
                : "text-muted-foreground hover:text-foreground hover:bg-muted"
            }`}
          >
            <span className="truncate">All Files</span>
            <span className="font-mono text-[10px] text-muted-foreground">
              {translatedAllFiles}/{totalAllFiles}
            </span>
          </div>

          {filteredFiles.map((f) => {
            const isSelected = selectedFile === f.path;
            const pct = f.total > 0 ? (f.translated / f.total) * 100 : 0;
            return (
              <div
                key={f.path}
                onClick={() => {
                  onSelectFile(f.path);
                  setPage(0);
                }}
                className={`px-3 py-2 cursor-pointer flex flex-col gap-1 transition-colors ${
                  isSelected
                    ? "bg-primary/20 text-primary font-semibold"
                    : "text-muted-foreground hover:text-foreground hover:bg-muted"
                }`}
              >
                <div className="flex items-center justify-between gap-1">
                  <span className="truncate" title={f.path}>
                    {f.path}
                  </span>
                  <span className="font-mono text-[10px] text-muted-foreground shrink-0">
                    {f.translated}/{f.total}
                  </span>
                </div>
                <div className="w-full bg-muted h-1 rounded-full overflow-hidden">
                  <div
                    className="bg-primary h-full transition-all"
                    style={{ width: `${pct}%` }}
                  />
                </div>
              </div>
            );
          })}
        </div>

        {/* Resizer Handle */}
        <div
          onMouseDown={() => setIsResizingLeft(true)}
          className="absolute right-0 top-0 bottom-0 w-1 hover:w-1.5 cursor-col-resize hover:bg-primary transition-all z-10"
        />
      </div>

      {/* 2. Main Grid & Detail Pane */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Toolbar */}
        <div className="h-10 bg-card border-b border-border px-3 flex items-center justify-between gap-2 text-xs shrink-0 select-none">
          {/* Status Filter Buttons */}
          <div className="flex items-center gap-1 bg-background p-0.5 rounded-md border border-border">
            {/* @ui-allow-native */}
            <button
              onClick={() => {
                setStatusFilter("all");
                setPage(0);
              }}
              className={`px-2 py-0.5 rounded text-xs transition-colors cursor-pointer ${
                statusFilter === "all"
                  ? "bg-primary text-primary-foreground font-medium"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              All
            </button>
            {/* @ui-allow-native */}
            <button
              onClick={() => {
                setStatusFilter("untranslated");
                setPage(0);
              }}
              className={`px-2 py-0.5 rounded text-xs transition-colors cursor-pointer ${
                statusFilter === "untranslated"
                  ? "bg-destructive text-destructive-foreground font-medium"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              Untranslated
            </button>
            {/* @ui-allow-native */}
            <button
              onClick={() => {
                setStatusFilter("translated");
                setPage(0);
              }}
              className={`px-2 py-0.5 rounded text-xs transition-colors cursor-pointer ${
                statusFilter === "translated"
                  ? "bg-emerald-600 text-white font-medium"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              Translated
            </button>
          </div>

          {/* Search Input */}
          <div className="flex-1 max-w-sm relative">
            <Search className="w-3.5 h-3.5 absolute left-2.5 top-2 text-muted-foreground" />
            <Input
              ref={searchInputRef}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search source or translation… (Ctrl+F)"
              className="h-7 pl-8 pr-7 text-xs bg-background"
            />
            {searchQuery && (
              // @ui-allow-native
              <button
                onClick={() => setSearchQuery("")}
                className="absolute right-2 top-1.5 text-xs text-muted-foreground hover:text-foreground cursor-pointer"
              >
                ✕
              </button>
            )}
          </div>

          {/* Action buttons */}
          <div className="flex items-center gap-2">
            <Button
              size="sm"
              variant="outline"
              onClick={onOpenTranslate}
              className="h-7 text-xs gap-1"
            >
              <Sparkles className="w-3 h-3 text-primary" />
              AI Translate…
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={onOpenDeploy}
              className="h-7 text-xs gap-1"
            >
              <Rocket className="w-3 h-3 text-amber-400" />
              Deploy…
            </Button>
          </div>
        </div>

        {/* Virtualized Table Grid */}
        <div
          ref={parentRef}
          className="flex-1 overflow-auto bg-background relative"
        >
          {/* Header Row */}
          <div className="sticky top-0 z-20 bg-card border-b border-border flex text-xs font-semibold text-muted-foreground select-none h-7 items-center">
            <div className="w-14 px-2 shrink-0 text-center font-mono text-[11px]">#</div>
            <div className="w-48 px-2 shrink-0 truncate">File & Key</div>
            <div className="flex-1 min-w-0 px-3 truncate">Original Source</div>
            <div className="flex-1 min-w-0 px-3 truncate">Target Translation</div>
          </div>

          {loading ? (
            <div className="p-8 text-center text-xs text-muted-foreground flex flex-col items-center justify-center gap-2">
              <Loader2 className="w-4 h-4 animate-spin text-primary" />
              <span>Loading entries…</span>
            </div>
          ) : entries.length === 0 ? (
            <div className="p-8 text-center text-xs text-muted-foreground">
              No entries match the current filter or search.
            </div>
          ) : (
            <div
              style={{
                height: `${rowVirtualizer.getTotalSize()}px`,
                width: "100%",
                position: "relative",
              }}
            >
              {rowVirtualizer.getVirtualItems().map((virtualRow) => {
                const entry = entries[virtualRow.index];
                const isSelected = selectedIndex === virtualRow.index;
                const isEditing = editingId === entry.id;
                const isTranslated =
                  (entry.target ?? "").trim() !== "" &&
                  entry.status !== "untranslated";

                return (
                  <div
                    key={entry.id}
                    onClick={() => setSelectedIndex(virtualRow.index)}
                    onDoubleClick={() => startEditing(entry)}
                    style={{
                      position: "absolute",
                      top: 0,
                      left: 0,
                      width: "100%",
                      height: `${virtualRow.size}px`,
                      transform: `translateY(${virtualRow.start}px)`,
                    }}
                    className={`flex items-center text-xs border-b border-border/50 cursor-pointer transition-colors overflow-hidden ${
                      isSelected
                        ? "bg-primary/20 text-foreground"
                        : virtualRow.index % 2 === 0
                        ? "bg-card/40 text-foreground hover:bg-muted"
                        : "bg-background text-foreground hover:bg-muted"
                    }`}
                  >
                    {/* Status Dot & Index */}
                    <div className="w-14 px-2 shrink-0 flex items-center gap-1.5 justify-start">
                      <span
                        className={`w-2 h-2 rounded-full shrink-0 ${
                          isTranslated ? "bg-emerald-400" : "bg-destructive"
                        }`}
                        title={isTranslated ? "Translated" : "Untranslated"}
                      />
                      <span className="font-mono text-[10px] text-muted-foreground truncate">
                        {page * pageSize + virtualRow.index + 1}
                      </span>
                    </div>

                    {/* Key / File path */}
                    <div
                      className="w-48 px-2 shrink-0 font-mono text-[11px] text-muted-foreground truncate"
                      title={`${entry.file_path} :: ${entry.key_path}`}
                    >
                      {selectedFile !== "all" ? (
                        <span className="text-foreground/90">{entry.key_path}</span>
                      ) : (
                        <>
                          <span className="text-foreground/80">{entry.file_path}</span>
                          <span className="text-muted-foreground/60"> : </span>
                          <span>{entry.key_path}</span>
                        </>
                      )}
                    </div>

                    {/* Source Text (Clipped single-line with token formatting) */}
                    <div
                      className="flex-1 min-w-0 px-3 overflow-hidden"
                      title={entry.source}
                    >
                      <TokenizedText
                        text={entry.source}
                        highlightSearch={debouncedSearch}
                        singleLine
                      />
                    </div>

                    {/* Target Translation (Inline editable or clipped single-line) */}
                    <div
                      className="flex-1 min-w-0 px-3 overflow-hidden"
                      title={entry.target || ""}
                    >
                      {isEditing ? (
                        // @ui-allow-native
                        <input
                          autoFocus
                          value={editValue}
                          onChange={(e) => setEditValue(e.target.value)}
                          onBlur={() => commitEdit(entry.id, editValue)}
                          onKeyDown={(e) => {
                            if ((e.key === "Enter" || e.code === "Enter" || e.code === "NumpadEnter") && !e.shiftKey) {
                              commitEdit(entry.id, editValue);
                            } else if (e.key === "Escape" || e.code === "Escape") {
                              cancelEdit();
                            }
                          }}
                          className="w-full bg-background text-foreground px-1.5 py-0.5 rounded border border-primary focus:outline-none font-mono text-xs"
                        />
                      ) : (
                        <span
                          className={
                            (entry.target ?? "").trim()
                              ? "text-emerald-300 font-mono text-xs block truncate"
                              : "text-muted-foreground/60 italic text-xs block truncate"
                          }
                        >
                          {(entry.target ?? "").trim() ? (
                            <TokenizedText
                              text={entry.target || ""}
                              highlightSearch={debouncedSearch}
                              singleLine
                            />
                          ) : (
                            "(untranslated)"
                          )}
                        </span>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Pagination Bar */}
        <div className="h-7 bg-card border-t border-border px-3 flex items-center justify-between text-[11px] text-muted-foreground select-none shrink-0">
          <span>
            Showing {Math.min(totalCount, page * pageSize + 1)}–
            {Math.min(totalCount, (page + 1) * pageSize)} of{" "}
            {totalCount.toLocaleString()} items
          </span>

          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="sm"
              disabled={page === 0}
              onClick={() => setPage((p) => Math.max(0, p - 1))}
              className="h-5 px-1.5 text-xs text-muted-foreground disabled:opacity-30"
            >
              <ChevronLeft className="w-3.5 h-3.5" />
              Previous
            </Button>
            <span className="font-mono text-foreground">
              Page {page + 1} of {Math.max(1, Math.ceil(totalCount / pageSize))}
            </span>
            <Button
              variant="ghost"
              size="sm"
              disabled={(page + 1) * pageSize >= totalCount}
              onClick={() => setPage((p) => p + 1)}
              className="h-5 px-1.5 text-xs text-muted-foreground disabled:opacity-30"
            >
              Next
              <ChevronRight className="w-3.5 h-3.5" />
            </Button>
          </div>
        </div>

        {/* Resizer Handle for Detail Panel */}
        <div
          onMouseDown={() => setIsResizingBottom(true)}
          className="h-1 -mt-0.5 hover:h-1.5 cursor-row-resize hover:bg-primary transition-all z-20 shrink-0"
        />

        {/* 3. Bottom Detail Panel */}
        <div
          style={{ height: `${bottomPanelHeight}px` }}
          className="bg-card border-t border-border flex flex-col shrink-0 p-3 select-none relative"
        >
          <div className="flex items-center justify-between pb-1.5 mb-1.5 border-b border-border text-xs">
            <span className="font-semibold text-foreground flex items-center gap-2">
              <span>Selected Entry Detail</span>
              {activeEntry && (
                <span className="font-mono text-[11px] text-muted-foreground font-normal">
                  {activeEntry.file_path} :: {activeEntry.key_path}
                </span>
              )}
            </span>

            {activeEntry && (
              <span className="text-[11px] text-muted-foreground">
                Press <kbd className="px-1 py-0.5 bg-muted border border-border rounded text-foreground">Ctrl+Enter</kbd> to save & advance
              </span>
            )}
          </div>

          {activeEntry ? (
            <div className="flex-1 grid grid-cols-2 gap-3 overflow-hidden text-xs">
              {/* Source Box */}
              <div className="flex flex-col bg-background border border-border rounded p-2 overflow-y-auto">
                <div className="flex items-center justify-between mb-1">
                  <span className="text-[10px] uppercase font-bold text-primary">
                    Source (Original)
                  </span>
                  <span className="text-[10px] text-muted-foreground font-mono">
                    {activeEntry.source.length} chars
                  </span>
                </div>
                <div className="flex-1 select-text">
                  <TokenizedText
                    text={activeEntry.source}
                    highlightSearch={debouncedSearch}
                  />
                </div>
              </div>

              {/* Target Box (Interactive Textarea) */}
              <div className="flex flex-col bg-background border border-border rounded p-2">
                <div className="flex items-center justify-between mb-1">
                  <span className="text-[10px] uppercase font-bold text-emerald-400">
                    Target (Translation)
                  </span>
                  <div className="flex items-center gap-2 text-[10px] text-muted-foreground">
                    <span className="font-mono">
                      {(editingId === activeEntry.id ? editValue : (activeEntry.target || "")).length} chars
                    </span>
                    <span>•</span>
                    <span>Status: {activeEntry.status}</span>
                  </div>
                </div>

                <textarea
                  value={editingId === activeEntry.id ? editValue : (activeEntry.target || "")}
                  onChange={(e) => {
                    if (editingId !== activeEntry.id) {
                      setEditingId(activeEntry.id);
                      setOriginalValue(activeEntry.target || "");
                    }
                    setEditValue(e.target.value);
                  }}
                  onKeyDown={(e) => {
                    if ((e.ctrlKey || e.metaKey) && (e.key === "Enter" || e.code === "Enter" || e.code === "NumpadEnter")) {
                      e.preventDefault();
                      commitEdit(activeEntry.id, editValue);
                      // Advance to next row
                      if (selectedIndex !== null && selectedIndex + 1 < entries.length) {
                        setSelectedIndex(selectedIndex + 1);
                      }
                    }
                  }}
                  onBlur={() => {
                    if (editingId === activeEntry.id) {
                      commitEdit(activeEntry.id, editValue);
                    }
                  }}
                  placeholder="Type translated text here…"
                  className="flex-1 bg-transparent text-foreground font-mono text-xs resize-none focus:outline-none placeholder:text-muted-foreground/50 overflow-y-auto"
                />
              </div>
            </div>
          ) : (
            <div className="flex-1 flex items-center justify-center text-xs text-muted-foreground/70 italic">
              Select a row in the grid above to view and edit its translation
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
