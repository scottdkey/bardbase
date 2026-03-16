/**
 * Typed query helpers for the Shakespeare DB.
 * All functions accept a better-sqlite3 Database instance so they remain
 * usable in any server context without coupling to the singleton in db.ts.
 */
import type Database from 'better-sqlite3';
import type { Works, Characters, Editions, TextLines, LexiconEntries } from '$lib/generated/db';

// ─── Convenience re-exports (singular aliases for route ergonomics) ───────────
export type Work = Works;
export type Character = Characters;
export type Edition = Editions;
export type TextLine = TextLines;
export type LexiconEntry = LexiconEntries;

// ─── Works ────────────────────────────────────────────────────────────────────

export function getWorkList(db: Database.Database): Pick<Works, 'id' | 'title' | 'full_title' | 'work_type' | 'date_composed'>[] {
	return db
		.prepare(
			`SELECT id, title, full_title, work_type, date_composed
       FROM works
       ORDER BY title`
		)
		.all() as Pick<Works, 'id' | 'title' | 'full_title' | 'work_type' | 'date_composed'>[];
}

export function getPlaysAndPoetry(db: Database.Database): {
	plays: Pick<Works, 'id' | 'title' | 'work_type' | 'date_composed'>[];
	poetry: Pick<Works, 'id' | 'title' | 'work_type' | 'date_composed'>[];
} {
	type Row = Pick<Works, 'id' | 'title' | 'work_type' | 'date_composed'>;
	const sql = `SELECT id, title, work_type, date_composed FROM works WHERE work_type {filter} ORDER BY title`;
	const plays = db
		.prepare(sql.replace('{filter}', `IN ('Comedy', 'Tragedy', 'History')`))
		.all() as Row[];
	const poetry = db
		.prepare(sql.replace('{filter}', `NOT IN ('Comedy', 'Tragedy', 'History')`))
		.all() as Row[];
	return { plays, poetry };
}

export function getWork(db: Database.Database, id: number): Works | undefined {
	return db.prepare('SELECT * FROM works WHERE id = ?').get(id) as Works | undefined;
}

// ─── Characters ───────────────────────────────────────────────────────────────

export function getCharactersByWork(
	db: Database.Database,
	workId: number
): Pick<Characters, 'id' | 'char_id' | 'name' | 'abbrev' | 'description' | 'speech_count'>[] {
	return db
		.prepare(
			`SELECT id, char_id, name, abbrev, description, speech_count
       FROM characters
       WHERE work_id = ?
       ORDER BY speech_count DESC`
		)
		.all(workId) as Pick<
		Characters,
		'id' | 'char_id' | 'name' | 'abbrev' | 'description' | 'speech_count'
	>[];
}

// ─── Editions ─────────────────────────────────────────────────────────────────

export type EditionWithSource = Pick<Editions, 'id' | 'name' | 'short_code' | 'year'> & {
	source_name: string;
};

export function getEditionsByWork(db: Database.Database, workId: number): EditionWithSource[] {
	return db
		.prepare(
			`SELECT e.id, e.name, e.short_code, e.year, s.name AS source_name
       FROM editions e
       JOIN sources s ON s.id = e.source_id
       WHERE e.id IN (SELECT DISTINCT edition_id FROM text_lines WHERE work_id = ?)`
		)
		.all(workId) as EditionWithSource[];
}

// ─── Text lines ───────────────────────────────────────────────────────────────

export type TextLineWithCharacter = Pick<
	TextLines,
	'id' | 'act' | 'scene' | 'line_number' | 'content' | 'content_type'
> & { character_name: string | null };

export function getTextLines(
	db: Database.Database,
	workId: number,
	editionId: number
): TextLineWithCharacter[] {
	return db
		.prepare(
			`SELECT tl.id, tl.act, tl.scene, tl.line_number, tl.content, tl.content_type,
              c.name AS character_name
       FROM text_lines tl
       LEFT JOIN characters c ON c.id = tl.character_id
       WHERE tl.work_id = ? AND tl.edition_id = ?
       ORDER BY tl.act, tl.scene, tl.line_number`
		)
		.all(workId, editionId) as TextLineWithCharacter[];
}

// ─── Lexicon ──────────────────────────────────────────────────────────────────

export function getLexiconLetters(db: Database.Database): { letter: string; count: number }[] {
	return db
		.prepare(
			`SELECT letter, COUNT(*) AS count
       FROM lexicon_entries
       GROUP BY letter
       ORDER BY letter`
		)
		.all() as { letter: string; count: number }[];
}

export function getLexiconEntries(
	db: Database.Database,
	letter: string,
	limit = 100
): Pick<LexiconEntries, 'id' | 'key' | 'letter' | 'orthography' | 'full_text'>[] {
	return db
		.prepare(
			`SELECT id, key, letter, orthography, full_text
       FROM lexicon_entries
       WHERE letter = ?
       ORDER BY key
       LIMIT ?`
		)
		.all(letter, limit) as Pick<
		LexiconEntries,
		'id' | 'key' | 'letter' | 'orthography' | 'full_text'
	>[];
}

// ─── Stats ────────────────────────────────────────────────────────────────────

export interface DbStats {
	work_count: number;
	character_count: number;
	line_count: number;
	lexicon_count: number;
}

export function getStats(db: Database.Database): DbStats {
	return db
		.prepare(
			`SELECT
        (SELECT COUNT(*) FROM works)          AS work_count,
        (SELECT COUNT(*) FROM characters)     AS character_count,
        (SELECT COUNT(*) FROM text_lines)     AS line_count,
        (SELECT COUNT(*) FROM lexicon_entries) AS lexicon_count`
		)
		.get() as DbStats;
}
