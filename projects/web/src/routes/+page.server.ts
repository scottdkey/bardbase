import { getDb } from '$lib/server/db';
import type { Work } from '$lib/types';

export function load() {
	const db = getDb();
	const works = db.prepare(`
		SELECT id, title, long_title, work_type, year
		FROM works
		ORDER BY title
	`).all() as Work[];

	const stats = db.prepare(`
		SELECT
			(SELECT COUNT(*) FROM works) AS work_count,
			(SELECT COUNT(*) FROM characters) AS character_count,
			(SELECT COUNT(*) FROM text_lines) AS line_count,
			(SELECT COUNT(*) FROM lexicon_entries) AS lexicon_count
	`).get() as { work_count: number; character_count: number; line_count: number; lexicon_count: number };

	return { works, stats };
}
