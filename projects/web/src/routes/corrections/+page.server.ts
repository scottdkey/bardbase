import { api } from '$lib/server/api';

export async function load() {
	try {
		const issues = await api.getCorrections('all');
		return { issues };
	} catch (err) {
		console.error('[corrections] failed to load issues:', err);
		return { issues: [] };
	}
}
