import Database from "better-sqlite3";
import { v4 } from "uuid";

// Create or connect to SQLite database
export const dbPath = "./lexicon.db";
export const db = new Database(dbPath);
console.log(`Connected to SQLite database at ${dbPath}`);

export const setupDatabase = () => {
  const dbPath = "./lexicon.db";
  const db = new Database(dbPath);
  console.log(`Connected to SQLite database at ${dbPath}`);

  // Enable foreign keys
  db.pragma("foreign_keys = ON");

  // Drop existing tables if they exist
  const tables = [
    "lexicon_citations",
    "lexicon_descriptions",
    "lexicon_emphases",
    "lexicon_entries",
  ];

  for (const table of tables) {
    const tableExists = db
      .prepare(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`)
      .get(table);
    if (tableExists) {
      console.log(`Dropping existing table: ${table}`);
      db.exec(`DROP TABLE IF EXISTS ${table}`);
    }
  }

  // Create tables with new schema
  db.exec(`
    CREATE TABLE lexicon_entries (
      id TEXT PRIMARY KEY,
      entry_json JSON NOT NULL,
      word_value TEXT NOT NULL,
      word_key TEXT NOT NULL,
      letter_grouping TEXT NOT NULL,
      previous_letter TEXT,
      current_letter TEXT,
      next_letter TEXT,
      order_index INTEGER
    );

    CREATE TABLE lexicon_descriptions (
      id TEXT PRIMARY KEY,
      entry_id TEXT NOT NULL,
      description_value TEXT,
      description_index INTEGER NOT NULL,
      FOREIGN KEY (entry_id) REFERENCES lexicon_entries(id)
    );

    CREATE TABLE lexicon_emphases (
      id TEXT PRIMARY KEY,
      entry_id TEXT NOT NULL,
      emphasis_value TEXT NOT NULL,
      emphasis_index INTEGER NOT NULL,
      FOREIGN KEY (entry_id) REFERENCES lexicon_entries(id)
    );

    CREATE TABLE lexicon_citations (
      id TEXT PRIMARY KEY,
      entry_id TEXT NOT NULL,
      quote TEXT NOT NULL,
      title TEXT,
      act INTEGER,
      scene INTEGER,
      line INTEGER,
      is_sonnet BOOLEAN NOT NULL DEFAULT 0,
      is_poem BOOLEAN NOT NULL DEFAULT 0,
      FOREIGN KEY (entry_id) REFERENCES lexicon_entries(id)
    );

    CREATE TABLE lexicon_bibliography (
      id TEXT PRIMARY KEY,
      entry_id TEXT NOT NULL,
      raw_text TEXT,
      title TEXT,
      act INTEGER,
      scene INTEGER,
      line INTEGER,
      reference_key TEXT,
      bibliography_index INTEGER NOT NULL,
      FOREIGN KEY (entry_id) REFERENCES lexicon_entries(id)
    );

    CREATE INDEX idx_word_value ON lexicon_entries(word_value);
    CREATE INDEX idx_letter_grouping ON lexicon_entries(letter_grouping);
    CREATE INDEX idx_word_key ON lexicon_entries(word_key);
    CREATE INDEX idx_order_index ON lexicon_entries(order_index);
    CREATE INDEX idx_citations_title ON lexicon_citations(title);
    CREATE INDEX idx_citations_quote ON lexicon_citations(quote);
  `);

  return db;
};

export const prepareStatements = (database: any) => {
  // Prepare insert statements
  const insertEntryStmt = database.prepare(`
      INSERT INTO lexicon_entries (
      id, entry_json, word_value, word_key, letter_grouping, 
      previous_letter, current_letter, next_letter, order_index
    )
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
  `);

  const insertDescriptionStmt = database.prepare(`
    INSERT INTO lexicon_descriptions (
      id, entry_id, description_value, description_index
    )
    VALUES (?, ?, ?, ?)
  `);

  const insertEmphasisStmt = database.prepare(`
    INSERT INTO lexicon_emphases (
      id, entry_id, emphasis_value, emphasis_index
    )
    VALUES (?, ?, ?, ?)
  `);

  const insertCitationStmt = database.prepare(`
    INSERT INTO lexicon_citations (
      id, entry_id, quote, title, act, scene, line, is_sonnet, is_poem
    )
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
  `);

  const insertBibliographyStmt = database.prepare(`
    INSERT INTO lexicon_bibliography (
      id, entry_id, raw_text, title, act, scene, line, reference_key, bibliography_index
    )
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
  `);

  return {
    insertEntryStmt,
    insertDescriptionStmt,
    insertEmphasisStmt,
    insertCitationStmt,
    insertBibliographyStmt,
  };
};

export const insertMany = (entries: EntryItem[], db: any) => {
  // Get the prepared statements from the passed db
  const {
    insertEntryStmt,
    insertDescriptionStmt,
    insertEmphasisStmt,
    insertCitationStmt,
    insertBibliographyStmt,
  } = prepareStatements(db);

  // Create a transaction for better performance
  const insertManyTransaction = db.transaction((entries: EntryItem[]) => {
    for (let i = 0; i < entries.length; i++) {
      const entry = entries[i];
      try {
        const entryId = v4();
        const entryData = entry.entry || {};
        const wordValue = entryData.mappedObject?.value || "";
        const wordKey = entryData.mappedObject?.key || "";
        const letterGrouping =
          entryData.mappedObject?.letterGrouping ||
          (wordValue && typeof wordValue === "string" && wordValue.length > 0
            ? wordValue.charAt(0).toUpperCase()
            : "");

        // Insert main entry with order_index to maintain sequence
        insertEntryStmt.run(
          entryId,
          JSON.stringify(entry || {}),
          wordValue,
          wordKey,
          letterGrouping,
          entry.previousWord || null,
          entry.currentWord || null,
          entry.nextWord || null,
          i // Use array index as order_index
        );

        // Insert descriptions
        if (
          entryData.mappedObject?.description &&
          Array.isArray(entryData.mappedObject?.description)
        ) {
          entryData.mappedObject?.description.forEach(
            (
              desc: {
                value?: string;
                bibl?: any[];
              },
              index: number
            ) => {
              if (desc && desc.value !== undefined) {
                insertDescriptionStmt.run(v4(), entryId, desc.value, index);
              }
            }
          );
        }

        // Handle both string and array emphases
        if (entryData.mappedObject?.emph) {
          if (Array.isArray(entryData.mappedObject?.emph)) {
            entryData.mappedObject?.emph.forEach(
              (emphasis: string, index: number) => {
                if (emphasis) {
                  insertEmphasisStmt.run(
                    v4(),
                    entryId,
                    typeof emphasis === "string"
                      ? emphasis
                      : JSON.stringify(emphasis),
                    index
                  );
                }
              }
            );
          } else if (typeof entryData.mappedObject?.emph === "string") {
            // Handle single string emphasis
            insertEmphasisStmt.run(
              v4(),
              entryId,
              entryData.mappedObject?.emph,
              0
            );
          }
        }

        // Insert citations
        if (
          entryData.mappedObject?.citations &&
          Array.isArray(entryData.mappedObject?.citations)
        ) {
          entryData.mappedObject?.citations.forEach(
            (citation: {
              quote?: string;
              title?: string | null;
              act?: number | null;
              scene?: number | null;
              line?: number | null;
              isSonnet?: boolean;
              isPoem?: boolean;
            }) => {
              if (citation) {
                insertCitationStmt.run(
                  v4(),
                  entryId,
                  citation.quote || "",
                  citation.title || null,
                  citation.act || null,
                  citation.scene || null,
                  citation.line || null,
                  citation.isSonnet ? 1 : 0,
                  citation.isPoem ? 1 : 0
                );
              }
            }
          );
        }

        // Insert bibliographic references
        if (
          entryData.mappedObject?.bibl &&
          Array.isArray(entryData.mappedObject?.bibl)
        ) {
          entryData.mappedObject?.bibl.forEach(
            (
              bib: {
                _?: string;
                $?: {
                  n?: string;
                  default?: string;
                  valid?: string;
                };
                title?: string;
                act?: number | null;
                scene?: number | null;
                line?: number | null;
              },
              index: number
            ) => {
              if (bib) {
                // Create a bibliographic reference ID
                const biblId = v4();

                // Extract title, act, scene, line from the bibliographic reference
                // Handle both formats seen in the data
                let title = bib.title || (bib._ ? bib._.split(" ")[0] : null);
                let act = bib.act;
                let scene = bib.scene;
                let line = bib.line;

                // If the bibl contains a reference in format like "Tp. I, 2, 30"
                if (bib._ && !act && !scene && !line) {
                  const parts = bib._.split(",").map((part) => part.trim());
                  if (parts.length >= 3) {
                    // Extract act/scene/line from the format "Play I, 2, 30"
                    const titleParts = parts[0].split(" ");
                    // If we already have title from before, skip this
                    if (!title && titleParts.length > 0) {
                      title = titleParts[0];
                    }

                    // Handle roman numeral acts like "I" or arabic like "1"
                    act = parts[1]
                      ? parseInt(parts[1].replace(/[^\d]/g, "")) || null
                      : null;
                    scene = parts[2]
                      ? parseInt(parts[2].replace(/[^\d]/g, "")) || null
                      : null;
                    line =
                      parts.length > 3
                        ? parseInt(parts[3].replace(/[^\d]/g, "")) || null
                        : null;
                  }
                }

                // Insert into the appropriate table for bibliography
                insertBibliographyStmt.run(
                  biblId,
                  entryId,
                  bib._ || null,
                  title,
                  act || null,
                  scene || null,
                  line || null,
                  bib.$ && bib.$.n ? bib.$.n : null,
                  index
                );
              }
            }
          );
        }
      } catch (error) {
        console.error(`Error inserting entry: ${error}`);
        console.error(
          `Problematic entry: ${JSON.stringify(entry).substring(0, 200)}...`
        );
      }
    }
  });

  // Execute the transaction with the entries
  insertManyTransaction(entries);
};
