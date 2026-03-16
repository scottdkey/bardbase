/**
 * Client-side SQLite via sql.js (WASM).
 * Used for offline full-text search after the DB is cached by the service worker.
 */
import type { Database as SqlJsDatabase } from 'sql.js';

let db: SqlJsDatabase | null = null;
let loading: Promise<SqlJsDatabase> | null = null;

/**
 * Lazily load the search database. The .db file is cached by the
 * service worker for offline access.
 */
export async function getSearchDb(): Promise<SqlJsDatabase> {
	if (db) return db;
	if (loading) return loading;

	loading = (async () => {
		const initSqlJs = (await import('sql.js')).default;
		const SQL = await initSqlJs({
			locateFile: (file: string) => `/wasm/${file}`
		});

		const response = await fetch('/shakespeare-search.db');
		const buffer = await response.arrayBuffer();
		db = new SQL.Database(new Uint8Array(buffer));
		return db;
	})();

	return loading;
}

export interface SearchResult {
	work_title: string;
	act: number;
	scene: number;
	line_number: number;
	content: string;
	character_name: string;
}

/**
 * Full-text search across all text lines.
 */
export async function searchText(query: string, limit = 50): Promise<SearchResult[]> {
	const database = await getSearchDb();
	const stmt = database.prepare(`
		SELECT
			w.title AS work_title,
			tl.act,
			tl.scene,
			tl.line_number,
			tl.content,
			COALESCE(c.name, '') AS character_name
		FROM text_fts fts
		JOIN text_lines tl ON tl.rowid = fts.rowid
		JOIN works w ON w.id = tl.work_id
		LEFT JOIN characters c ON c.id = tl.character_id
		WHERE text_fts MATCH :query
		ORDER BY rank
		LIMIT :limit
	`);
	stmt.bind({ ':query': query, ':limit': limit });

	const results: SearchResult[] = [];
	while (stmt.step()) {
		const row = stmt.getAsObject() as unknown as SearchResult;
		results.push(row);
	}
	stmt.free();
	return results;
}
