import { json, error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { CACHE_STATIC } from '$lib/server/cache';
import { getDb } from '$lib/server/db';

function slugify(title: string): string {
	return title.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '');
}

// Lookup reference entry by source_code + headword — stable across DB rebuilds.
// Called as: GET /api/reference/key?source=onions&headword=dead
// Headword is passed as a query param (not a path segment) because Bartlett
// headwords are full Shakespeare quotes containing slashes and special chars.
export const GET: RequestHandler = async ({ url, platform }) => {
	const sourceCode = url.searchParams.get('source');
	const headword = url.searchParams.get('headword');

	if (!sourceCode || !headword) throw error(400, 'Missing source or headword');

	try {
		const db = getDb(platform);

		const base = await db
			.prepare(
				`SELECT re.id, re.headword, re.raw_text,
				        s.short_code AS source_code, s.name AS source_name
				 FROM reference_entries re
				 JOIN sources s ON s.id = re.source_id
				 WHERE re.headword = ? AND s.short_code = ?
				 LIMIT 1`
			)
			.bind(headword, sourceCode)
			.first<{
				id: number;
				headword: string;
				raw_text: string;
				source_code: string;
				source_name: string;
			}>();

		if (!base) throw error(404, 'Entry not found');

		const citationsResult = await db
			.prepare(
				`SELECT rc.id, rc.work_id, rc.work_abbrev, w.title AS work_title,
				        rc.act, rc.scene, rc.line
				 FROM reference_citations rc
				 LEFT JOIN works w ON w.id = rc.work_id
				 WHERE rc.entry_id = ?
				 ORDER BY rc.id`
			)
			.bind(base.id)
			.all<{
				id: number;
				work_id: number | null;
				work_abbrev: string | null;
				work_title: string | null;
				act: number | null;
				scene: number | null;
				line: number | null;
			}>();

		const citations = (citationsResult.results ?? []).map((c) => ({
			...c,
			work_slug: c.work_title ? slugify(c.work_title) : null
		}));

		return json(
			{ ...base, citation_spans: [], citations },
			{ headers: { 'cache-control': CACHE_STATIC } }
		);
	} catch (err) {
		if (err && typeof err === 'object' && 'status' in err) throw err;
		console.error('[reference/key] query failed:', err);
		throw error(500, 'Query failed');
	}
};
