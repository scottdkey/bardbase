import { api } from '$lib/server/api';

export async function load() {
	try {
		const works = await api.getWorks();
		return { works };
	} catch (err) {
		console.error('[home] failed to load works:', err);
		return { works: { plays: [], poetry: [] } };
	}
}
