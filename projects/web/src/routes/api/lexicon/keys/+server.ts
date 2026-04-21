import { json, error } from '@sveltejs/kit';
import { getLexiconKeys } from '$lib/server/api';
import { getDb } from '$lib/server/db';

export async function GET({ platform }) {
	try {
		const keys = await getLexiconKeys(getDb(platform));
		return json(keys);
	} catch (err) {
		console.error('[lexicon/keys]', err);
		throw error(502, 'Keys unavailable');
	}
}
