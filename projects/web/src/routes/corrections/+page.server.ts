import { getCorrections } from '$lib/server/api';

export async function load() {
	try {
		const issues = await getCorrections('all');
		return { issues };
	} catch (err) {
		console.error('[corrections] failed to load issues:', err);
		return { issues: [] };
	}
}
