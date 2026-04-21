import type { PageServerLoad } from './$types';
import { getReferenceSources } from '$lib/server/api';
import { getDb } from '$lib/server/db';
import { CACHE_STATIC } from '$lib/server/cache';

export const load: PageServerLoad = async ({ platform, url, setHeaders }) => {
	const initialSource = url.searchParams.get('source') ?? '';
	const initialQuery = url.searchParams.get('q') ?? '';
	setHeaders({ 'cache-control': CACHE_STATIC });
	try {
		const sources = await getReferenceSources(getDb(platform));
		return { sources, initialSource, initialQuery };
	} catch (err) {
		console.error('[references] failed to load:', err);
		return { sources: [], initialSource, initialQuery };
	}
};
