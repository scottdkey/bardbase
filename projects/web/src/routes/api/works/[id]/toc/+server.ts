import { json, error } from '@sveltejs/kit';
import { api } from '$lib/server/api';

export async function GET({ params }) {
	const id = parseInt(params.id, 10);
	if (isNaN(id)) throw error(400, 'Invalid work ID');

	try {
		const toc = await api.getWorkTOC(id);
		return json(toc);
	} catch (err) {
		console.error('[works/toc]', err);
		throw error(502, 'TOC unavailable');
	}
}
