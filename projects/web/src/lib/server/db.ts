/**
 * Build-time database access using better-sqlite3.
 * This module is only used during `vite build` (SSG) and dev server,
 * never shipped to the browser.
 *
 * In dev mode, the database connection is automatically refreshed when
 * the .db file changes (e.g. after `make capell run`), so you don't
 * need to restart the dev server.
 */
import Database from 'better-sqlite3';
import { resolve } from 'node:path';
import { statSync } from 'node:fs';
import { dev } from '$app/environment';

const DB_PATH = process.env.BARDBASE_DB_PATH ?? resolve(import.meta.dirname, '../../../../../build/bardbase.db');

let _db: Database.Database | null = null;
let _dbMtimeMs: number = 0;

export function getDb(): Database.Database {
	if (dev) {
		// In dev mode, check if the database file has been modified since
		// we last opened it. If so, close the old connection and reopen.
		try {
			const stat = statSync(DB_PATH);
			if (stat.mtimeMs !== _dbMtimeMs) {
				if (_db) {
					_db.close();
					_db = null;
				}
				_dbMtimeMs = stat.mtimeMs;
			}
		} catch {
			// File doesn't exist yet — will error on Database() below.
		}
	}

	if (!_db) {
		_db = new Database(DB_PATH, { readonly: true });
		_db.pragma('journal_mode = WAL');
		_db.pragma('cache_size = -64000');
		_db.pragma('mmap_size = 268435456');
	}
	return _db;
}
