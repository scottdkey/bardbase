import { normalizeEntry } from "./normalizeEntry";

export const normalizeEntries = (entries: EntryItem[]): EntryItem[] => {
  return entries.map(normalizeEntry);
};
