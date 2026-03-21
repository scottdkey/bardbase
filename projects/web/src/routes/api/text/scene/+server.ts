import { json, error } from '@sveltejs/kit';
import { getDb } from '$lib/server/db';
import { getSceneText, resolveScene } from '$lib/server/queries';

export function GET({ url }) {
	const workId = parseInt(url.searchParams.get('workId') ?? '', 10);
	const act = parseInt(url.searchParams.get('act') ?? '', 10);
	const sceneParam = url.searchParams.get('scene');
	const lineParam = url.searchParams.get('line');
	const editionParam = url.searchParams.get('editionId');
	const editionId = editionParam ? parseInt(editionParam, 10) : undefined;

	if (isNaN(workId) || isNaN(act)) {
		throw error(400, 'Missing or invalid workId or act');
	}

	const db = getDb();

	let scene: number;
	if (sceneParam && !isNaN(parseInt(sceneParam, 10))) {
		scene = parseInt(sceneParam, 10);
	} else if (lineParam && !isNaN(parseInt(lineParam, 10))) {
		// Resolve scene from line number when scene is not provided
		const resolved = resolveScene(db, workId, act, parseInt(lineParam, 10), editionId);
		if (resolved == null) throw error(404, 'Could not determine scene for this line');
		scene = resolved;
	} else {
		// Default to scene 1 (many acts only have one scene)
		scene = 1;
	}

	const result = getSceneText(db, workId, act, scene, editionId);
	if (!result) throw error(404, 'Scene not found');

	return json(result);
}
