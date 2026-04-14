import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

// Reference search backed by D1 (trigram FTS5).
// Searches Schmidt lexicon + Onions, Abbott, Bartlett, Henley-Farmer.
// Called client-side from the references browse page.
export const GET: RequestHandler = async ({ url, platform }) => {
	const q = (url.searchParams.get('q') ?? '').trim();
	const sourceCode = url.searchParams.get('source') ?? '';
	const limit = Math.min(parseInt(url.searchParams.get('limit') ?? '50', 10), 200);
	const offset = parseInt(url.searchParams.get('offset') ?? '0', 10);

	const db = platform?.env?.SEARCH_DB;
	if (!db) {
		console.error('[reference/search] SEARCH_DB binding not available');
		return json([]);
	}

	try {
		const results: unknown[] = [];

		const includeSchmidt = !sourceCode || sourceCode === 'schmidt';
		const includeRefs = !sourceCode || sourceCode !== 'schmidt';

		if (includeSchmidt) {
			const schmidtArgs: unknown[] = [];
			let schmidtQuery = `SELECT rowid AS id, key AS headword, full_text AS raw_text,
				'schmidt' AS source_code, 'Schmidt Shakespeare Lexicon' AS source_name
				FROM lexicon_fts`;

			if (q) {
				schmidtQuery += ` WHERE lexicon_fts MATCH ?`;
				schmidtArgs.push(q);
			}

			const schmidtResult = await db
				.prepare(schmidtQuery + ` LIMIT ? OFFSET ?`)
				.bind(...schmidtArgs, limit, offset)
				.all();

			for (const row of schmidtResult.results ?? []) {
				const r = row as { raw_text?: string };
				if (r.raw_text && r.raw_text.length > 200) r.raw_text = r.raw_text.slice(0, 200) + '…';
				results.push(row);
			}
		}

		if (includeRefs) {
			const refArgs: unknown[] = [];
			let refQuery = `SELECT rowid AS id, headword, raw_text, source_code, source_name
				FROM reference_fts`;
			const conditions: string[] = [];

			if (q) {
				conditions.push(`reference_fts MATCH ?`);
				refArgs.push(q);
			}
			if (sourceCode && sourceCode !== 'schmidt') {
				conditions.push(`source_code = ?`);
				refArgs.push(sourceCode);
			}
			if (conditions.length) refQuery += ` WHERE ` + conditions.join(' AND ');

			const refResult = await db
				.prepare(refQuery + ` ORDER BY headword LIMIT ? OFFSET ?`)
				.bind(...refArgs, limit, offset)
				.all();

			for (const row of refResult.results ?? []) {
				const r = row as { raw_text?: string };
				if (r.raw_text && r.raw_text.length > 200) r.raw_text = r.raw_text.slice(0, 200) + '…';
				results.push(row);
			}
		}

		return json(results);
	} catch (err) {
		console.error('[reference/search] D1 query failed:', err);
		return json([]);
	}
};
