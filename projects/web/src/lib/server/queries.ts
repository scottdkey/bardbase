/**
 * Typed query helpers for the Shakespeare DB.
 * All functions accept a better-sqlite3 Database instance so they remain
 * usable in any server context without coupling to the singleton in db.ts.
 */
import type Database from 'better-sqlite3';
import type { Works, Characters, Editions, TextLines, LexiconEntries, LexiconSenses, LexiconCitations } from '$lib/generated/db';

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

export interface LexiconListItem {
	id: number;
	key: string;
	orthography: string | null;
}

export function getLexiconEntriesPage(
	db: Database.Database,
	letter: string,
	offset: number,
	limit: number
): LexiconListItem[] {
	return db
		.prepare(
			`SELECT id, key, orthography
       FROM lexicon_entries
       WHERE letter = ?
       ORDER BY key
       LIMIT ? OFFSET ?`
		)
		.all(letter, limit, offset) as LexiconListItem[];
}

export interface LexiconCitationDetail {
	id: number;
	entry_id: number;
	sense_id: number | null;
	work_id: number | null;
	work_abbrev: string | null;
	work_title: string | null;
	act: number | null;
	scene: number | null;
	line: number | null;
	quote_text: string | null;
	display_text: string | null;
	raw_bibl: string | null;
	matched_line: string | null;
	matched_line_number: number | null;
	matched_character: string | null;
	matched_edition_id: number | null;
}

export interface LexiconSenseDetail {
	id: number;
	entry_id: number;
	sense_number: number;
	sub_sense: string | null;
	definition_text: string | null;
}

export interface LexiconSubEntryDetail {
	id: number;
	key: string;
	entry_type: string | null;
	full_text: string | null;
	orthography: string | null;
	senses: LexiconSenseDetail[];
	citations: LexiconCitationDetail[];
}

export interface LexiconEntryDetail {
	id: number;
	key: string;
	orthography: string | null;
	entry_type: string | null;
	full_text: string | null;
	subEntries: LexiconSubEntryDetail[];
	senses: LexiconSenseDetail[];
	citations: LexiconCitationDetail[];
}

