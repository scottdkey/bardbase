import { logger } from "./logger";

export // Order entries by linked list structure (previous_letter -> next_letter)
const orderEntries = (entries: EntryItem[]) => {
  const entriesByCurrentLetter: { [key: string]: EntryItem } = {};

  // Create a map of entries by currentLetter for quick lookup
  entries.forEach((entry: EntryItem) => {
    if (entry.currentWord) {
      entriesByCurrentLetter[entry.currentWord] = entry;
    }
  });

  // Find the first entry (with null previousLetter)
  let currentEntry = entries.find(
    (entry: EntryItem) => entry.previousWord === null
  );
  if (!currentEntry && entries.length > 0) {
    // If no entry has null previousLetter, just take the first one
    currentEntry = entries[0];
    logger.warn(
      "No entry with null previousLetter found, using first entry instead"
    );
  }

  const orderedEntries: EntryItem[] = [];
  const visitedLetters = new Set<string>();

  // Follow the chain of nextLetter references
  while (currentEntry && !visitedLetters.has(currentEntry.currentWord)) {
    orderedEntries.push(currentEntry);
    visitedLetters.add(currentEntry.currentWord);

    // Find the next entry in the chain
    if (
      currentEntry.nextWord &&
      entriesByCurrentLetter[currentEntry.nextWord]
    ) {
      currentEntry = entriesByCurrentLetter[currentEntry.nextWord];
    } else {
      currentEntry = undefined;
    }
  }

  // Add any remaining entries that weren't in the main chain
  entries.forEach((entry) => {
    if (entry.currentWord && !visitedLetters.has(entry.currentWord)) {
      orderedEntries.push(entry);
    }
  });

  return orderedEntries;
};
