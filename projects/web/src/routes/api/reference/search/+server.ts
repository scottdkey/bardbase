import { json, error } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';

export async function GET({ url }) {
	const API_URL =
		(typeof process !== 'undefined' ? process.env.API_URL : undefined) ??
		env.API_URL ??
		'http://localhost:8080';
	const API_KEY =
		(typeof process !== 'undefined' ? process.env.API_KEY : undefined) ??
		env.API_KEY ??
		undefined;

	const headers: Record<string, string> = {};
	if (API_KEY) headers['Authorization'] = `Bearer ${API_KEY}`;

	try {
		const res = await fetch(`${API_URL}/api/reference/search${url.search}`, { headers });
		if (!res.ok) throw new Error(`API ${res.status}`);
		const data = await res.json();
		return json(data);
	} catch (err) {
		console.error('[reference/search]', err);
		throw error(502, 'Search unavailable');
	}
}
