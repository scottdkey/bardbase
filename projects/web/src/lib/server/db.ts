/**
 * Build-time database access using better-sqlite3.
 * This module is only used during `vite build` (SSG) and dev server,
 * never shipped to the browser.
 */
import Database from 'better-sqlite3';
import { resolve } from 'node:path';

const DB_PATH = process.env.BARDBASE_DB_PATH ?? resolve(import.meta.dirname, '../../../../build/bardbase.db');

let _db: Database.Database | null = null;

export function getDb(): Database.Database {
	if (!_db) {
		_db = new Database(DB_PATH, { readonly: true });
		_db.pragma('journal_mode = WAL');
		_db.pragma('cache_size = -64000');
		_db.pragma('mmap_size = 268435456');
	}
	return _db;
}
