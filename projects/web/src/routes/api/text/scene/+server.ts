import { json, error } from '@sveltejs/kit';
import { getDb } from '$lib/server/db';
import { getSceneText } from '$lib/server/queries';

export function GET({ url }) {
	const workId = parseInt(url.searchParams.get('workId') ?? '', 10);
	const act = parseInt(url.searchParams.get('act') ?? '', 10);
	const scene = parseInt(url.searchParams.get('scene') ?? '', 10);
	const editionParam = url.searchParams.get('editionId');
	const editionId = editionParam ? parseInt(editionParam, 10) : undefined;

	if (isNaN(workId) || isNaN(act) || isNaN(scene)) {
		throw error(400, 'Missing or invalid workId, act, or scene');
	}

	const db = getDb();
	const result = getSceneText(db, workId, act, scene, editionId);
	if (!result) throw error(404, 'Scene not found');

	return json(result);
}
