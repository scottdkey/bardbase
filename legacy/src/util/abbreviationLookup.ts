// Define the Abbreviation interface
export interface Abbreviation {
  pattern: RegExp;
  description: string;
}

// Helper function to create consistent regex patterns
function createPattern(text: string, preventDotAfter = false): RegExp {
  if (preventDotAfter) {
    return new RegExp(`\\b${text}\\b(?!\\.)`, "g");
  }
  return new RegExp(`\\b${text}\\b`, "g");
}

export const abbreviationLookup: { [key: string]: Abbreviation } = {
  "Qq.": { pattern: createPattern("Qq\\."), description: "Quartos" },
  Q: { pattern: createPattern("Q", true), description: "Quarto" },
  Ff: { pattern: createPattern("Ff"), description: "First Folio" },
  F: { pattern: createPattern("F", true), description: "Folios" },
  "f. i.": { pattern: createPattern("f\\. i\\."), description: "for instance" },
  "cf.": { pattern: createPattern("cf\\."), description: "compare" },
  "i. e.": { pattern: createPattern("i\\. e\\."), description: "in example" },
  "q. v.": { pattern: createPattern("q\\. v\\."), description: "which see" },
  "M. Edd.": {
    pattern: createPattern("M\\. Edd\\."),
    description: "Modern Editors",
  },
  "auth.": { pattern: createPattern("auth\\."), description: "authentic" },
  "om.": { pattern: createPattern("om\\."), description: "omitted" },
  "ger.": { pattern: createPattern("ger\\."), description: "gerunds" },
  "art.": { pattern: createPattern("art\\."), description: "article" },
  "ind.": { pattern: createPattern("ind\\."), description: "indefinite" },
};
