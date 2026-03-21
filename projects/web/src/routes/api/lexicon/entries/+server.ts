import { json } from '@sveltejs/kit';
import { getDb } from '$lib/server/db';

export function GET({ url }) {
	const offset = parseInt(url.searchParams.get('offset') ?? '0', 10);
	const limit = Math.min(parseInt(url.searchParams.get('limit') ?? '50', 10), 200);

	const db = getDb();
	const entries = db
		.prepare(
			`SELECT MIN(id) as id, base_key as key, MIN(orthography) as orthography,
			        COUNT(*) as variant_count
			 FROM lexicon_entries
			 GROUP BY base_key
			 ORDER BY base_key
			 LIMIT ? OFFSET ?`
		)
		.all(limit, offset) as { id: number; key: string; orthography: string | null; variant_count: number }[];

	return json({ entries, hasMore: entries.length === limit });
}
