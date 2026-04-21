import { json, error } from '@sveltejs/kit';
import { getLexiconKeys } from '$lib/server/api';
import { getDb } from '$lib/server/db';
import { CACHE_STATIC } from '$lib/server/cache';

export async function GET({ platform }) {
	try {
		const keys = await getLexiconKeys(getDb(platform));
		return json(keys, { headers: { 'cache-control': CACHE_STATIC } });
	} catch (err) {
		console.error('[lexicon/keys]', err);
		throw error(502, 'Keys unavailable');
	}
}
