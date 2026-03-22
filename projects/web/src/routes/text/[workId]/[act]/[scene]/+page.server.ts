import { error, redirect } from '@sveltejs/kit';
import { api } from '$lib/server/api';

export async function load({ params, url }) {
	const act = parseInt(params.act, 10);
	const scene = parseInt(params.scene, 10);

	if (isNaN(act) || isNaN(scene)) {
		throw error(400, 'Invalid parameters');
	}

	// Resolve workId: accept numeric ID or slug
	let workId: number;
	let slug: string;
	const maybeId = parseInt(params.workId, 10);

	if (!isNaN(maybeId)) {
		// Numeric ID — resolve to slug and redirect
		try {
			const works = await api.getWorks();
			const all = [...works.plays, ...works.poetry];
			const work = all.find((w) => w.id === maybeId);
			if (!work) throw error(404, 'Work not found');
			slug = work.slug;
			workId = maybeId;
			// Redirect to slug URL
			const qs = url.search || '';
			throw redirect(301, `/text/${slug}/${act}/${scene}${qs}`);
		} catch (e) {
			if (e && typeof e === 'object' && 'status' in e && (e as { status: number }).status === 301) throw e;
			throw error(404, 'Work not found');
		}
	} else {
		// Slug — resolve to ID
		slug = params.workId;
		try {
			const work = await api.getWorkBySlug(slug);
			workId = work.id;
		} catch {
			throw error(404, 'Work not found');
		}
	}

	try {
		const [sceneData, toc, references] = await Promise.all([
			api.getScene(workId, act, scene),
			api.getWorkTOC(workId),
			api.getSceneReferences(workId, act, scene)
		]);
		return {
			scene: sceneData,
			toc,
			references,
			workId,
			slug,
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
