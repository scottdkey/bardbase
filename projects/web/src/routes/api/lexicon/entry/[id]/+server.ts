import { json, error } from '@sveltejs/kit';
import { getDb } from '$lib/server/db';
import { getLexiconEntryFull } from '$lib/server/queries';

export function GET({ params }) {
	const id = parseInt(params.id, 10);
	if (isNaN(id)) throw error(400, 'Invalid entry ID');

	const db = getDb();
	const entry = getLexiconEntryFull(db, id);
	if (!entry) throw error(404, 'Entry not found');

	return json(entry);
}
