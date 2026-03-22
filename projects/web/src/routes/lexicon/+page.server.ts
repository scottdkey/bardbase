import { api } from '$lib/server/api';

export async function load() {
	try {
		return { letters: await api.getLexiconLetters() };
	} catch (err) {
		console.error('[page] failed to load letters:', err);
		return { letters: [] };
	}
}
