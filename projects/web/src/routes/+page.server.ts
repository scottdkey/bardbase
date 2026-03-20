import { getDb } from '$lib/server/db';
import { getLexiconLetters, getLexiconEntriesPage } from '$lib/server/queries';

export function load() {
	const db = getDb();
	return {
		letters: getLexiconLetters(db),
		entries: getLexiconEntriesPage(db, 'A', 0, 50)
	};
}
