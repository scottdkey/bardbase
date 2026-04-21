import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ platform, url }) => {
	const initialSource = url.searchParams.get('source') ?? '';
	const initialQuery = url.searchParams.get('q') ?? '';
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
					`SELECT s.short_code AS code, s.name AS name, COUNT(*) AS count
					 FROM reference_entries re
					 JOIN sources s ON s.id = re.source_id
					 GROUP BY s.id
					 ORDER BY s.name`
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

		return { sources, initialSource, initialQuery };
	} catch (err) {
		console.error('[references] failed to load:', err);
		return { sources: [], initialSource, initialQuery };
	}
};
