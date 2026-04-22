import { json, error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { CACHE_STATIC } from '$lib/server/cache';
import { getDb } from '$lib/server/db';

// Lookup lexicon entry by base_key string — stable across DB rebuilds.
// Used by the drawer instead of the numeric ID endpoint, which breaks when
// the pre-rendered page (built against local SQLite) is served against a
// Turso DB that has different auto-increment IDs for the same entries.
export const GET: RequestHandler = async ({ params, platform }) => {
	const key = params.key;
	if (!key) throw error(400, 'Missing key');

	try {
		const db = getDb(platform);

		const subEntriesResult = await db
			.prepare(
				`SELECT id, key, orthography, entry_type, full_text
				 FROM lexicon_entries WHERE base_key = ?
				 ORDER BY sense_group, id`
			)
			.bind(key)
			.all<{
				id: number;
				key: string;
				orthography: string | null;
				entry_type: string | null;
				full_text: string | null;
			}>();

		const rows = subEntriesResult.results ?? [];
		if (rows.length === 0) throw error(404, 'Entry not found');

		const idList = rows.map((r) => r.id);
		const ph = idList.map(() => '?').join(',');

		const sensesResult = await db
			.prepare(
				`SELECT id, entry_id, sense_number, sub_sense, definition_text
				 FROM lexicon_senses WHERE entry_id IN (${ph})
				 ORDER BY entry_id, sense_number, COALESCE(sub_sense, '')`
			)
			.bind(...idList)
			.all<{
				id: number;
				entry_id: number;
				sense_number: number;
				sub_sense: string | null;
				definition_text: string | null;
			}>();

		const sensesByEntry = new Map<number, (typeof sensesResult.results)[number][]>();
		for (const s of sensesResult.results ?? []) {
			if (!sensesByEntry.has(s.entry_id)) sensesByEntry.set(s.entry_id, []);
			sensesByEntry.get(s.entry_id)!.push(s);
		}

		const first = rows[0];
		const subEntries = rows.map((sub) => ({
			id: sub.id,
			key: sub.key,
			orthography: sub.orthography,
			entry_type: sub.entry_type,
			full_text: sub.full_text,
			senses: sensesByEntry.get(sub.id) ?? [],
			citations: []
		}));

		return json(
			{
				id: first.id,
				key,
				orthography: first.orthography,
				entry_type: first.entry_type,
				full_text: first.full_text,
				subEntries,
				senses: sensesByEntry.get(first.id) ?? [],
				citations: [],
				references: []
			},
			{ headers: { 'cache-control': CACHE_STATIC } }
		);
	} catch (err) {
		if (err && typeof err === 'object' && 'status' in err) throw err;
		console.error('[lexicon/key] query failed:', err);
		throw error(500, 'Query failed');
	}
};
