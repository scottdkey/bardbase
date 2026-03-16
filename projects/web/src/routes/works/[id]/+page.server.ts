import { getDb } from '$lib/server/db';
import { error } from '@sveltejs/kit';
import type { Work, Character, TextLine, Edition } from '$lib/types';

export function load({ params }) {
	const db = getDb();
	const id = Number(params.id);

	const work = db.prepare('SELECT * FROM works WHERE id = ?').get(id) as Work | undefined;
	if (!work) throw error(404, 'Work not found');

	const characters = db.prepare(`
		SELECT id, char_id, name, abbrev, description, speech_count
		FROM characters WHERE work_id = ? ORDER BY speech_count DESC
	`).all(id) as Character[];

	const editions = db.prepare(`
		SELECT e.id, e.name, e.short_code, e.year, s.name AS source_name
		FROM editions e
		JOIN sources s ON s.id = e.source_id
		WHERE e.id IN (SELECT DISTINCT edition_id FROM text_lines WHERE work_id = ?)
	`).all(id) as (Edition & { source_name: string })[];

	// Get text for the first available edition
	const defaultEdition = editions[0];
	let lines: TextLine[] = [];
	if (defaultEdition) {
		lines = db.prepare(`
			SELECT tl.id, tl.act, tl.scene, tl.line_number, tl.content,
				   tl.is_stage_direction, c.name AS character_name
			FROM text_lines tl
			LEFT JOIN characters c ON c.id = tl.character_id
			WHERE tl.work_id = ? AND tl.edition_id = ?
			ORDER BY tl.act, tl.scene, tl.line_number
		`).all(id, defaultEdition.id) as (TextLine & { character_name: string })[];
	}

	return { work, characters, editions, lines, currentEdition: defaultEdition };
}
