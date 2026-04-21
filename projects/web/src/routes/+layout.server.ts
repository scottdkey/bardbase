import { api } from '$lib/server/api';

export async function load() {
	try {
		const [attributions, works] = await Promise.all([
			api.getAttributions(),
			api.getWorks()
		]);
		return { attributions, works };
	} catch (err) {
		console.error('[layout] failed to load:', err);
		return { attributions: [], works: { plays: [], poetry: [] } };
	}
}
