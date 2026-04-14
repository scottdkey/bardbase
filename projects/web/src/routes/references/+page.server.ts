import { api } from '$lib/server/api';

export const prerender = false;

export async function load() {
	try {
		const [sources, works] = await Promise.all([
			api.getReferenceSources(),
			api.getWorks()
		]);
		return {
			sources,
			works: [...works.plays, ...works.poetry].sort((a, b) => a.title.localeCompare(b.title))
		};
	} catch (err) {
		console.error('[references] failed to load:', err);
		return { sources: [], works: [] };
	}
}
