import { json, error } from '@sveltejs/kit';
import { getDb } from '$lib/server/db';
import { getSceneText } from '$lib/server/queries';
import type { EntryGenerator } from './$types';

// Prerender all work/act/scene combinations at build time
export const entries: EntryGenerator = () => {
	const db = getDb();
	const rows = db
		.prepare(
			`SELECT DISTINCT tl.work_id, tl.act, tl.scene
			 FROM text_lines tl
			 WHERE tl.edition_id IN (1, 2, 3, 4, 5)
			   AND tl.act IS NOT NULL AND tl.scene IS NOT NULL
			 GROUP BY tl.work_id, tl.act, tl.scene`
		)
		.all() as { work_id: number; act: number; scene: number }[];
	return rows.map((r) => ({
		workId: String(r.work_id),
		act: String(r.act),
		scene: String(r.scene)
	}));
};

export function GET({ params }) {
	const workId = parseInt(params.workId, 10);
	const act = parseInt(params.act, 10);
	const scene = parseInt(params.scene, 10);

	if (isNaN(workId) || isNaN(act) || isNaN(scene)) {
		throw error(400, 'Invalid parameters');
	}

	const db = getDb();
	const result = getSceneText(db, workId, act, scene);
	if (!result) throw error(404, 'Scene not found');

	return json(result);
}
