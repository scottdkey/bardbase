import { error } from '@sveltejs/kit';
import { CACHE_STATIC } from '$lib/server/cache';

export const prerender = false;

export async function load({ params, fetch, setHeaders }) {
	const id = parseInt(params.id, 10);
	if (isNaN(id)) throw error(400, 'Invalid entry ID');

	setHeaders({ 'cache-control': CACHE_STATIC });
	const res = await fetch(`/api/lexicon/entry/${id}`);
	if (res.status === 404) throw error(404, 'Entry not found');
	if (!res.ok) throw error(502, 'Entry unavailable');

	const entry = await res.json();
	return { entry };
}
