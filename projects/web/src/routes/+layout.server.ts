import { api } from '$lib/server/api';

// Layout loads from D1 at runtime — prerendering requires build-time data access
// which D1 doesn't support. Disabling prerender prevents getPlatformProxy/workerd
// from being invoked during vite build, which crashes in CI.
export const prerender = false;

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
