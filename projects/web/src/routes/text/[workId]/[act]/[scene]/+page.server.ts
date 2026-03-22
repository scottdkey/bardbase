import { error } from '@sveltejs/kit';
import { api } from '$lib/server/api';

export async function load({ params, url }) {
	const workId = parseInt(params.workId, 10);
	const act = parseInt(params.act, 10);
	const scene = parseInt(params.scene, 10);

	if (isNaN(workId) || isNaN(act) || isNaN(scene)) {
		throw error(400, 'Invalid parameters');
	}

	try {
		const [sceneData, toc] = await Promise.all([
			api.getScene(workId, act, scene),
			api.getWorkTOC(workId)
		]);
		return {
			scene: sceneData,
			toc,
			workId,
			act,
			sceneNum: scene,
			isReference: !!url.searchParams.get('hw'),
			headword: url.searchParams.get('hw') ?? '',
			line: url.searchParams.has('line')
				? parseInt(url.searchParams.get('line')!, 10)
				: null,
			editionId: url.searchParams.has('ed')
				? parseInt(url.searchParams.get('ed')!, 10)
				: null
		};
	} catch (err) {
		console.error('[text/scene]', err);
		throw error(502, 'Scene unavailable');
	}
}
