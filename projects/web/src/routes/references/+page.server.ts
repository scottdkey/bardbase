import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ platform }) => {
	try {
		const db = platform?.env?.SEARCH_DB;
		if (!db) {
			console.error('[references] SEARCH_DB binding not available');
			return { sources: [] };
		}

		const [schmidtRow, refSources] = await Promise.all([
			db.prepare('SELECT COUNT(DISTINCT key) AS cnt FROM lexicon_fts').first<{ cnt: number }>(),
			db
				.prepare(
					`SELECT source_code AS code, source_name AS name, COUNT(*) AS count
					 FROM reference_fts
					 GROUP BY source_code, source_name
					 ORDER BY source_name`
				)
				.all<{ code: string; name: string; count: number }>()
		]);

		const sources = [];
		if (schmidtRow) {
			sources.push({ code: 'schmidt', name: 'Schmidt Shakespeare Lexicon', count: schmidtRow.cnt });
		}
		for (const row of refSources.results ?? []) {
			sources.push(row);
		}

		return { sources };
	} catch (err) {
		console.error('[references] failed to load:', err);
		return { sources: [] };
	}
};
