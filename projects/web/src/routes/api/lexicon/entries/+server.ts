import { json } from '@sveltejs/kit';
import { getDb } from '$lib/server/db';
import { getLexiconEntriesPage } from '$lib/server/queries';

export function GET({ url }) {
	const letter = url.searchParams.get('letter') ?? 'A';
	const offset = parseInt(url.searchParams.get('offset') ?? '0', 10);
	const limit = Math.min(parseInt(url.searchParams.get('limit') ?? '50', 10), 200);

	const db = getDb();
	const entries = getLexiconEntriesPage(db, letter.toUpperCase(), offset, limit);

	return json({ entries, hasMore: entries.length === limit });
}
