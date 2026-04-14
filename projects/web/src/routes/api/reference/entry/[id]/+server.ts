import { json, error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

// Reference entry detail backed by D1.
// Merges base fields from reference_fts with pre-computed spans/citations
// from reference_spans. Called client-side by ReferenceDrawer.svelte.
export const GET: RequestHandler = async ({ params, platform }) => {
	const id = parseInt(params.id, 10);
	if (isNaN(id)) throw error(400, 'Invalid entry ID');

	const db = platform?.env?.SEARCH_DB;
	if (!db) {
		console.error('[reference/entry] SEARCH_DB binding not available');
		throw error(503, 'Search database unavailable');
	}

	try {
		const base = await db
			.prepare(
				`SELECT rowid AS id, headword, raw_text, source_code, source_name
				 FROM reference_fts WHERE rowid = ?`
			)
			.bind(id)
			.first<{ id: number; headword: string; raw_text: string; source_code: string; source_name: string }>();

		if (!base) throw error(404, 'Entry not found');

		const spans = await db
			.prepare(`SELECT citation_spans, citations FROM reference_spans WHERE id = ?`)
			.bind(id)
			.first<{ citation_spans: string; citations: string }>();

		const citation_spans = spans ? JSON.parse(spans.citation_spans) : [];
		const citations = spans ? JSON.parse(spans.citations) : [];

		return json({ ...base, citation_spans, citations });
	} catch (err) {
		if (err && typeof err === 'object' && 'status' in err) throw err;
		console.error('[reference/entry] D1 query failed:', err);
		throw error(500, 'Query failed');
	}
};
