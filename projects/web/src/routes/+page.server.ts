import { api } from '$lib/server/api';

export const prerender = true;

export async function load() {
	try {
		const works = await api.getWorks();
		const all = [...works.plays, ...works.poetry];
		const tocEntries = await Promise.all(
			all.map((work) =>
				api
					.getWorkTOC(work.slug)
					.then((toc) => [work.id, toc] as const)
					.catch(() => [work.id, []] as const)
			)
		);
		const tocs = Object.fromEntries(tocEntries);
		return { works, tocs };
	} catch (err) {
		console.error('[home] failed to load works:', err);
		return { works: { plays: [], poetry: [] }, tocs: {} };
	}
}
