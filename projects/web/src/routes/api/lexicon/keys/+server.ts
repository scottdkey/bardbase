import { json, error } from '@sveltejs/kit';
import { api } from '$lib/server/api';

export async function GET() {
	try {
		const keys = await api.getLexiconKeys();
		return json(keys);
	} catch (err) {
		console.error('[lexicon/keys]', err);
		throw error(502, 'Keys unavailable');
	}
}
