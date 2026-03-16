#!/usr/bin/env node
// Introspects the built Shakespeare SQLite database and emits:
//   src/lib/generated/db.ts  — TypeScript interfaces + Zod schemas for every table
//
// Run:  node scripts/generate-types.js
// Also runs automatically via `npm run build` and `npm run dev`.

import Database from 'better-sqlite3';
import { writeFileSync, mkdirSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const DB_PATH = resolve(__dirname, '../../../build/shakespeare.db');
const OUT_PATH = resolve(__dirname, '../src/lib/generated/db.ts');

// ─── Type mapping helpers ────────────────────────────────────────────────────

/** Maps a SQLite declared type + nullability to a TypeScript type string. */
function toTSType(sqlType, notnull, isPK) {
	const nullable = !notnull && !isPK;
	const t = (sqlType || '').toUpperCase();

	let base;
	if (/\bBOOL/.test(t)) {
		base = 'number'; // SQLite stores BOOLEAN as 0 | 1
	} else if (/INT|NUMERIC|DECIMAL|REAL|FLOAT|DOUBLE/.test(t)) {
		base = 'number';
	} else if (/TIMESTAMP|DATETIME|DATE/.test(t)) {
		base = 'string';
	} else if (/BLOB/.test(t)) {
		base = 'Buffer';
	} else {
		// TEXT, VARCHAR, CHAR, CLOB, or unspecified affinity
		base = 'string';
	}

	return nullable ? `${base} | null` : base;
}

/** Maps a SQLite declared type + nullability to a Zod schema expression. */
function toZodExpr(sqlType, notnull, isPK) {
	const nullable = !notnull && !isPK;
	const t = (sqlType || '').toUpperCase();

	let base;
	if (/\bBOOL/.test(t)) {
		base = 'z.number()'; // 0 | 1
	} else if (/INT|NUMERIC|DECIMAL|REAL|FLOAT|DOUBLE/.test(t)) {
		base = 'z.number()';
	} else if (/TIMESTAMP|DATETIME|DATE/.test(t)) {
		base = 'z.string()';
	} else if (/BLOB/.test(t)) {
		base = 'z.instanceof(Buffer)';
	} else {
		base = 'z.string()';
	}

	return nullable ? `${base}.nullable()` : base;
}

/** snake_case → PascalCase */
function toPascal(str) {
	return str.replace(/(^|_)([a-z])/g, (_, _sep, c) => c.toUpperCase());
}

// ─── Introspect ──────────────────────────────────────────────────────────────

const db = new Database(DB_PATH, { readonly: true });

/** @type {Array<{name: string}>} */
const tables = db
	.prepare(
		`SELECT name FROM sqlite_master
     WHERE type = 'table'
       AND name NOT LIKE 'sqlite_%'
       AND name NOT LIKE '%_fts%'
     ORDER BY name`
	)
	.all();

// ─── Emit ────────────────────────────────────────────────────────────────────

const lines = [
	'// AUTO-GENERATED — do not edit manually.',
	'// Regenerate with: npm run generate:types',
	'// Source: sqlite schema introspected from build/shakespeare.db',
	'',
	"import { z } from 'zod';",
	''
];

for (const { name } of tables) {
	/** @type {Array<{cid:number, name:string, type:string, notnull:number, dflt_value:any, pk:number}>} */
	const cols = db.prepare(`PRAGMA table_info(${name})`).all();
	if (cols.length === 0) continue; // virtual table with no schema info

	const typeName = toPascal(name);

	// TypeScript interface
	lines.push(`export interface ${typeName} {`);
	for (const col of cols) {
		const tsType = toTSType(col.type, col.notnull, col.pk);
		lines.push(`\t${col.name}: ${tsType};`);
	}
	lines.push('}', '');

	// Zod schema
	lines.push(`export const ${typeName}Schema = z.object({`);
	for (const col of cols) {
		const zodExpr = toZodExpr(col.type, col.notnull, col.pk);
		lines.push(`\t${col.name}: ${zodExpr},`);
	}
	lines.push('});', '');

	// Inferred type alias for convenience (avoids re-typing Zod .infer everywhere)
	lines.push(`export type ${typeName}Row = z.infer<typeof ${typeName}Schema>;`, '');
}

db.close();

mkdirSync(dirname(OUT_PATH), { recursive: true });
writeFileSync(OUT_PATH, lines.join('\n'));
console.log(`generate-types: ${tables.length} tables → src/lib/generated/db.ts`);
