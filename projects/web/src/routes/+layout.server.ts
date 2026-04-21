import { getAttributions, getWorks } from '$lib/server/api';
import { getDb } from '$lib/server/db';

export async function load({ platform }) {
	try {
		const db = getDb(platform);
		const [attributions, works] = await Promise.all([getAttributions(db), getWorks(db)]);
		return { attributions, works };
	} catch (err) {
		console.error('[layout] failed to load:', err);
		return { attributions: [], works: { plays: [], poetry: [] } };
	}
}
