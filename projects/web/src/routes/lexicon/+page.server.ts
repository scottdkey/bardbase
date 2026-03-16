import { getDb } from '$lib/server/db';
import type { LexiconEntry } from '$lib/types';

export function load() {
	const db = getDb();

	// Get letter counts for navigation
	const letters = db.prepare(`
		SELECT letter, COUNT(*) AS count
		FROM lexicon_entries
		GROUP BY letter
		ORDER BY letter
	`).all() as { letter: string; count: number }[];

	// First page of entries (A)
	const entries = db.prepare(`
		SELECT id, key, letter, orthography, full_text
		FROM lexicon_entries
		WHERE letter = 'A'
		ORDER BY key
		LIMIT 100
	`).all() as LexiconEntry[];

	return { letters, entries };
}
