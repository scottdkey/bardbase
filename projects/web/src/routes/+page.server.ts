import { getDb } from '$lib/server/db';

export function load() {
	const db = getDb();
	// Load first 50 base keys (grouped — A1-A7 show as one "A" entry).
	const entries = db
		.prepare(
			`SELECT MIN(id) as id, base_key as key, MIN(orthography) as orthography,
			        COUNT(*) as variant_count
			 FROM lexicon_entries
			 GROUP BY base_key
			 ORDER BY base_key
			 LIMIT 50`
		)
		.all() as { id: number; key: string; orthography: string | null; variant_count: number }[];

	return { entries };
}
