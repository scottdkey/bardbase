import { json, error } from '@sveltejs/kit';
import { api } from '$lib/server/api';

export async function GET({ params }) {
	const id = parseInt(params.id, 10);
	if (isNaN(id)) throw error(400, 'Invalid entry ID');

	try {
		const entry = await api.getLexiconEntry(id);
		return json(entry);
	} catch (err) {
		console.error('[lexicon/entry]', err);
		throw error(502, 'Entry unavailable — is the API running?');
	}
}
