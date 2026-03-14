import { test } from "@playwright/test";
import { insertMany, setupDatabase } from "../src/db";
import { processEntries } from "../src/util/processEntry";

test("scrape", async ({ page }) => {
  test.setTimeout(620000);

  // Instead of scraping, use the mock data
  console.log("Using mock data instead of scraping");

  // Create or connect to SQLite database
  const db = setupDatabase();
  console.log("Created tables with correct schema");

  const chunkSize = 1000;

  const entries = await processEntries(page, test, {
    chunkSize,
    timeoutBetweenChunks: 10000,
    onChunkComplete: async (chunk: EntryItem[]) => {
      insertMany(chunk, db);
      return;
    },
  });
  console.log(`Inserted ${entries.length} entries into the database`);

  // Get count of entries in each table
  const tables_to_count = [
    "lexicon_entries",
    "lexicon_descriptions",
    "lexicon_emphases",
    "lexicon_citations",
  ];
  for (const table of tables_to_count) {
    const count = db
      .prepare(`SELECT COUNT(*) as count FROM ${table}`)
      .get() as { count: number };
    console.log(`Table ${table} contains ${count.count} rows`);
  }

  // Close the database connection
  db.close();
  console.log("Database connection closed");
});
