import { getDb } from '$lib/server/db';
import { getWorkList, getStats } from '$lib/server/queries';

export function load() {
	const db = getDb();
	return {
		works: getWorkList(db),
		stats: getStats(db)
	};
}
