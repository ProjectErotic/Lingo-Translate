import type { Command, ParsedKeybinding } from "./types";
import { parseKeybinding, matchesKeybinding } from "./normalizer";

function isEditableElement(target: EventTarget | null): boolean {
  if (!target || !(target instanceof HTMLElement)) return false;
  const tag = target.tagName.toLowerCase();
  if (tag === "input") {
    const inputType = (target as HTMLInputElement).type.toLowerCase();
    // Non-text inputs (checkbox, radio, button) don't capture text
    return !["checkbox", "radio", "button", "submit", "reset", "range", "color"].includes(inputType);
  }
  if (tag === "textarea") return true;
  if (target.isContentEditable) return true;
  return false;
}

export class CommandRegistry {
  private commands = new Map<string, Command>();
  private parsedKeybindings = new Map<string, ParsedKeybinding>();
  private listeners = new Set<() => void>();

  /**
   * Registers a single command. Returns an unregister function for easy cleanup.
   */
  public register(command: Command): () => void {
    this.commands.set(command.id, command);
    if (command.keybinding) {
      this.parsedKeybindings.set(command.id, parseKeybinding(command.keybinding));
    } else {
      this.parsedKeybindings.delete(command.id);
    }
    this.notify();

    return () => {
      this.unregister(command.id);
    };
  }

  /**
   * Registers multiple commands at once. Returns an unregister function that cleans up all of them.
   */
  public registerMany(commands: Command[]): () => void {
    for (const cmd of commands) {
      this.commands.set(cmd.id, cmd);
      if (cmd.keybinding) {
        this.parsedKeybindings.set(cmd.id, parseKeybinding(cmd.keybinding));
      } else {
        this.parsedKeybindings.delete(cmd.id);
      }
    }
    this.notify();

    return () => {
      for (const cmd of commands) {
        this.commands.delete(cmd.id);
        this.parsedKeybindings.delete(cmd.id);
      }
      this.notify();
    };
  }

  /**
   * Unregisters a command by ID.
   */
  public unregister(id: string): void {
    if (this.commands.delete(id)) {
      this.parsedKeybindings.delete(id);
      this.notify();
    }
  }

  /**
   * Returns all currently registered commands.
   */
  public getAll(): Command[] {
    return Array.from(this.commands.values());
  }

  /**
   * Returns commands applicable to a specific scope.
   */
  public getByScope(scope: string): Command[] {
    return this.getAll().filter((c) => (c.scope || "global") === scope || (c.scope || "global") === "global");
  }

  /**
   * Executes a command by its ID.
   */
  public execute(id: string): void {
    const cmd = this.commands.get(id);
    if (!cmd) {
      console.warn(`[CommandRegistry] Command not found: ${id}`);
      return;
    }
    if (cmd.when && !cmd.when()) {
      return;
    }
    cmd.run();
  }

  private debugMode = false;

  /** Enable or disable shortcut diagnostic logs in console */
  public setDebug(enabled = true): void {
    this.debugMode = enabled;
    console.info(`[CommandRegistry] Shortcut diagnostic logging ${enabled ? "ENABLED" : "DISABLED"}`);
  }

  public isDebugEnabled(): boolean {
    if (this.debugMode) return true;
    if (typeof window !== "undefined" && (window as any).__LINGO_KEYBINDINGS_DEBUG__) return true;
    return false;
  }

  /**
   * Dispatches a KeyboardEvent through the registry.
   * Priority:
   * 1. Commands matching the active scope.
   * 2. Global commands.
   *
   * Automatically checks input focus state, when-conditions, and physical W3C keycodes.
   */
  public handleKeyEvent(e: KeyboardEvent, activeScope = "global"): boolean {
    const isEditing = isEditableElement(e.target);
    const debug = this.isDebugEnabled();

    // Group commands into activeScope vs global
    const scopedCmds: Command[] = [];
    const globalCmds: Command[] = [];

    for (const cmd of this.commands.values()) {
      const scope = cmd.scope || "global";
      if (scope === activeScope && scope !== "global") {
        scopedCmds.push(cmd);
      } else if (scope === "global") {
        globalCmds.push(cmd);
      }
    }

    // Evaluate scoped commands first, then global
    const candidateList = [...scopedCmds, ...globalCmds];

    for (const cmd of candidateList) {
      if (!cmd.keybinding) continue;

      const parsed = this.parsedKeybindings.get(cmd.id);
      if (!parsed) continue;

      // Check if shortcut matches physical key & modifiers
      if (!matchesKeybinding(e, parsed)) {
        continue;
      }

      // If user is typing in an input/textarea, ignore shortcuts that have preventInInput: true
      const preventInInput = cmd.preventInInput ?? true;
      if (isEditing && preventInInput) {
        if (debug) {
          console.debug(`[Lingo Shortcut Suppressed in Input] command='${cmd.id}' keybinding='${cmd.keybinding}'`);
        }
        continue;
      }

      // Check conditional 'when' guard
      if (cmd.when && !cmd.when()) {
        if (debug) {
          console.debug(`[Lingo Shortcut Guard Failed] command='${cmd.id}' 'when' condition returned false`);
        }
        continue;
      }

      if (debug) {
        console.debug(`[Lingo Shortcut Triggered] command='${cmd.id}' keybinding='${cmd.keybinding}' scope='${cmd.scope || "global"}' (hardware code='${e.code}', printed key='${e.key}')`);
      }

      // Matched! Consume event and execute command
      e.preventDefault();
      e.stopPropagation();

      try {
        cmd.run();
      } catch (err) {
        console.error(`[CommandRegistry] Error executing command ${cmd.id}:`, err);
      }

      return true;
    }

    if (debug && (e.ctrlKey || e.metaKey || e.altKey)) {
      console.debug(`[Lingo Shortcut Unmatched] code='${e.code}' key='${e.key}' activeScope='${activeScope}' isEditing=${isEditing}`);
    }

    return false;
  }

  /** Subscribe to registry changes (for Command Palette or cheat sheets) */
  public subscribe(listener: () => void): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  private notify(): void {
    for (const listener of this.listeners) {
      try {
        listener();
      } catch (err) {
        console.error("[CommandRegistry] Listener error:", err);
      }
    }
  }
}

/** Default singleton instance for the app */
export const defaultCommandRegistry = new CommandRegistry();

// Expose on window object in browser / desktop runtime for instant DevTools diagnostics
if (typeof window !== "undefined") {
  (window as any).__LINGO_COMMANDS__ = defaultCommandRegistry;
}

