import { getDb } from '$lib/server/db';

export function load() {
	const db = getDb();

	// Load ALL base keys for client-side search/filtering.
	// ~20k entries at ~50 bytes each ≈ 1MB — fine for static prerender.
	const entries = db
		.prepare(
			`SELECT MIN(id) as id, base_key as key, MIN(orthography) as orthography,
			        COUNT(*) as variant_count
			 FROM lexicon_entries
			 GROUP BY base_key
			 ORDER BY base_key`
		)
		.all() as { id: number; key: string; orthography: string | null; variant_count: number }[];

	return { entries };
}
