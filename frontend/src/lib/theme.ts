export type ThemeMode = "dark" | "light" | "midnight" | "system";
export type LayoutDensity = "comfortable" | "compact";
export type FontSize = "small" | "medium" | "large";

export function applyThemeSettings(theme?: string, density?: string, fontSize?: string) {
  const root = document.documentElement;

  if (theme) {
    if (theme === "system") {
      const prefersDark = window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches;
      root.setAttribute("data-theme", prefersDark ? "dark" : "light");
    } else {
      root.setAttribute("data-theme", theme);
    }
  }

  if (density) {
    root.setAttribute("data-density", density);
  }

  if (fontSize) {
    root.setAttribute("data-font-size", fontSize);
  }
}
