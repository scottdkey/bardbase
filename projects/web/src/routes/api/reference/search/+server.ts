import { json, error } from '@sveltejs/kit';

// Direct proxy — pass query params through to Go API
export async function GET({ url }) {
	const API_URL =
		(typeof process !== 'undefined' ? process.env.API_URL : undefined) ??
		'http://localhost:8080';

	try {
		const res = await fetch(`${API_URL}/api/reference/search${url.search}`);
		if (!res.ok) throw new Error(`API ${res.status}`);
		const data = await res.json();
		return json(data);
	} catch (err) {
		console.error('[reference/search]', err);
		throw error(502, 'Search unavailable');
	}
}
