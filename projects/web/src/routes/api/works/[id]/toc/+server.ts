import { json, error } from '@sveltejs/kit';
import { getWorkTOC } from '$lib/server/api';
import { getDb } from '$lib/server/db';

export async function GET({ params, platform }) {
	const id = parseInt(params.id, 10);
	if (isNaN(id)) throw error(400, 'Invalid work ID');

	try {
		const toc = await getWorkTOC(getDb(platform), id);
		return json(toc);
	} catch (err) {
		console.error('[works/toc]', err);
		throw error(502, 'TOC unavailable');
	}
}
