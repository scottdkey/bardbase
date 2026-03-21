import { json } from '@sveltejs/kit';
import { getDb } from '$lib/server/db';

export function GET({ url }) {
	const q = url.searchParams.get('q')?.trim() ?? '';
	const limit = Math.min(parseInt(url.searchParams.get('limit') ?? '50', 10), 200);

	if (q.length === 0) {
		return json({ entries: [] });
	}

	const db = getDb();

	// Search by base_key (groups A1-A7 as one result "A").
	// Returns one row per unique base_key with the first entry's id.
	let entries = db
		.prepare(
			`SELECT MIN(id) as id, base_key as key, MIN(orthography) as orthography,
			        COUNT(*) as variant_count
			 FROM lexicon_entries
			 WHERE base_key LIKE ? || '%'
			 GROUP BY base_key
			 ORDER BY base_key
			 LIMIT ?`
		)
		.all(q, limit) as { id: number; key: string; orthography: string | null; variant_count: number }[];

	if (entries.length === 0) {
		// Substring fallback
		entries = db
			.prepare(
				`SELECT MIN(id) as id, base_key as key, MIN(orthography) as orthography,
				        COUNT(*) as variant_count
				 FROM lexicon_entries
				 WHERE base_key LIKE '%' || ? || '%'
				 GROUP BY base_key
				 ORDER BY base_key
				 LIMIT ?`
			)
			.all(q, limit) as { id: number; key: string; orthography: string | null; variant_count: number }[];
	}

	return json({ entries });
}