export function getLexiconEntryFull(db: Database.Database, id: number): LexiconEntryDetail | null {
	// Find the entry and its base_key so we can load all related entries
	// (e.g., A1-A7 all grouped under base_key "A").
	const entry = db
		.prepare('SELECT id, key, base_key, orthography, entry_type, full_text FROM lexicon_entries WHERE id = ?')
		.get(id) as (Pick<LexiconEntries, 'id' | 'key' | 'orthography' | 'entry_type' | 'full_text'> & { base_key: string }) | undefined;

	if (!entry) return null;

	// Get all entry IDs in this group (e.g., all entries with base_key = "A").
	const groupEntries = db
		.prepare('SELECT id, key, sense_group, orthography, entry_type, full_text FROM lexicon_entries WHERE base_key = ? ORDER BY sense_group, id')
		.all(entry.base_key) as { id: number; key: string; sense_group: number | null; orthography: string | null; entry_type: string | null; full_text: string | null }[];

	const entryIds = groupEntries.map(e => e.id);
	const placeholders = entryIds.map(() => '?').join(',');

	// Load senses from ALL entries in the group, keeping entry_id for grouping.
	const senses = db
		.prepare(
			`SELECT ls.id, ls.entry_id,
			        ls.sense_number,
			        ls.sub_sense, ls.definition_text
			 FROM lexicon_senses ls
			 JOIN lexicon_entries le ON le.id = ls.entry_id
			 WHERE ls.entry_id IN (${placeholders})
			 ORDER BY le.sense_group, ls.sense_number, COALESCE(ls.sub_sense, '')`)
		.all(...entryIds) as LexiconSenseDetail[];

	// Load citations from ALL entries in the group, deduplicating by location.
	const citations = db
		.prepare(
			`SELECT MIN(lc.id) AS id, lc.entry_id, lc.sense_id, lc.work_id, lc.work_abbrev,
			        w.title AS work_title,
			        lc.act, lc.scene, lc.line,
			        MAX(lc.quote_text) AS quote_text, lc.display_text, lc.raw_bibl,
			        tl.content AS matched_line,
			        tl.line_number AS matched_line_number,
			        tl.char_name AS matched_character,
			        cm.edition_id AS matched_edition_id
			 FROM lexicon_citations lc
			 LEFT JOIN works w ON w.id = lc.work_id
			 LEFT JOIN citation_matches cm ON cm.citation_id = lc.id
			   AND cm.id = (
			     SELECT cm2.id FROM citation_matches cm2
			     WHERE cm2.citation_id = lc.id
			     ORDER BY CASE WHEN cm2.edition_id = 3 THEN 0 ELSE 1 END, cm2.confidence DESC
			     LIMIT 1
			   )
			 LEFT JOIN text_lines tl ON tl.id = cm.text_line_id
			 WHERE lc.entry_id IN (${placeholders})
			 GROUP BY lc.entry_id, lc.work_id, COALESCE(lc.act, -1), COALESCE(lc.scene, -1), COALESCE(lc.line, -1)
			 ORDER BY w.title, lc.act, lc.scene, lc.line`
		)
		.all(...entryIds) as (LexiconCitationDetail & { entry_id: number })[];

	// Validate and correct citation line matches against the headword.
	// Group citations by scene to batch lookups.
	const headword = entry.base_key.replace(/\d+$/, '').toLowerCase();
	const hwEscaped = headword.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
	const hwPattern = new RegExp(`\\b${hwEscaped}`, 'i');

	// Batch: group citations needing correction by (work_id, edition_id, act, scene)
	type SceneKey = string;
	const needsCorrection = new Map<SceneKey, typeof citations>();
	for (const c of citations) {
		if (c.matched_line && hwPattern.test(c.matched_line)) continue;
		if (c.work_id == null || c.act == null || c.scene == null) continue;
		const edId = c.matched_edition_id ?? 3; // default to Perseus
		const key: SceneKey = `${c.work_id}:${edId}:${c.act}:${c.scene}`;
		const list = needsCorrection.get(key) ?? [];
		list.push(c);
		needsCorrection.set(key, list);
	}

	// For each scene with citations needing correction, load a window of lines
	const nearbyStmt = db.prepare(
		`SELECT line_number, content, char_name
		 FROM text_lines
		 WHERE work_id = ? AND edition_id = ? AND act = ? AND scene = ?
		   AND line_number BETWEEN ? AND ?
		 ORDER BY line_number`
	);

	for (const [key, cits] of needsCorrection) {
		const [workId, edId, act, scene] = key.split(':').map(Number);
		// Find the range we need to cover
		const lineNums = cits.map(c => c.matched_line_number ?? c.line ?? 0);
		const minLine = Math.min(...lineNums) - 5;
		const maxLine = Math.max(...lineNums) + 5;

		const nearbyLines = nearbyStmt.all(workId, edId, act, scene, minLine, maxLine) as {
			line_number: number;
			content: string;
			char_name: string | null;
		}[];

		for (const c of cits) {
			const target = c.matched_line_number ?? c.line ?? 0;
			// Search nearby first (±5 from target), closest first
			let found: typeof nearbyLines[0] | null = null;
			for (let offset = 0; offset <= 5; offset++) {
				for (const delta of offset === 0 ? [0] : [-offset, offset]) {
					const candidate = nearbyLines.find(l => l.line_number === target + delta);
					if (candidate && hwPattern.test(candidate.content)) {
						found = candidate;
						break;
					}
				}
				if (found) break;
			}

			if (found) {
				c.matched_line = found.content;
				c.matched_line_number = found.line_number;
				c.matched_character = found.char_name;
				c.line = found.line_number;
			}
		}
	}

	// Build sub-entries: group senses and citations by their parent entry
	const sensesByEntry = new Map<number, LexiconSenseDetail[]>();
	for (const s of senses) {
		const list = sensesByEntry.get(s.entry_id) ?? [];
		list.push(s);
		sensesByEntry.set(s.entry_id, list);
	}

	const citationsByEntry = new Map<number, LexiconCitationDetail[]>();
	for (const c of citations) {
		const eid = (c as any).entry_id as number;
		const list = citationsByEntry.get(eid) ?? [];
		list.push(c);
		citationsByEntry.set(eid, list);
	}

	const subEntries: LexiconSubEntryDetail[] = groupEntries.map(ge => ({
		id: ge.id,
		key: ge.key,
		entry_type: ge.entry_type,
		full_text: ge.full_text,
		orthography: ge.orthography,
		senses: sensesByEntry.get(ge.id) ?? [],
		citations: citationsByEntry.get(ge.id) ?? []
	}));

	// Use base_key as the display key (not "A1", just "A").
	return {
		id: entry.id,
		key: entry.base_key,
		orthography: entry.orthography,
		entry_type: entry.entry_type,
		full_text: entry.full_text,
		subEntries,
		senses,
		citations
	};
}

