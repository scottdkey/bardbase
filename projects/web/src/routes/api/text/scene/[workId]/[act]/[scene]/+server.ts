import { json, error } from '@sveltejs/kit';
import { api } from '$lib/server/api';

export async function GET({ params }) {
	const workId = parseInt(params.workId, 10);
	const act = parseInt(params.act, 10);
	const scene = parseInt(params.scene, 10);

	if (isNaN(workId) || isNaN(act) || isNaN(scene)) {
		throw error(400, 'Invalid parameters');
	}

	try {
		const result = await api.getScene(workId, act, scene);
		return json(result);
	} catch (err) {
		console.error('[text/scene]', err);
		throw error(502, 'Scene unavailable — is the API running?');
	}
}
