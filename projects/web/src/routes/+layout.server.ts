import { api } from '$lib/server/api';
import { building } from '$app/environment';

export async function load() {
	// During prerender (build time), D1 is unavailable — return empty shell.
	// Sidebar data loads client-side on static pages; dynamic pages have D1 at runtime.
	if (building) return { attributions: [], works: { plays: [], poetry: [] } };
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
