import { getDb } from '$lib/server/db';
import { error } from '@sveltejs/kit';
import { getWork, getCharactersByWork, getEditionsByWork, getTextLines } from '$lib/server/queries';

export function load({ params }: { params: { id: string } }) {
	const db = getDb();
	const id = Number(params.id);

	const work = getWork(db, id);
	if (!work) throw error(404, 'Work not found');

	const characters = getCharactersByWork(db, id);
	const editions = getEditionsByWork(db, id);

	const defaultEdition = editions[0];
	const lines = defaultEdition ? getTextLines(db, id, defaultEdition.id) : [];

	return { work, characters, editions, lines, currentEdition: defaultEdition };
}
