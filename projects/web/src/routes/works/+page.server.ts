import { getDb } from '$lib/server/db';
import type { Work } from '$lib/types';

export function load() {
	const db = getDb();

	const plays = db.prepare(`SELECT id, title, work_type, year FROM works WHERE work_type IN ('Comedy', 'Tragedy', 'History') ORDER BY title`).all() as Work[];
	const poetry = db.prepare(`SELECT id, title, work_type, year FROM works WHERE work_type NOT IN ('Comedy', 'Tragedy', 'History') ORDER BY title`).all() as Work[];

	return { plays, poetry };
}
