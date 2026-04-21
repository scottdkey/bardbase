import { json, error } from '@sveltejs/kit';
import { getScene } from '$lib/server/api';
import { getDb } from '$lib/server/db';
import { CACHE_STATIC } from '$lib/server/cache';

export async function GET({ params, platform }) {
	const workId = parseInt(params.workId, 10);
	const act = parseInt(params.act, 10);
	const scene = parseInt(params.scene, 10);

	if (isNaN(workId) || isNaN(act) || isNaN(scene)) {
		throw error(400, 'Invalid parameters');
	}

	try {
		const result = await getScene(getDb(platform), workId, act, scene);
		return json(result, { headers: { 'cache-control': CACHE_STATIC } });
	} catch (err) {
		console.error('[text/scene]', err);
		throw error(502, 'Scene unavailable');
	}
}
