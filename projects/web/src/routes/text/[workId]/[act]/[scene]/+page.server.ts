import { error, redirect } from '@sveltejs/kit';
import { api } from '$lib/server/api';

export async function load({ params, url }) {
	const act = parseInt(params.act, 10);
	const scene = parseInt(params.scene, 10);

	if (isNaN(act) || isNaN(scene)) {
		throw error(400, 'Invalid parameters');
	}

	const workParam = params.workId;
	let slug: string;
	let workId: number;

	// If numeric ID, redirect to slug URL
	const maybeId = parseInt(workParam, 10);
	if (!isNaN(maybeId)) {
		try {
			const works = await api.getWorks();
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

	// Slug — resolve ID for local use, but pass slug to API
	slug = workParam;
	try {
		const work = await api.getWorkBySlug(slug);
		workId = work.id;
	} catch {
		throw error(404, 'Work not found');
	}

	try {
		const [sceneData, toc, references] = await Promise.all([
			api.getScene(slug, act, scene),
			api.getWorkTOC(slug),
			api.getSceneReferences(slug, act, scene)
		]);
		return {
			scene: sceneData,
			toc,
			references,
			workId,
			slug,
			act,
			sceneNum: scene,
			isReference: !!url.searchParams.get('hw') || url.searchParams.has('line'),
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
