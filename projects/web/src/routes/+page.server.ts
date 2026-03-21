import { getDb } from '$lib/server/db';

export function load() {
	const db = getDb();
	const pageSize = 50;

	// Load first page of base keys (grouped — A1-A7 show as one "A" entry).
	const entries = db
		.prepare(
			`SELECT MIN(id) as id, base_key as key, MIN(orthography) as orthography,
			        COUNT(*) as variant_count
			 FROM lexicon_entries
			 GROUP BY base_key
			 ORDER BY base_key
			 LIMIT ?`
		)
		.all(pageSize) as { id: number; key: string; orthography: string | null; variant_count: number }[];

	const totalRow = db.prepare('SELECT COUNT(DISTINCT base_key) as count FROM lexicon_entries').get() as { count: number };

	return { entries, total: totalRow.count, pageSize };
}
