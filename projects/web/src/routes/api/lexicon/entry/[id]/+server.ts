import { json, error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

// Lexicon entry detail backed by D1.
// Returns a stripped entry JSON (senses + reference-work links) for the
// inline ReferenceDrawer. Full entry detail lives on the prerendered
// /lexicon/entry/[id] page.
export const GET: RequestHandler = async ({ params, platform }) => {
	const id = parseInt(params.id, 10);
	if (isNaN(id)) throw error(400, 'Invalid entry ID');

	const db = platform?.env?.SEARCH_DB;
	if (!db) {
		console.error('[lexicon/entry] SEARCH_DB binding not available');
		throw error(503, 'Search database unavailable');
	}

	try {
		const row = await db
			.prepare(`SELECT data FROM lexicon_drawer WHERE id = ?`)
			.bind(id)
			.first<{ data: string }>();

		if (!row) throw error(404, 'Entry not found');

		return json(JSON.parse(row.data));
	} catch (err) {
		if (err && typeof err === 'object' && 'status' in err) throw err;
		console.error('[lexicon/entry] D1 query failed:', err);
		throw error(500, 'Query failed');
	}
};
