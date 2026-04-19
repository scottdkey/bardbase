import { error } from '@sveltejs/kit';

export const prerender = false;

export async function load({ params, fetch }) {
	const id = parseInt(params.id, 10);
	if (isNaN(id)) throw error(400, 'Invalid entry ID');

	const res = await fetch(`/api/reference/entry/${id}`);
	if (res.status === 404) throw error(404, 'Entry not found');
	if (!res.ok) throw error(502, 'Entry unavailable');

	const entry = await res.json();
	return { entry };
}
