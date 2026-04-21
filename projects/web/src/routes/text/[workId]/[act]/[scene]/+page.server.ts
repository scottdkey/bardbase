import { error, redirect } from '@sveltejs/kit';
import { getScene, getSceneReferences, getWorkBySlug, getWorkTOC, getWorks } from '$lib/server/api';
import { getDb } from '$lib/server/db';
import { CACHE_STATIC } from '$lib/server/cache';

export async function load({ params, url, platform, setHeaders }) {
	setHeaders({ 'cache-control': CACHE_STATIC });
	const act = parseInt(params.act, 10);
	const scene = parseInt(params.scene, 10);

	if (isNaN(act) || isNaN(scene)) {
		throw error(400, 'Invalid parameters');
	}

	const db = getDb(platform);
	const workParam = params.workId;

	// Numeric ID → redirect to slug URL
	const maybeId = parseInt(workParam, 10);
	if (!isNaN(maybeId)) {
		try {
			const works = await getWorks(db);
			const all = [...works.plays, ...works.poetry];
			const work = all.find((w) => w.id === maybeId);
			if (!work) throw error(404, 'Work not found');
			const qs = url.search || '';
			throw redirect(301, `/text/${work.slug}/${act}/${scene}${qs}`);
		} catch (e) {
			if (e && typeof e === 'object' && 'status' in e) throw e;
			throw error(404, 'Work not found');
		}
	}

	const slug = workParam;
	let workId: number;
	try {
		const work = await getWorkBySlug(db, slug);
		workId = work.id;
	} catch {
		throw error(404, 'Work not found');
	}

	try {
		const [sceneData, toc, references] = await Promise.all([
			getScene(db, slug, act, scene),
			getWorkTOC(db, slug),
			getSceneReferences(db, slug, act, scene)
		]);
		const headword = url.searchParams.get('hw') ?? '';
		const isReference = !!headword || url.searchParams.has('line');
		const line = url.searchParams.has('line') ? parseInt(url.searchParams.get('line')!, 10) : null;
		const editionId = url.searchParams.has('ed') ? parseInt(url.searchParams.get('ed')!, 10) : null;
		return {
			scene: sceneData,
			toc,
			references,
			workId,
			slug,
			act,
			sceneNum: scene,
			isReference,
			headword,
			line,
			editionId
		};
	} catch (err) {
		console.error('[text/scene]', err);
		throw error(502, 'Scene unavailable');
	}
}
