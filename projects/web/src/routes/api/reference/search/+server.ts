import { json, error } from '@sveltejs/kit';
import { apiProxy } from '$lib/server/api';

export async function GET({ url }) {
	try {
		const res = await apiProxy(`/api/reference/search${url.search}`);
		if (!res.ok) throw new Error(`API ${res.status}`);
		return json(await res.json());
	} catch (err) {
		console.error('[reference/search]', err);
		throw error(502, 'Search unavailable');
	}
}
