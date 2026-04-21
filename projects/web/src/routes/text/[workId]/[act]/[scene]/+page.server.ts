import { error } from '@sveltejs/kit';
import { getScene, getSceneReferences, getWorkBySlug, getWorkTOC, getWorks } from '$lib/server/api';
import { getDb } from '$lib/server/db';

export const prerender = true;

// entries() enumerates all (slug, act, scene) triples the Go pipeline produced,
// so SvelteKit can bake each scene into static HTML at build time. Turso is
// hit once per scene during the build; at runtime the pages are served from
// the CF edge as plain files — zero DB load.
export async function entries() {
	const db = getDb();
	const works = await getWorks(db);
	const all = [...works.plays, ...works.poetry];
	const tocs = await Promise.all(
		all.map((w) => getWorkTOC(db, w.slug).then((toc) => ({ slug: w.slug, toc })))
	);
	const paths: { workId: string; act: string; scene: string }[] = [];
	for (const { slug, toc } of tocs) {
		for (const div of toc) {
			paths.push({ workId: slug, act: String(div.act), scene: String(div.scene) });
		}
	}
	return paths;
}

export async function load({ params, platform }) {
	const act = parseInt(params.act, 10);
	const scene = parseInt(params.scene, 10);

	if (isNaN(act) || isNaN(scene)) {
		throw error(400, 'Invalid parameters');
	}

	const db = getDb(platform);
	const slug = params.workId;

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

		// URL search params (hw/line/ed) are read client-side in +page.svelte —
		// they don't belong in the prerendered data.
		return {
			scene: sceneData,
			toc,
			references,
			workId,
			slug,
			act,
			sceneNum: scene
		};
	} catch (err) {
		console.error('[text/scene]', err);
		throw error(502, 'Scene unavailable');
	}
}
