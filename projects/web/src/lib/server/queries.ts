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

export interface EditionLineRef {
	edition_id: number;
	edition_code: string;
	line_number: number | null;
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
	edition_lines: EditionLineRef[];
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

export interface ReferenceCitation {
	source_name: string;
	source_code: string;
	work_title: string | null;
	work_abbrev: string | null;
	act: number | null;
	scene: number | null;
	line: number | null;
	edition_lines: EditionLineRef[];
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
	references: ReferenceCitation[];
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
			        COALESCE(
			          (SELECT ch.name FROM characters ch WHERE ch.id = tl.character_id),
			          (SELECT ch2.name FROM characters ch2
			           WHERE ch2.work_id = tl.work_id
			             AND LOWER(ch2.name) LIKE LOWER(REPLACE(REPLACE(REPLACE(tl.char_name, '.', ''), 'æ', 'ae'), 'Æ', 'Ae')) || '%'
			           LIMIT 1),
			          (SELECT ch3.name FROM characters ch3
			           WHERE ch3.work_id = tl.work_id
			             AND (' ' || LOWER(ch3.name)) LIKE '% ' || LOWER(REPLACE(REPLACE(REPLACE(tl.char_name, '.', ''), 'æ', 'ae'), 'Æ', 'Ae')) || '%'
			           LIMIT 1),
			          tl.char_name
			        ) AS matched_character,
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
		const minLine = Math.min(...lineNums) - 10;
		const maxLine = Math.max(...lineNums) + 10;

		const nearbyLines = nearbyStmt.all(workId, edId, act, scene, minLine, maxLine) as {
			line_number: number;
			content: string;
			char_name: string | null;
		}[];

		for (const c of cits) {
			const target = c.matched_line_number ?? c.line ?? 0;
			// Search nearby first (±10 from target), closest first
			let found: typeof nearbyLines[0] | null = null;
			for (let offset = 0; offset <= 10; offset++) {
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
			} else {
				// No nearby line contains the headword — clear matched_line so the
				// UI falls back to quote_text instead of showing wrong text.
				c.matched_line = null;
			}
		}
	}

