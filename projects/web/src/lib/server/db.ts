import { DatabaseSync } from 'node:sqlite';
import path from 'node:path';

let _db: DatabaseSync | null = null;

export function getDb(): DatabaseSync {
    if (_db) return _db;
    const dbPath = process.env.DB_PATH ?? path.join(process.cwd(), '../../build/bardbase.db');
    _db = new DatabaseSync(dbPath);
    _db.exec(`
        PRAGMA journal_mode=WAL;
        PRAGMA cache_size=-64000;
        PRAGMA mmap_size=268435456;
        PRAGMA foreign_keys=ON;
    `);
    return _db;
}
