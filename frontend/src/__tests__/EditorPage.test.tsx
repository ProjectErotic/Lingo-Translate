import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor, cleanup } from "@testing-library/react";
import "@testing-library/jest-dom/vitest";
import { EditorPage } from "../pages/EditorPage";
import { TranslationStatus } from "@bindings/lingo-translate/pkg/model";

// Mock @wailsio/runtime
vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn((msg) => console.log("TOAST ERROR:", msg)),
  },
}));

// Mock EntryService
const mockFiles = [
  { path: "Map001.json", total: 10, translated: 5 },
  { path: "System.json", total: 4, translated: 4 },
];

const mockEntries = [
  {
    id: "entry-1",
    file_path: "Map001.json",
    key_path: "events[1].code",
    source: "こんにちは",
    target: "Hello",
    status: TranslationStatus.StatusTranslated,
    updated_at: "2026-01-01T00:00:00Z",
  },
  {
    id: "entry-2",
    file_path: "Map001.json",
    key_path: "events[2].code",
    source: "さようなら",
    target: "",
    status: TranslationStatus.StatusUntranslated,
    updated_at: "2026-01-01T00:00:00Z",
  },
];

vi.mock("@bindings/lingo-translate/cmd/lingo-desktop", () => ({
  EntryService: {
    Files: vi.fn(async () => mockFiles),
    Query: vi.fn(async (_q) => ({
      entries: mockEntries,
      total: mockEntries.length,
    })),
    Update: vi.fn(async (_id: string, _target: string) => {}),
    Stats: vi.fn(async () => ({ total: 14, translated: 9, percent: 64.3 })),
  },
  ProjectService: {
    PickDirectory: vi.fn(async () => "/dummy"),
    PickWorkspace: vi.fn(async () => "/dummy/workspace.nst"),
  },
}));

// Mock @tanstack/react-virtual so that it produces rows in jsdom
vi.mock("@tanstack/react-virtual", () => ({
  useVirtualizer: vi.fn(({ count }) => ({
    getTotalSize: () => count * 36,
    getVirtualItems: () =>
      Array.from({ length: count }, (_, index) => ({
        index,
        start: index * 36,
        size: 36,
        key: String(index),
      })),
  })),
}));

describe("EditorPage Grid & Editing Behavior", () => {
  const defaultProps = {
    onOpenTranslate: vi.fn(),
    onOpenDeploy: vi.fn(),
    selectedFile: "all",
    onSelectFile: vi.fn(),
    onStatsUpdated: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it("renders files and entry items correctly", async () => {
    render(<EditorPage {...defaultProps} />);
    await waitFor(() => {
      expect(screen.getByText("All Files")).toBeInTheDocument();
      expect(screen.getAllByText("Map001.json").length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByText("こんにちは").length).toBeGreaterThanOrEqual(1);
      expect(screen.getByText("さようなら")).toBeInTheDocument();
    });
  });

  it("selects an entry on click and populates the bottom detail pane", async () => {
    render(<EditorPage {...defaultProps} />);

    await waitFor(() => {
      expect(screen.getByText("さようなら")).toBeInTheDocument();
    });

    const row = screen.getByText("さようなら").closest("div[class*='cursor-pointer']");
    expect(row).toBeTruthy();
    fireEvent.click(row!);

    await waitFor(() => {
      expect(screen.getByText("Selected Entry Detail")).toBeInTheDocument();
      const textarea = screen.getByPlaceholderText("Type translated text here…");
      expect((textarea as HTMLTextAreaElement).value).toBe("");
    });
  });

  it("commits edit on blur and calls EntryService.Update", async () => {
    const { EntryService } = await import("@bindings/lingo-translate/cmd/lingo-desktop");
    render(<EditorPage {...defaultProps} />);

    await waitFor(() => {
      expect(screen.getAllByText("こんにちは").length).toBeGreaterThanOrEqual(1);
    });

    // Double click to enter inline edit mode
    const row = screen.getAllByText("こんにちは")[0].closest("div[class*='cursor-pointer']");
    fireEvent.doubleClick(row!);

    const input = row!.querySelector("input");
    expect(input).toBeTruthy();
    fireEvent.change(input!, { target: { value: "Hello World" } });
    fireEvent.blur(input!);

    await waitFor(() => {
      expect(EntryService.Update).toHaveBeenCalledWith("entry-1", "Hello World");
      expect(defaultProps.onStatsUpdated).toHaveBeenCalled();
    });
  });

  it("reverts edit on Escape without calling EntryService.Update", async () => {
    const { EntryService } = await import("@bindings/lingo-translate/cmd/lingo-desktop");
    render(<EditorPage {...defaultProps} />);

    await waitFor(() => {
      expect(screen.getAllByText("こんにちは").length).toBeGreaterThanOrEqual(1);
    });

    const row = screen.getAllByText("こんにちは")[0].closest("div[class*='cursor-pointer']");
    fireEvent.doubleClick(row!);

    const input = row!.querySelector("input");
    expect(input).toBeTruthy();
    fireEvent.change(input!, { target: { value: "Changed but cancelled" } });
    fireEvent.keyDown(input!, { key: "Escape" });

    expect(EntryService.Update).not.toHaveBeenCalled();
  });
});
