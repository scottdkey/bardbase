import { api } from '$lib/server/api';

export async function load() {
	try {
		return { attributions: await api.getAttributions() };
	} catch (err) {
		console.error('[layout] failed to load attributions:', err);
		return { attributions: [] };
	}
}
