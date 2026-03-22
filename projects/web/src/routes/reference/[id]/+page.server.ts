import { error } from '@sveltejs/kit';
import { api } from '$lib/server/api';

export async function load({ params }) {
	const id = parseInt(params.id, 10);
	if (isNaN(id)) throw error(400, 'Invalid entry ID');

	try {
		const entry = await api.getReferenceEntry(id);
		return { entry };
	} catch (err) {
		console.error('[reference]', err);
		throw error(404, 'Entry not found');
	}
}
