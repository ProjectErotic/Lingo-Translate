import React from "react";

// Regex matching NST masked tags and common RPG Maker / game escape codes
const TOKEN_REGEX = /(__LINGO_TAG_\d+__|\\[A-Za-z]+\[[^\]]*\]|\\[!><|.^$%\\])/g;
const SINGLE_LINE_TOKEN_REGEX = /(__LINGO_TAG_\d+__|\\[A-Za-z]+\[[^\]]*\]|\\[!><|.^$%\\]|\r\n|\r|\n)/g;

export interface TokenizedTextProps {
  text: string;
  className?: string;
  highlightSearch?: string;
  singleLine?: boolean;
}

export const TokenizedText: React.FC<TokenizedTextProps> = ({
  text,
  className = "",
  highlightSearch = "",
  singleLine = false,
}) => {
  if (!text) {
    return <span className="text-muted-foreground italic text-xs">(empty)</span>;
  }

  // Split text by tokens while capturing the matches
  const regex = singleLine ? SINGLE_LINE_TOKEN_REGEX : TOKEN_REGEX;
  const parts = text.split(regex);

  const containerClasses = singleLine
    ? `font-mono text-xs truncate block ${className}`
    : `font-mono text-xs whitespace-pre-wrap break-words leading-relaxed ${className}`;

  return (
    <span className={containerClasses}>
      {parts.map((part, index) => {
        if (!part) return null;

        // In single-line mode, collapse line breaks into a clean badge symbol
        if (singleLine && (part === "\n" || part === "\r" || part === "\r\n")) {
          return (
            <span
              key={index}
              className="text-primary/70 font-sans text-[10px] px-0.5 select-none font-bold inline-block"
              title="Line Break"
            >
              ↵{" "}
            </span>
          );
        }

        if (part.startsWith("__LINGO_TAG_") && part.endsWith("__")) {
          return (
            <span key={index} className="lingo-token select-all" title="Lingo Masked Tag">
              {part}
            </span>
          );
        }

        if (part.startsWith("\\")) {
          return (
            <span key={index} className="lingo-escape select-all" title="Game Engine Escape Code">
              {part}
            </span>
          );
        }

        // If search highlight is active
        if (highlightSearch && highlightSearch.trim() !== "") {
          const searchLower = highlightSearch.toLowerCase();
          const partLower = part.toLowerCase();
          const matchIndex = partLower.indexOf(searchLower);

          if (matchIndex !== -1) {
            const before = part.slice(0, matchIndex);
            const match = part.slice(matchIndex, matchIndex + highlightSearch.length);
            const after = part.slice(matchIndex + highlightSearch.length);

            return (
              <React.Fragment key={index}>
                {before}
                <mark className="bg-amber-400/40 text-foreground rounded-xs px-0.5">{match}</mark>
                {after}
              </React.Fragment>
            );
          }
        }

        // Plain string safely rendered by React
        return <React.Fragment key={index}>{part}</React.Fragment>;
      })}
    </span>
  );
};

