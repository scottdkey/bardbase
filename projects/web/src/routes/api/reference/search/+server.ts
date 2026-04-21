import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ url, platform }) => {
	const q = (url.searchParams.get('q') ?? '').trim();
	const sourceCode = url.searchParams.get('source') ?? '';
	const sourcesParam = url.searchParams.get('sources') ?? '';
	const sourcesFilter = sourcesParam
		? new Set(sourcesParam.split(',').map((s) => s.trim()).filter(Boolean))
		: null;
	const limit = Math.min(parseInt(url.searchParams.get('limit') ?? '50', 10), 200);
	const offset = parseInt(url.searchParams.get('offset') ?? '0', 10);

	const db = platform?.env?.SEARCH_DB;
	if (!db) {
		console.error('[reference/search] SEARCH_DB binding not available');
		return json([]);
	}

	try {
		const results: unknown[] = [];

		const includeSchmidt = sourcesFilter
			? sourcesFilter.has('schmidt')
			: !sourceCode || sourceCode === 'schmidt';
		const nonSchmidtSources = sourcesFilter
			? [...sourcesFilter].filter((s) => s !== 'schmidt')
			: null;
		const includeRefs = sourcesFilter
			? nonSchmidtSources!.length > 0
			: !sourceCode || sourceCode !== 'schmidt';

		if (includeSchmidt) {
			const schmidtArgs: unknown[] = [];
			let schmidtQuery = `SELECT rowid AS id, key AS headword, full_text AS raw_text,
				'schmidt' AS source_code, 'Schmidt Shakespeare Lexicon' AS source_name
				FROM lexicon_fts`;

			if (q) {
				schmidtQuery += ` WHERE lexicon_fts MATCH ?`;
				schmidtArgs.push(q);
				schmidtQuery += ` ORDER BY CASE WHEN LOWER(key) = LOWER(?) THEN 0 WHEN LOWER(key) LIKE LOWER(?) || '%' THEN 1 ELSE 2 END, rank`;
				schmidtArgs.push(q, q);
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
			// Slim reference_fts has no source metadata — join through reference_entries → sources.
			const refArgs: unknown[] = [];
			const conditions: string[] = [];

			if (q) {
				conditions.push(`reference_fts MATCH ?`);
				refArgs.push(q);
			}
			if (sourceCode && sourceCode !== 'schmidt') {
				conditions.push(`s.short_code = ?`);
				refArgs.push(sourceCode);
			} else if (nonSchmidtSources && nonSchmidtSources.length > 0) {
				conditions.push(`s.short_code IN (${nonSchmidtSources.map(() => '?').join(',')})`);
				refArgs.push(...nonSchmidtSources);
			}

			const where = conditions.length ? `WHERE ${conditions.join(' AND ')}` : '';
			const orderBy = q
				? `ORDER BY CASE WHEN LOWER(re.headword) = LOWER(?) THEN 0 WHEN LOWER(re.headword) LIKE LOWER(?) || '%' THEN 1 ELSE 2 END, reference_fts.rank`
				: `ORDER BY re.headword`;
			if (q) {
				refArgs.push(q, q);
			}

			// FTS branch: use the MATCH so ranks work; otherwise query reference_entries directly.
			const refQuery = q
				? `SELECT re.id AS id, re.headword, re.raw_text,
					s.short_code AS source_code, s.name AS source_name
					FROM reference_fts
					JOIN reference_entries re ON re.id = reference_fts.rowid
					JOIN sources s ON s.id = re.source_id
					${where}
					${orderBy} LIMIT ? OFFSET ?`
				: `SELECT re.id AS id, re.headword, re.raw_text,
					s.short_code AS source_code, s.name AS source_name
					FROM reference_entries re
					JOIN sources s ON s.id = re.source_id
					${where}
					${orderBy} LIMIT ? OFFSET ?`;

			const refResult = await db
				.prepare(refQuery)
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