	// Batch-load cross-edition line numbers for all citations.
	// citation_matches already maps each citation to text_lines in multiple editions.
	const citationIds = citations.map(c => c.id);
	if (citationIds.length > 0) {
		const edLinePlaceholders = citationIds.map(() => '?').join(',');
		const edLineRows = db
			.prepare(
				`SELECT cm.citation_id, cm.edition_id, e.short_code AS edition_code, tl.line_number
				 FROM citation_matches cm
				 JOIN text_lines tl ON tl.id = cm.text_line_id
				 JOIN editions e ON e.id = cm.edition_id
				 WHERE cm.citation_id IN (${edLinePlaceholders})
				   AND cm.edition_id IN (1, 2, 3, 4, 5)
				 ORDER BY cm.citation_id, cm.edition_id`
			)
			.all(...citationIds) as { citation_id: number; edition_id: number; edition_code: string; line_number: number | null }[];

		const edLinesByCitation = new Map<number, EditionLineRef[]>();
		for (const row of edLineRows) {
			const list = edLinesByCitation.get(row.citation_id) ?? [];
			list.push({ edition_id: row.edition_id, edition_code: row.edition_code, line_number: row.line_number });
			edLinesByCitation.set(row.citation_id, list);
		}
		for (const c of citations) {
			c.edition_lines = edLinesByCitation.get(c.id) ?? [];
		}
	} else {
		for (const c of citations) {
			c.edition_lines = [];
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

	// Load reference work citations (Onions, Abbott, Bartlett, Henley & Farmer)
	const references = getReferenceCitationsForEntry(db, entry.base_key);

	// Use base_key as the display key (not "A1", just "A").
	return {
		id: entry.id,
		key: entry.base_key,
		orthography: entry.orthography,
		entry_type: entry.entry_type,
		full_text: entry.full_text,
		subEntries,
		senses,
		citations,
		references
	};
}

// ─── Reference citations ─────────────────────────────────────────────────────

function getReferenceCitationsForEntry(db: Database.Database, baseKey: string): ReferenceCitation[] {
	// Find reference entries matching this headword
	const refCitations = db
		.prepare(
			`SELECT rc.id, s.name AS source_name, s.short_code AS source_code,
			        w.title AS work_title, rc.work_abbrev,
			        rc.act, rc.scene, rc.line
			 FROM reference_citations rc
			 JOIN reference_entries re ON re.id = rc.entry_id
			 JOIN sources s ON s.id = rc.source_id
			 LEFT JOIN works w ON w.id = rc.work_id
			 WHERE LOWER(re.headword) = LOWER(?)
			 ORDER BY s.name, w.title, rc.act, rc.scene, rc.line`
		)
		.all(baseKey) as { id: number; source_name: string; source_code: string; work_title: string | null; work_abbrev: string | null; act: number | null; scene: number | null; line: number | null }[];

	if (refCitations.length === 0) return [];

	// Batch-load edition line numbers
	const refCitIds = refCitations.map(r => r.id);
	const refPlaceholders = refCitIds.map(() => '?').join(',');
	const refEdLines = db
		.prepare(
			`SELECT rcm.ref_citation_id, rcm.edition_id, e.short_code AS edition_code, tl.line_number
			 FROM reference_citation_matches rcm
			 JOIN text_lines tl ON tl.id = rcm.text_line_id
			 JOIN editions e ON e.id = rcm.edition_id
			 WHERE rcm.ref_citation_id IN (${refPlaceholders})
			   AND rcm.edition_id IN (1, 2, 3, 4, 5)
			 ORDER BY rcm.ref_citation_id, rcm.edition_id`
		)
		.all(...refCitIds) as { ref_citation_id: number; edition_id: number; edition_code: string; line_number: number | null }[];

	const refEdLinesByRef = new Map<number, EditionLineRef[]>();
	for (const row of refEdLines) {
		const list = refEdLinesByRef.get(row.ref_citation_id) ?? [];
		list.push({ edition_id: row.edition_id, edition_code: row.edition_code, line_number: row.line_number });
		refEdLinesByRef.set(row.ref_citation_id, list);
	}

	return refCitations.map(rc => ({
		source_name: rc.source_name,
		source_code: rc.source_code,
		work_title: rc.work_title,
		work_abbrev: rc.work_abbrev,
		act: rc.act,
		scene: rc.scene,
		line: rc.line,
		edition_lines: refEdLinesByRef.get(rc.id) ?? []
	}));
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

// ─── Multi-edition scene ─────────────────────────────────────────────────────

export interface EditionInfo {
	id: number;
	code: string;
	name: string;
}

export interface AlignedEditionLine {
	line_number: number | null;
	content: string;
	content_type: string | null;
	character_name: string | null;
}

export interface AlignedSceneRow {
	editions: Record<number, AlignedEditionLine | null>;
}

export interface MultiEditionScene {
	work_title: string;
	act: number;
	scene: number;
	available_editions: EditionInfo[];
	rows: AlignedSceneRow[];
}

export function getMultiEditionScene(
	db: Database.Database,
	workId: number,
	act: number,
	scene: number
): MultiEditionScene | null {
	const work = db.prepare('SELECT title, work_type FROM works WHERE id = ?').get(workId) as { title: string; work_type: string } | undefined;
	if (!work) return null;

	const isPoem = ['poem', 'poem_collection', 'sonnet_sequence'].includes(work.work_type);

	// Find which main editions have data for this scene
	const availEditions = db
		.prepare(
			isPoem
				? `SELECT DISTINCT e.id, e.short_code AS code, e.name
				   FROM editions e
				   JOIN text_lines tl ON tl.edition_id = e.id
				   WHERE tl.work_id = ? AND e.id IN (1,2,3,4,5)
				     AND (tl.scene = ? OR tl.scene IS NULL OR ? = 0)
				   ORDER BY e.id`
				: `SELECT DISTINCT e.id, e.short_code AS code, e.name
				   FROM editions e
				   JOIN text_lines tl ON tl.edition_id = e.id
				   WHERE tl.work_id = ? AND tl.act = ? AND tl.scene = ?
				     AND e.id IN (1,2,3,4,5)
				   ORDER BY e.id`
		)
		.all(...(isPoem ? [workId, scene, scene] : [workId, act, scene])) as EditionInfo[];

	if (availEditions.length === 0) return null;

	const charCoalesce = `COALESCE(
		c.name,
		(SELECT c2.name FROM characters c2
		 WHERE c2.work_id = tl.work_id
		   AND LOWER(c2.name) LIKE LOWER(REPLACE(REPLACE(REPLACE(tl.char_name, '.', ''), 'æ', 'ae'), 'Æ', 'Ae')) || '%'
		 LIMIT 1),
		(SELECT c3.name FROM characters c3
		 WHERE c3.work_id = tl.work_id
		   AND (' ' || LOWER(c3.name)) LIKE '% ' || LOWER(REPLACE(REPLACE(REPLACE(tl.char_name, '.', ''), 'æ', 'ae'), 'Æ', 'Ae')) || '%'
		 LIMIT 1),
		tl.char_name
	)`;

	// Load all lines for all available editions for this scene
	type LineRow = { id: number; edition_id: number; line_number: number | null; content: string; content_type: string | null; character_name: string | null };
	const editionIds = availEditions.map(e => e.id);
	const edPlaceholders = editionIds.map(() => '?').join(',');

	const sceneFilter = isPoem
		? (scene === 0 ? '1=1' : '(tl.scene = ? OR tl.scene IS NULL)')
		: 'tl.act = ? AND tl.scene = ?';
	const sceneParams = isPoem
		? (scene === 0 ? [] : [scene])
		: [act, scene];

	const allLines = db
		.prepare(
			`SELECT tl.id, tl.edition_id, tl.line_number, tl.content, tl.content_type,
			        ${charCoalesce} AS character_name
			 FROM text_lines tl
			 LEFT JOIN characters c ON c.id = tl.character_id
			 WHERE tl.work_id = ? AND tl.edition_id IN (${edPlaceholders})
			   AND ${sceneFilter}
			   AND tl.id = (
			     SELECT MIN(t2.id) FROM text_lines t2
			     WHERE t2.work_id = tl.work_id AND t2.edition_id = tl.edition_id
			       AND COALESCE(t2.act, 0) = COALESCE(tl.act, 0)
			       AND COALESCE(t2.scene, 0) = COALESCE(tl.scene, 0)
			       AND t2.line_number = tl.line_number
			   )
			 ORDER BY tl.edition_id, tl.line_number, tl.id`
		)
		.all(workId, ...editionIds, ...sceneParams) as LineRow[];

	// Group lines by edition, indexed by text_lines.id
	const linesByEdition = new Map<number, LineRow[]>();
	const lineById = new Map<number, LineRow>();
	for (const line of allLines) {
		const list = linesByEdition.get(line.edition_id) ?? [];
		list.push(line);
		linesByEdition.set(line.edition_id, list);
		lineById.set(line.id, line);
	}

	// Use edition 1 (OSS) as anchor if available, otherwise first available
	const anchorEditionId = editionIds.includes(1) ? 1 : editionIds[0];
	const anchorLines = linesByEdition.get(anchorEditionId) ?? [];

	if (availEditions.length <= 1) {
		// Only one edition — simple case, no alignment needed
		const rows: AlignedSceneRow[] = anchorLines.map(line => ({
			editions: { [anchorEditionId]: { line_number: line.line_number, content: line.content, content_type: line.content_type, character_name: line.character_name } }
		}));
		return { work_title: work.title, act, scene, available_editions: availEditions, rows };
	}

	// Load line_mappings between anchor and all other editions for this scene
	const otherEditionIds = editionIds.filter(id => id !== anchorEditionId);
	type MappingRow = { edition_a_id: number; edition_b_id: number; line_a_id: number | null; line_b_id: number | null; align_order: number; match_type: string };

	const mappingFilter = isPoem
		? (scene === 0 ? '1=1' : '(lm.scene = ? OR lm.scene = 0)')
		: 'lm.act = ? AND lm.scene = ?';
	const mappingParams = isPoem
		? (scene === 0 ? [] : [scene])
		: [act, scene];

	const otherPlaceholders = otherEditionIds.map(() => '?').join(',');
	const mappings = db
		.prepare(
			`SELECT lm.edition_a_id, lm.edition_b_id, lm.line_a_id, lm.line_b_id, lm.align_order, lm.match_type
			 FROM line_mappings lm
			 WHERE lm.work_id = ? AND ${mappingFilter}
			   AND (
			     (lm.edition_a_id = ? AND lm.edition_b_id IN (${otherPlaceholders}))
			     OR (lm.edition_b_id = ? AND lm.edition_a_id IN (${otherPlaceholders}))
			   )
			 ORDER BY lm.edition_b_id, lm.align_order`
		)
		.all(workId, ...mappingParams, anchorEditionId, ...otherEditionIds, anchorEditionId, ...otherEditionIds) as MappingRow[];

	// Build mapping: anchor line_id → other edition line_id, per edition
	// line_mappings stores pairs where edition_a_id < edition_b_id
	const anchorToOther = new Map<number, Map<number, number>>(); // anchorLineId → Map<otherEditionId, otherLineId>
	const otherOnlyLines = new Map<number, number[]>(); // editionId → [lineIds not in anchor]

	for (const m of mappings) {
		let anchorLineId: number | null;
		let otherLineId: number | null;
		let otherEdId: number;

		if (m.edition_a_id === anchorEditionId) {
			anchorLineId = m.line_a_id;
			otherLineId = m.line_b_id;
			otherEdId = m.edition_b_id;
		} else {
			anchorLineId = m.line_b_id;
			otherLineId = m.line_a_id;
			otherEdId = m.edition_a_id;
		}

		if (anchorLineId != null && otherLineId != null) {
			const edMap = anchorToOther.get(anchorLineId) ?? new Map();
			edMap.set(otherEdId, otherLineId);
			anchorToOther.set(anchorLineId, edMap);
		} else if (anchorLineId == null && otherLineId != null) {
			// Line exists only in other edition
			const list = otherOnlyLines.get(otherEdId) ?? [];
			list.push(otherLineId);
			otherOnlyLines.set(otherEdId, list);
		}
	}

	// Build aligned rows: walk anchor lines, look up corresponding lines in other editions
	const usedOtherLines = new Set<number>(); // track which other-edition lines have been placed
	const rows: AlignedSceneRow[] = [];

	for (const anchorLine of anchorLines) {
		const row: AlignedSceneRow = { editions: {} };
		row.editions[anchorEditionId] = {
			line_number: anchorLine.line_number,
			content: anchorLine.content,
			content_type: anchorLine.content_type,
			character_name: anchorLine.character_name
		};

		const edMap = anchorToOther.get(anchorLine.id);
		if (edMap) {
			for (const [otherEdId, otherLineId] of edMap) {
				const otherLine = lineById.get(otherLineId);
				if (otherLine) {
					row.editions[otherEdId] = {
						line_number: otherLine.line_number,
						content: otherLine.content,
						content_type: otherLine.content_type,
						character_name: otherLine.character_name
					};
					usedOtherLines.add(otherLineId);
				}
			}
		}

		rows.push(row);
	}

	// Append lines that exist only in non-anchor editions (only_b entries)
	for (const [otherEdId, lineIds] of otherOnlyLines) {
		for (const lineId of lineIds) {
			if (usedOtherLines.has(lineId)) continue;
			const otherLine = lineById.get(lineId);
			if (!otherLine) continue;

			// Find insertion point: after the last row that has a line
			// from this edition, or at the end
			const row: AlignedSceneRow = { editions: {} };
			row.editions[otherEdId] = {
				line_number: otherLine.line_number,
				content: otherLine.content,
				content_type: otherLine.content_type,
				character_name: otherLine.character_name
			};
			rows.push(row);
		}
	}

	return { work_title: work.title, act, scene, available_editions: availEditions, rows };
}

/** Resolve which scene a line number belongs to within a given act. */
export function resolveScene(
	db: Database.Database,
	workId: number,
	act: number,
	lineNumber: number,
	preferredEditionId?: number
): number | null {
	const editionOrder = preferredEditionId
		? [preferredEditionId, ...PREFERRED_EDITION_IDS.filter((id) => id !== preferredEditionId)]
		: PREFERRED_EDITION_IDS;

	for (const eid of editionOrder) {
		const row = db
			.prepare(
				`SELECT scene FROM text_lines
				 WHERE work_id = ? AND edition_id = ? AND act = ? AND line_number = ?
				 LIMIT 1`
			)
			.get(workId, eid, act, lineNumber) as { scene: number } | undefined;
		if (row) return row.scene;
	}

	// Fallback: return the first scene in the act
	const fallback = db
		.prepare(
			`SELECT DISTINCT scene FROM text_lines
			 WHERE work_id = ? AND act = ? AND scene IS NOT NULL
			 ORDER BY scene LIMIT 1`
		)
		.get(workId, act) as { scene: number } | undefined;
	return fallback?.scene ?? null;
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
	const work = db.prepare('SELECT title, work_type FROM works WHERE id = ?').get(workId) as { title: string; work_type: string } | undefined;
	if (!work) return null;

	const isPoem = ['poem', 'poem_collection', 'sonnet_sequence'].includes(work.work_type);

	// Pick best available edition for this work, checking the specific act/scene exists
	let editionId: number | null = null;
	let editionName = '';

	// Try the citation's matched edition first, then fall back to preference order
	const editionOrder = preferredEditionId
		? [preferredEditionId, ...PREFERRED_EDITION_IDS.filter((id) => id !== preferredEditionId)]
		: PREFERRED_EDITION_IDS;

	// For poems/sonnets, match flexibly on act/scene since editions differ
	// (OSS uses act=1, SE uses act=NULL, etc.)
	const editionCheckSql = isPoem
		? `SELECT e.id, e.name FROM editions e WHERE e.id = ? AND EXISTS (
			SELECT 1 FROM text_lines WHERE work_id = ? AND edition_id = ?
			AND (scene = ? OR scene IS NULL OR ? = 0) LIMIT 1)`
		: `SELECT e.id, e.name FROM editions e WHERE e.id = ? AND EXISTS (
			SELECT 1 FROM text_lines WHERE work_id = ? AND edition_id = ? LIMIT 1)`;

	for (const eid of editionOrder) {
		const row = isPoem
			? db.prepare(editionCheckSql).get(eid, workId, eid, scene, scene) as { id: number; name: string } | undefined
			: db.prepare(editionCheckSql).get(eid, workId, eid) as { id: number; name: string } | undefined;
		if (row) {
			editionId = row.id;
			editionName = row.name;
			break;
		}
	}
	if (editionId == null) return null;

	// Character name expansion COALESCE (shared between poem and play queries)
	const charCoalesce = `COALESCE(
                c.name,
                (SELECT c2.name FROM characters c2
                 WHERE c2.work_id = tl.work_id
                   AND LOWER(c2.name) LIKE LOWER(REPLACE(REPLACE(REPLACE(tl.char_name, '.', ''), 'æ', 'ae'), 'Æ', 'Ae')) || '%'
                 LIMIT 1),
                (SELECT c3.name FROM characters c3
                 WHERE c3.work_id = tl.work_id
                   AND (' ' || LOWER(c3.name)) LIKE '% ' || LOWER(REPLACE(REPLACE(REPLACE(tl.char_name, '.', ''), 'æ', 'ae'), 'Æ', 'Ae')) || '%'
                 LIMIT 1),
                tl.char_name
              )`;

	// For poems/sonnets, match flexibly on act/scene since editions vary.
	// Sonnets: scene = sonnet number (e.g., 20 for Sonnet 20).
	// Poems: scene=0 means "the whole poem" — match all scenes or NULL.
	const poemSceneFilter = scene === 0
		? '1=1'  // scene=0 → get all lines for the poem
		: '(tl.scene = ? OR tl.scene IS NULL)';

	const lineQuery = isPoem
		? `SELECT tl.id, tl.line_number, tl.content, tl.content_type,
              ${charCoalesce} AS character_name
       FROM text_lines tl
       LEFT JOIN characters c ON c.id = tl.character_id
       WHERE tl.work_id = ? AND tl.edition_id = ?
         AND ${poemSceneFilter}
         AND tl.id = (
           SELECT MIN(t2.id) FROM text_lines t2
           WHERE t2.work_id = tl.work_id AND t2.edition_id = tl.edition_id
             AND COALESCE(t2.act, 0) = COALESCE(tl.act, 0)
             AND COALESCE(t2.scene, 0) = COALESCE(tl.scene, 0)
             AND t2.line_number = tl.line_number
         )
       ORDER BY tl.line_number, tl.id`
		: `SELECT tl.id, tl.line_number, tl.content, tl.content_type,
              ${charCoalesce} AS character_name
       FROM text_lines tl
       LEFT JOIN characters c ON c.id = tl.character_id
       WHERE tl.work_id = ? AND tl.edition_id = ? AND tl.act = ? AND tl.scene = ?
         AND tl.id = (
           SELECT MIN(t2.id) FROM text_lines t2
           WHERE t2.work_id = tl.work_id AND t2.edition_id = tl.edition_id
             AND t2.act = tl.act AND t2.scene = tl.scene AND t2.line_number = tl.line_number
         )
       ORDER BY tl.line_number, tl.id`;

	let lines: SceneTextLine[];
	if (isPoem) {
		lines = scene === 0
			? db.prepare(lineQuery).all(workId, editionId) as SceneTextLine[]
			: db.prepare(lineQuery).all(workId, editionId, scene) as SceneTextLine[];
	} else {
		lines = db.prepare(lineQuery).all(workId, editionId, act, scene) as SceneTextLine[];
	}

	return { work_title: work.title, act, scene, edition_name: editionName, lines };
}

// ─── Attributions ─────────────────────────────────────────────────────────────

export interface FooterAttribution {
	source_name: string;
	attribution_html: string;
	license_notice_text: string | null;
	display_priority: number;
	required: boolean;
}

export function getFooterAttributions(db: Database.Database): FooterAttribution[] {
	return db
		.prepare(
			`SELECT s.name AS source_name,
			        a.attribution_html,
			        a.license_notice_text,
			        COALESCE(a.display_priority, 0) AS display_priority,
			        CASE WHEN a.required = 1 THEN 1 ELSE 0 END AS required
			 FROM attributions a
			 JOIN sources s ON s.id = a.source_id
			 WHERE a.display_format = 'footer'
			   AND (
			     s.id IN (
			       SELECT DISTINCT e.source_id FROM editions e
			       WHERE EXISTS (SELECT 1 FROM text_lines tl WHERE tl.edition_id = e.id)
			     )
			     OR (s.short_code = 'perseus_schmidt'
			         AND EXISTS (SELECT 1 FROM lexicon_entries LIMIT 1))
			   )
			 ORDER BY a.display_priority DESC, s.name`
		)
		.all() as FooterAttribution[];
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
