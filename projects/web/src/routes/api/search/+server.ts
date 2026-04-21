import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { CACHE_SHORT } from '$lib/server/cache';
import { getDb } from '$lib/server/db';

// Lexicon search backed by libSQL trigram FTS5.
// Called client-side from the lexicon page search autocomplete.
export const GET: RequestHandler = async ({ url, platform }) => {
	const q = (url.searchParams.get('q') ?? '').trim();
	const limit = Math.min(parseInt(url.searchParams.get('limit') ?? '20', 10), 100);

	if (!q) return json([]);

	try {
		const db = getDb(platform);
		const result = await db
			.prepare(
				`SELECT rowid AS id, key, orthography
				 FROM lexicon_fts
				 WHERE lexicon_fts MATCH ?
				 ORDER BY rank
				 LIMIT ?`
			)
			.bind(q, limit)
			.all();

		return json(result.results ?? [], { headers: { 'cache-control': CACHE_SHORT } });
	} catch (err) {
		console.error('[search] query failed:', err);
		return json([]);
	}
};