// ─── Scene text ──────────────────────────────────────────────────────────────

export interface SceneTextLine {
	id: number;
	line_number: number | null;
	content: string;
	content_type: string | null;
	character_name: string | null;
}

export interface SceneTextResult {
	work_title: string;
	act: number;
	scene: number;
	edition_name: string;
	lines: SceneTextLine[];
}

/** Default edition preference order: Perseus Globe first (matches Schmidt citations) */
const PREFERRED_EDITION_IDS = [3, 4, 5, 1, 2];

export function getSceneText(
	db: Database.Database,
	workId: number,
	act: number,
	scene: number,
	preferredEditionId?: number
): SceneTextResult | null {
	const work = db.prepare('SELECT title FROM works WHERE id = ?').get(workId) as { title: string } | undefined;
	if (!work) return null;

	// Pick best available edition for this work
	let editionId: number | null = null;
	let editionName = '';

	// Try the citation's matched edition first, then fall back to preference order
	const editionOrder = preferredEditionId
		? [preferredEditionId, ...PREFERRED_EDITION_IDS.filter((id) => id !== preferredEditionId)]
		: PREFERRED_EDITION_IDS;

	for (const eid of editionOrder) {
		const row = db
			.prepare('SELECT e.id, e.name FROM editions e WHERE e.id = ? AND EXISTS (SELECT 1 FROM text_lines WHERE work_id = ? AND edition_id = ? LIMIT 1)')
			.get(eid, workId, eid) as { id: number; name: string } | undefined;
		if (row) {
			editionId = row.id;
			editionName = row.name;
			break;
		}
	}
	if (editionId == null) return null;

	const lines = db
		.prepare(
			`SELECT tl.id, tl.line_number, tl.content, tl.content_type,
              COALESCE(
                c.name,
                (SELECT c2.name FROM characters c2
                 WHERE c2.work_id = tl.work_id
                   AND LOWER(c2.name) LIKE LOWER(REPLACE(tl.char_name, '.', '')) || '%'
                 LIMIT 1),
                tl.char_name
              ) AS character_name
       FROM text_lines tl
       LEFT JOIN characters c ON c.id = tl.character_id
       WHERE tl.work_id = ? AND tl.edition_id = ? AND tl.act = ? AND tl.scene = ?
         AND tl.id = (
           SELECT MIN(t2.id) FROM text_lines t2
           WHERE t2.work_id = tl.work_id AND t2.edition_id = tl.edition_id
             AND t2.act = tl.act AND t2.scene = tl.scene AND t2.line_number = tl.line_number
         )
       ORDER BY tl.line_number, tl.id`
		)
		.all(workId, editionId, act, scene) as SceneTextLine[];

	return { work_title: work.title, act, scene, edition_name: editionName, lines };
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
