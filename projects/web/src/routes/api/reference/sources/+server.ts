import { json, error } from '@sveltejs/kit';
import { api } from '$lib/server/api';

export async function GET() {
	try {
		const sources = await api.getReferenceSources();
		return json(sources);
	} catch (err) {
		console.error('[reference/sources]', err);
		throw error(502, 'Sources unavailable');
	}
}
