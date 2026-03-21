import { json, error } from '@sveltejs/kit';
import { getDb } from '$lib/server/db';
import { getLexiconEntryFull } from '$lib/server/queries';
import type { EntryGenerator } from './$types';

export const prerender = true;

// Prerender all entry IDs at build time
export const entries: EntryGenerator = () => {
	const db = getDb();
	const rows = db
		.prepare('SELECT MIN(id) as id FROM lexicon_entries GROUP BY base_key')
		.all() as { id: number }[];
	return rows.map((r) => ({ id: String(r.id) }));
};

export function GET({ params }) {
	const id = parseInt(params.id, 10);
	if (isNaN(id)) throw error(400, 'Invalid entry ID');

	const db = getDb();
	const entry = getLexiconEntryFull(db, id);
	if (!entry) throw error(404, 'Entry not found');

	return json(entry);
}
