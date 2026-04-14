import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

// Lexicon search backed by D1 (trigram FTS5).
// Called client-side from the lexicon page search autocomplete.
export const GET: RequestHandler = async ({ url, platform }) => {
	const q = (url.searchParams.get('q') ?? '').trim();
	const limit = Math.min(parseInt(url.searchParams.get('limit') ?? '20', 10), 100);

	if (!q) return json([]);

	const db = platform?.env?.SEARCH_DB;
	if (!db) {
		console.error('[search] SEARCH_DB binding not available');
		return json([]);
	}

	try {
		// Trigram tokenizer: query as-is, no prefix * needed.
		// GROUP BY key dedups entries that share a base_key.
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

		return json(result.results ?? []);
	} catch (err) {
		console.error('[search] D1 query failed:', err);
		return json([]);
	}
};
