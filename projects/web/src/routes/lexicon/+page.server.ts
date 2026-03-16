import { getDb } from '$lib/server/db';
import { getLexiconLetters, getLexiconEntries } from '$lib/server/queries';

export function load() {
	const db = getDb();
	return {
		letters: getLexiconLetters(db),
		entries: getLexiconEntries(db, 'A')
	};
}
