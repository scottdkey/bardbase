import { getDb } from '$lib/server/db';
import { getPlaysAndPoetry } from '$lib/server/queries';

export function load() {
	return getPlaysAndPoetry(getDb());
}
