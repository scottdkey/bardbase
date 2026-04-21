import { json, error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { CACHE_STATIC } from '$lib/server/cache';
import { getDb } from '$lib/server/db';

export const GET: RequestHandler = async ({ params, platform }) => {
	const id = parseInt(params.id, 10);
	if (isNaN(id)) throw error(400, 'Invalid entry ID');

	try {
		const db = getDb(platform);
		// Load the entry and all sub-entries (same sense_group)
		const entryRow = await db
			.prepare(`SELECT id, key, base_key, orthography, entry_type, full_text FROM lexicon_entries WHERE id = ?`)
			.bind(id)
			.first<{ id: number; key: string; base_key: string; orthography: string | null; entry_type: string | null; full_text: string | null }>();

		if (!entryRow) throw error(404, 'Entry not found');

		// Fetch sub-entries sharing the same base_key (e.g. Bosom1, Bosom2 → base_key='Bosom')
		// Only expand when there are siblings (base_key differs from key = it's a numbered sub-entry)
		let idList: number[];
		let subEntriesResult: { results: { id: number; key: string; orthography: string | null; entry_type: string | null; full_text: string | null }[] };

		if (entryRow.base_key && entryRow.base_key !== entryRow.key) {
			subEntriesResult = await db
				.prepare(`SELECT id, key, orthography, entry_type, full_text FROM lexicon_entries WHERE base_key = ? ORDER BY sense_group, id`)
				.bind(entryRow.base_key)
				.all<{ id: number; key: string; orthography: string | null; entry_type: string | null; full_text: string | null }>();
			idList = (subEntriesResult.results ?? []).map((r) => r.id);
		} else {
			subEntriesResult = { results: [{ id: entryRow.id, key: entryRow.key, orthography: entryRow.orthography, entry_type: entryRow.entry_type, full_text: entryRow.full_text }] };
			idList = [entryRow.id];
		}

		// Fetch senses for all sub-entries
		const ph = idList.map(() => '?').join(',');
		const sensesResult = await db
			.prepare(`SELECT id, entry_id, sense_number, sub_sense, definition_text FROM lexicon_senses WHERE entry_id IN (${ph}) ORDER BY entry_id, sense_number`)
			.bind(...idList)
			.all<{ id: number; entry_id: number; sense_number: number; sub_sense: string | null; definition_text: string | null }>();

		// Fetch citations for all sub-entries
		const citationsResult = await db
			.prepare(`SELECT lc.id, lc.entry_id, lc.sense_id, lc.work_id, lc.work_abbrev,
				w.title AS work_title,
				lc.act, lc.scene, lc.line, lc.quote_text, lc.display_text, lc.raw_bibl
				FROM lexicon_citations lc
				LEFT JOIN works w ON w.id = lc.work_id
				WHERE lc.entry_id IN (${ph})
				ORDER BY lc.entry_id, lc.id`)
			.bind(...idList)
			.all<{ id: number; entry_id: number; sense_id: number | null; work_id: number | null; work_abbrev: string | null; work_title: string | null; act: number | null; scene: number | null; line: number | null; quote_text: string | null; display_text: string | null; raw_bibl: string | null }>();

		const sensesByEntry = new Map<number, typeof sensesResult.results>();
		for (const s of sensesResult.results ?? []) {
			if (!sensesByEntry.has(s.entry_id)) sensesByEntry.set(s.entry_id, []);
			sensesByEntry.get(s.entry_id)!.push(s);
		}

		const citationsByEntry = new Map<number, typeof citationsResult.results>();
		for (const c of citationsResult.results ?? []) {
			if (!citationsByEntry.has(c.entry_id)) citationsByEntry.set(c.entry_id, []);
			citationsByEntry.get(c.entry_id)!.push(c);
		}

		const subEntries = (subEntriesResult.results ?? []).map((sub) => ({
			id: sub.id,
			key: sub.key,
			orthography: sub.orthography,
			entry_type: sub.entry_type,
			full_text: sub.full_text,
			senses: sensesByEntry.get(sub.id) ?? [],
			citations: (citationsByEntry.get(sub.id) ?? []).map((c) => ({
				...c,
				matched_line: null,
				matched_line_number: null,
				matched_character: null,
				matched_edition_id: null,
				edition_lines: []
			}))
		}));

		const result = {
			id: entryRow.id,
			key: entryRow.key,
			orthography: entryRow.orthography,
			entry_type: entryRow.entry_type,
			full_text: entryRow.full_text,
			subEntries,
			senses: sensesByEntry.get(entryRow.id) ?? [],
			citations: (citationsByEntry.get(entryRow.id) ?? []).map((c) => ({
				...c,
				matched_line: null,
				matched_line_number: null,
				matched_character: null,
				matched_edition_id: null,
				edition_lines: []
			})),
			references: []
		};

		return json(result, { headers: { 'cache-control': CACHE_STATIC } });
	} catch (err) {
		if (err && typeof err === 'object' && 'status' in err) throw err;
		console.error('[lexicon/entry] query failed:', err);
		throw error(500, 'Query failed');
	}
};
