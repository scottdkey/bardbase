import { getDb } from '$lib/server/db';
import { getFooterAttributions } from '$lib/server/queries';

export function load() {
	const db = getDb();
	return { attributions: getFooterAttributions(db) };
}
