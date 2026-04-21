import { json, error } from '@sveltejs/kit';
import { getReferenceSources } from '$lib/server/api';
import { getDb } from '$lib/server/db';
import { CACHE_STATIC } from '$lib/server/cache';

export async function GET({ platform }) {
	try {
		const sources = await getReferenceSources(getDb(platform));
		return json(sources, { headers: { 'cache-control': CACHE_STATIC } });
	} catch (err) {
		console.error('[reference/sources]', err);
		throw error(502, 'Sources unavailable');
	}
}
