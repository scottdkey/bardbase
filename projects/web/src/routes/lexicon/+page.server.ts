import { getLexiconLetters } from '$lib/server/api';
import { getDb } from '$lib/server/db';

export async function load({ platform }) {
	try {
		return { letters: await getLexiconLetters(getDb(platform)) };
	} catch (err) {
		console.error('[page] failed to load letters:', err);
		return { letters: [] };
	}
}
