export const normalizeEntry = (entry: EntryItem): EntryItem => {
  // Deep clone to avoid mutating the original
  const clonedEntry = JSON.parse(JSON.stringify(entry));

  // If entry has bibl as a single object, convert it to an array
  if (
    clonedEntry.entry?.original?.entryFree?.bibl &&
    !Array.isArray(clonedEntry.entry.original.entryFree.bibl)
  ) {
    clonedEntry.entry.original.entryFree.bibl = [
      clonedEntry.entry.original.entryFree.bibl,
    ];
  }

  return clonedEntry as EntryItem;
};
