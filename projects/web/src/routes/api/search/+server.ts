import { json, error } from '@sveltejs/kit';
import { api } from '$lib/server/api';

export async function GET({ url }) {
	const q = url.searchParams.get('q') ?? '';
	const limit = parseInt(url.searchParams.get('limit') ?? '20', 10);

	if (!q) return json([]);

	try {
		const results = await api.search(q, limit);
		return json(results);
	} catch (err) {
		console.error('[search]', err);
		throw error(502, 'Search unavailable — is the API running?');
	}
}
