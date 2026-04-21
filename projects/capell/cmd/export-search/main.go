// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

// Command export-search builds a Turso-compatible SQL dump for the bardbase
// production database. It reads from the full bardbase.db and writes a SQL
// file that `sqlite3` can load into a slim runtime .db, which `turso db import`
// then uploads as the production DB. The SQL file also happens to be
// D1-compatible for fallback imports via `wrangler d1 execute --file`.
//
// Pipeline:
//   cmd/build  → bardbase.db           (full, intermediate)
//   export-search → bardbase-search.sql (Turso/D1-compatible dump)
//   sqlite3 < .sql → bardbase-search.db (canonical runtime artifact)
//   turso db import bardbase-search.db (production)
//
// The SQL file contains:
//
//   - All core app tables (works, sources, editions, attributions, characters,
//     text_divisions, text_lines, line_mappings, lexicon_*, citation_*,
//     reference_*) — the web app queries these directly at runtime.
//   - Slim FTS5 virtual tables (text_fts, lexicon_fts, reference_fts) whose
//     rowids match the primary keys of their backing tables. UNINDEXED
//     metadata has been removed; display metadata is fetched via rowid join.
//   - Derived tables (reference_spans, lexicon_drawer) with pre-computed
//     data for app features that need it at request time.
//
// All FTS tables use the trigram tokenizer (D1-compatible). Data is inserted
// in multi-row batched transactions bounded by both row count and byte size
// to stay within D1's per-query limits.
//
// Usage:
//
//	go run ./cmd/export-search
//	go run ./cmd/export-search -src ../../build/bardbase.db -out ../../build/bardbase-search.sql
package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"unicode"

	"github.com/scottdkey/bardbase/projects/capell/internal/constants"
	"github.com/scottdkey/bardbase/projects/capell/internal/parser"
	_ "modernc.org/sqlite"
)

// Per-statement byte cap for multi-row INSERTs. D1's hard cap is ~100KB.
const maxStmtBytes = 80_000

// Max rows per multi-row INSERT before forcing a flush.
const maxBatchRows = 500

func main() {
	srcPath := flag.String("src", "../../build/bardbase.db", "Source database path")
	outPath := flag.String("out", "../../build/bardbase-search.sql", "Output SQL file path")
	flag.Parse()

	if _, err := os.Stat(*srcPath); err != nil {
		log.Fatalf("source database not found: %s", *srcPath)
	}

	src, err := openReadOnly(*srcPath)
	if err != nil {
		log.Fatalf("open source db: %v", err)
	}
	defer src.Close()

	f, err := os.Create(*outPath)
	if err != nil {
		log.Fatalf("create output file: %v", err)
	}
	defer f.Close()

	w := bufio.NewWriterSize(f, 4*1024*1024)

	log.Printf("exporting database to %s", *outPath)

	writeHeader(w)
	writeSchema(w)

	// Core app tables — order matters for readability only (FKs are off).
	if err := writeCoreTables(w, src); err != nil {
		log.Fatalf("core tables: %v", err)
	}

	// FTS indexes — rowids line up with text_lines.id / reference_entries.id / lexicon_entries.id.
	if err := writeTextFTS(w, src); err != nil {
		log.Fatalf("text_fts: %v", err)
	}
	if err := writeLexiconFTS(w, src); err != nil {
		log.Fatalf("lexicon_fts: %v", err)
	}
	if err := writeReferenceFTS(w, src); err != nil {
		log.Fatalf("reference_fts: %v", err)
	}
	if err := writeReferenceSpans(w, src); err != nil {
		log.Fatalf("reference_spans: %v", err)
	}
	if err := writeLexiconDrawer(w, src); err != nil {
		log.Fatalf("lexicon_drawer: %v", err)
	}

	writeFooter(w)

	if err := w.Flush(); err != nil {
		log.Fatalf("flush: %v", err)
	}

	info, _ := f.Stat()
	log.Printf("done — %.1f MB written to %s", float64(info.Size())/1024/1024, *outPath)
}

func writeHeader(w *bufio.Writer) {
	// Turso's `--from-dump` validates the header verbatim — it requires
	// exactly these first two lines and a matching `COMMIT;` at the end
	// (same shape as `sqlite3 .dump`). Any comments or extra whitespace
	// here will be rejected.
	fmt.Fprint(w, "PRAGMA foreign_keys=OFF;\nBEGIN TRANSACTION;\n")
}

func writeFooter(w *bufio.Writer) {
	fmt.Fprint(w, "COMMIT;\n")
}

// ── schema ────────────────────────────────────────────────────────────────────

func writeSchema(w *bufio.Writer) {
	fmt.Fprint(w, `-- Drop everything (order respects FK references when enforced).
DROP TABLE IF EXISTS reference_citation_matches;
DROP TABLE IF EXISTS reference_citations;
DROP TABLE IF EXISTS reference_entries;
DROP TABLE IF EXISTS citation_matches;
DROP TABLE IF EXISTS lexicon_citations;
DROP TABLE IF EXISTS lexicon_senses;
DROP TABLE IF EXISTS lexicon_entries;
DROP TABLE IF EXISTS line_mappings;
DROP TABLE IF EXISTS text_lines;
DROP TABLE IF EXISTS text_divisions;
DROP TABLE IF EXISTS characters;
DROP TABLE IF EXISTS attributions;
DROP TABLE IF EXISTS editions;
DROP TABLE IF EXISTS sources;
DROP TABLE IF EXISTS works;
DROP TABLE IF EXISTS text_fts;
DROP TABLE IF EXISTS lexicon_fts;
DROP TABLE IF EXISTS reference_fts;
DROP TABLE IF EXISTS reference_spans;
DROP TABLE IF EXISTS lexicon_drawer;

-- ── core app tables ─────────────────────────────────────────────────────────
CREATE TABLE works (
  id INTEGER PRIMARY KEY,
  oss_id TEXT,
  title TEXT NOT NULL,
  full_title TEXT,
  short_title TEXT,
  schmidt_abbrev TEXT,
  work_type TEXT,
  date_composed INTEGER,
  genre_type TEXT,
  total_words INTEGER,
  total_paragraphs INTEGER,
  source_text TEXT,
  folger_url TEXT,
  perseus_id TEXT,
  notes TEXT
);
CREATE INDEX idx_works_type ON works(work_type);

CREATE TABLE sources (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  short_code TEXT NOT NULL,
  url TEXT,
  license TEXT,
  license_url TEXT,
  attribution_text TEXT,
  attribution_required INTEGER DEFAULT 0,
  notes TEXT
);

CREATE TABLE editions (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  short_code TEXT NOT NULL,
  source_id INTEGER,
  year INTEGER,
  editors TEXT,
  description TEXT,
  notes TEXT,
  source_key TEXT,
  license_tier TEXT
);

CREATE TABLE attributions (
  id INTEGER PRIMARY KEY,
  source_id INTEGER NOT NULL,
  required INTEGER DEFAULT 0,
  attribution_text TEXT NOT NULL,
  attribution_html TEXT,
  display_format TEXT DEFAULT 'footer',
  display_context TEXT DEFAULT 'always',
  display_priority INTEGER DEFAULT 0,
  requires_link_back INTEGER DEFAULT 0,
  link_back_url TEXT,
  requires_license_notice INTEGER DEFAULT 0,
  license_notice_text TEXT,
  requires_author_credit INTEGER DEFAULT 0,
  author_credit_text TEXT,
  share_alike_required INTEGER DEFAULT 0,
  commercial_use_allowed INTEGER DEFAULT 1,
  modification_allowed INTEGER DEFAULT 1,
  notes TEXT
);

CREATE TABLE characters (
  id INTEGER PRIMARY KEY,
  char_id TEXT,
  name TEXT NOT NULL,
  abbrev TEXT,
  work_id INTEGER,
  oss_work_id TEXT,
  description TEXT,
  speech_count INTEGER
);
CREATE INDEX idx_characters_work ON characters(work_id);

CREATE TABLE text_divisions (
  id INTEGER PRIMARY KEY,
  work_id INTEGER NOT NULL,
  edition_id INTEGER NOT NULL,
  act INTEGER NOT NULL,
  scene INTEGER NOT NULL,
  description TEXT,
  line_count INTEGER DEFAULT 0
);
CREATE INDEX idx_td_work_ed ON text_divisions(work_id, edition_id);

CREATE TABLE text_lines (
  id INTEGER PRIMARY KEY,
  work_id INTEGER NOT NULL,
  edition_id INTEGER NOT NULL,
  act INTEGER,
  scene INTEGER,
  paragraph_num INTEGER,
  line_number INTEGER,
  character_id INTEGER,
  char_name TEXT,
  content TEXT NOT NULL,
  content_type TEXT DEFAULT 'speech',
  word_count INTEGER DEFAULT 0,
  oss_paragraph_id INTEGER,
  sonnet_number INTEGER,
  stanza INTEGER,
  line_type TEXT,
  stage_type TEXT,
  stage_who TEXT
);
CREATE INDEX idx_text_work_edition ON text_lines(work_id, edition_id);
CREATE INDEX idx_text_location ON text_lines(work_id, act, scene);
CREATE INDEX idx_text_line_number ON text_lines(work_id, edition_id, act, scene, line_number);

CREATE TABLE line_mappings (
  id INTEGER PRIMARY KEY,
  work_id INTEGER NOT NULL,
  act INTEGER NOT NULL,
  scene INTEGER NOT NULL,
  align_order INTEGER NOT NULL,
  edition_a_id INTEGER NOT NULL,
  edition_b_id INTEGER NOT NULL,
  line_a_id INTEGER,
  line_b_id INTEGER,
  match_type TEXT DEFAULT 'aligned',
  similarity REAL DEFAULT 0.0
);
CREATE INDEX idx_line_mappings_scene ON line_mappings(work_id, act, scene);
CREATE INDEX idx_line_mappings_line_a ON line_mappings(line_a_id);
CREATE INDEX idx_line_mappings_line_b ON line_mappings(line_b_id);

CREATE TABLE lexicon_entries (
  id INTEGER PRIMARY KEY,
  key TEXT NOT NULL,
  base_key TEXT NOT NULL,
  sense_group INTEGER,
  letter TEXT NOT NULL,
  orthography TEXT,
  entry_type TEXT DEFAULT 'main',
  full_text TEXT,
  source_file TEXT
);
CREATE INDEX idx_lexicon_base_key ON lexicon_entries(base_key);
CREATE INDEX idx_lexicon_letter ON lexicon_entries(letter);

CREATE TABLE lexicon_senses (
  id INTEGER PRIMARY KEY,
  entry_id INTEGER NOT NULL,
  sense_number INTEGER NOT NULL,
  sub_sense TEXT,
  definition_text TEXT
);
CREATE INDEX idx_senses_entry ON lexicon_senses(entry_id);

CREATE TABLE lexicon_citations (
  id INTEGER PRIMARY KEY,
  entry_id INTEGER NOT NULL,
  sense_id INTEGER,
  work_id INTEGER,
  work_abbrev TEXT,
  perseus_ref TEXT,
  act INTEGER,
  scene INTEGER,
  line INTEGER,
  quote_text TEXT,
  display_text TEXT,
  raw_bibl TEXT
);
CREATE INDEX idx_citations_entry ON lexicon_citations(entry_id);
CREATE INDEX idx_citations_work ON lexicon_citations(work_id);

CREATE TABLE citation_matches (
  id INTEGER PRIMARY KEY,
  citation_id INTEGER NOT NULL,
  text_line_id INTEGER NOT NULL,
  edition_id INTEGER NOT NULL,
  match_type TEXT DEFAULT 'exact',
  confidence REAL DEFAULT 1.0,
  matched_text TEXT,
  notes TEXT
);
CREATE INDEX idx_cm_citation ON citation_matches(citation_id);
CREATE INDEX idx_cm_line ON citation_matches(text_line_id);

CREATE TABLE reference_entries (
  id INTEGER PRIMARY KEY,
  source_id INTEGER NOT NULL,
  headword TEXT NOT NULL,
  letter TEXT NOT NULL,
  raw_text TEXT NOT NULL
);
CREATE INDEX idx_ref_entries_source ON reference_entries(source_id);
CREATE INDEX idx_ref_entries_headword ON reference_entries(headword);

CREATE TABLE reference_citations (
  id INTEGER PRIMARY KEY,
  entry_id INTEGER NOT NULL,
  source_id INTEGER NOT NULL,
  work_id INTEGER,
  work_abbrev TEXT NOT NULL,
  act INTEGER,
  scene INTEGER,
  line INTEGER
);
CREATE INDEX idx_rc_entry ON reference_citations(entry_id);
CREATE INDEX idx_rc_work ON reference_citations(work_id);

CREATE TABLE reference_citation_matches (
  id INTEGER PRIMARY KEY,
  ref_citation_id INTEGER NOT NULL,
  text_line_id INTEGER NOT NULL,
  edition_id INTEGER NOT NULL,
  match_type TEXT DEFAULT 'line_number',
  confidence REAL DEFAULT 1.0,
  matched_text TEXT
);
CREATE INDEX idx_rcm_cit ON reference_citation_matches(ref_citation_id);
CREATE INDEX idx_rcm_line ON reference_citation_matches(text_line_id);

-- ── FTS5 indexes (trigram; rowid = backing table PK) ────────────────────────

-- text_fts rowid matches text_lines.id. Search content + char_name only;
-- display metadata is read via rowid join against text_lines.
CREATE VIRTUAL TABLE text_fts USING fts5(
  content,
  char_name,
  tokenize='trigram'
);

-- lexicon_fts rowid matches lexicon_entries.id.
CREATE VIRTUAL TABLE lexicon_fts USING fts5(
  key,
  orthography,
  full_text,
  letter,
  tokenize='trigram'
);

-- reference_fts rowid matches reference_entries.id. Search headword + raw_text only;
-- source metadata is read via rowid join against reference_entries + sources.
CREATE VIRTUAL TABLE reference_fts USING fts5(
  headword,
  raw_text,
  tokenize='trigram'
);

-- ── derived/runtime tables ──────────────────────────────────────────────────

CREATE TABLE reference_spans (
  id INTEGER PRIMARY KEY,
  citation_spans TEXT NOT NULL,
  citations TEXT NOT NULL
);

CREATE TABLE lexicon_drawer (
  id INTEGER PRIMARY KEY,
  data TEXT NOT NULL
);

`)
}

// ── core table exporter ──────────────────────────────────────────────────────

// tableSpec describes one core table to copy verbatim.
type tableSpec struct {
	name string
	cols []string
}

// coreTables lists every app table to export, in FK-safe order. Column lists
// mirror the CREATE TABLE statements emitted by writeSchema.
var coreTables = []tableSpec{
	{"works", []string{"id", "oss_id", "title", "full_title", "short_title", "schmidt_abbrev", "work_type", "date_composed", "genre_type", "total_words", "total_paragraphs", "source_text", "folger_url", "perseus_id", "notes"}},
	{"sources", []string{"id", "name", "short_code", "url", "license", "license_url", "attribution_text", "attribution_required", "notes"}},
	{"editions", []string{"id", "name", "short_code", "source_id", "year", "editors", "description", "notes", "source_key", "license_tier"}},
	{"attributions", []string{"id", "source_id", "required", "attribution_text", "attribution_html", "display_format", "display_context", "display_priority", "requires_link_back", "link_back_url", "requires_license_notice", "license_notice_text", "requires_author_credit", "author_credit_text", "share_alike_required", "commercial_use_allowed", "modification_allowed", "notes"}},
	{"characters", []string{"id", "char_id", "name", "abbrev", "work_id", "oss_work_id", "description", "speech_count"}},
	{"text_divisions", []string{"id", "work_id", "edition_id", "act", "scene", "description", "line_count"}},
	{"text_lines", []string{"id", "work_id", "edition_id", "act", "scene", "paragraph_num", "line_number", "character_id", "char_name", "content", "content_type", "word_count", "oss_paragraph_id", "sonnet_number", "stanza", "line_type", "stage_type", "stage_who"}},
	{"line_mappings", []string{"id", "work_id", "act", "scene", "align_order", "edition_a_id", "edition_b_id", "line_a_id", "line_b_id", "match_type", "similarity"}},
	{"lexicon_entries", []string{"id", "key", "base_key", "sense_group", "letter", "orthography", "entry_type", "full_text", "source_file"}},
	{"lexicon_senses", []string{"id", "entry_id", "sense_number", "sub_sense", "definition_text"}},
	{"lexicon_citations", []string{"id", "entry_id", "sense_id", "work_id", "work_abbrev", "perseus_ref", "act", "scene", "line", "quote_text", "display_text", "raw_bibl"}},
	{"citation_matches", []string{"id", "citation_id", "text_line_id", "edition_id", "match_type", "confidence", "matched_text", "notes"}},
	{"reference_entries", []string{"id", "source_id", "headword", "letter", "raw_text"}},
	{"reference_citations", []string{"id", "entry_id", "source_id", "work_id", "work_abbrev", "act", "scene", "line"}},
	{"reference_citation_matches", []string{"id", "ref_citation_id", "text_line_id", "edition_id", "match_type", "confidence", "matched_text"}},
}

func writeCoreTables(w *bufio.Writer, src *sql.DB) error {
	for _, t := range coreTables {
		if err := writeCoreTable(w, src, t); err != nil {
			return fmt.Errorf("%s: %w", t.name, err)
		}
	}
	return nil
}

func writeCoreTable(w *bufio.Writer, src *sql.DB, t tableSpec) error {
	log.Printf("writing %s...", t.name)

	query := fmt.Sprintf("SELECT %s FROM %s ORDER BY id", strings.Join(t.cols, ","), t.name)
	rows, err := src.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Reusable scan targets — all columns scanned as *any.
	scanVals := make([]any, len(t.cols))
	scanPtrs := make([]any, len(t.cols))
	for i := range scanVals {
		scanPtrs[i] = &scanVals[i]
	}

	insertPrefix := fmt.Sprintf("INSERT INTO %s(%s) VALUES ", t.name, strings.Join(t.cols, ","))
	var buf strings.Builder
	buf.Grow(maxStmtBytes + 4096)
	rowCount := 0
	total := 0

	flush := func() {
		if rowCount == 0 {
			return
		}
		w.WriteString(insertPrefix)
		w.WriteString(buf.String())
		w.WriteString(";\n")
		buf.Reset()
		rowCount = 0
	}

	for rows.Next() {
		if err := rows.Scan(scanPtrs...); err != nil {
			return err
		}

		var rowBuf strings.Builder
		rowBuf.WriteByte('(')
		for i, v := range scanVals {
			if i > 0 {
				rowBuf.WriteByte(',')
			}
			rowBuf.WriteString(sqlLiteral(v))
		}
		rowBuf.WriteByte(')')

		projected := len(insertPrefix) + buf.Len() + rowBuf.Len() + 3 // ",;\n"
		if rowCount > 0 && (projected > maxStmtBytes || rowCount >= maxBatchRows) {
			flush()
		}
		if rowCount > 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(rowBuf.String())
		rowCount++
		total++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	flush()

	log.Printf("  wrote %d %s rows", total, t.name)
	return nil
}

// sqlLiteral formats any scanned value as a SQL literal. []byte is treated as
// UTF-8 TEXT (modernc.org/sqlite returns TEXT columns as []byte by default).
func sqlLiteral(v any) string {
	switch x := v.(type) {
	case nil:
		return "NULL"
	case int64:
		return fmt.Sprintf("%d", x)
	case float64:
		return fmt.Sprintf("%g", x)
	case bool:
		if x {
			return "1"
		}
		return "0"
	case string:
		return sqlStr(x)
	case []byte:
		return sqlStr(string(x))
	default:
		return sqlStr(fmt.Sprintf("%v", x))
	}
}

// ── FTS writers ──────────────────────────────────────────────────────────────

func writeTextFTS(w *bufio.Writer, src *sql.DB) error {
	rows, err := src.Query(`
		SELECT id, content, COALESCE(char_name,'')
		FROM text_lines
		ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	log.Print("writing text_fts...")
	count := 0
	for rows.Next() {
		var id int
		var content, charName string
		if err := rows.Scan(&id, &content, &charName); err != nil {
			return err
		}
		fmt.Fprintf(w, "INSERT INTO text_fts(rowid,content,char_name) VALUES(%d,%s,%s);\n",
			id, sqlStr(content), sqlStr(charName))
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	log.Printf("  wrote %d text_fts rows", count)
	return nil
}

func writeLexiconFTS(w *bufio.Writer, src *sql.DB) error {
	rows, err := src.Query(`
		SELECT id, key, COALESCE(orthography,''), COALESCE(full_text,''), COALESCE(letter,'')
		FROM lexicon_entries
		ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	log.Print("writing lexicon_fts...")
	count := 0
	for rows.Next() {
		var id int
		var key, orthography, fullText, letter string
		if err := rows.Scan(&id, &key, &orthography, &fullText, &letter); err != nil {
			return err
		}
		fmt.Fprintf(w, "INSERT INTO lexicon_fts(rowid,key,orthography,full_text,letter) VALUES(%d,%s,%s,%s,%s);\n",
			id, sqlStr(key), sqlStr(orthography), sqlStr(fullText), sqlStr(letter))
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	log.Printf("  wrote %d lexicon_fts rows", count)
	return nil
}

func writeReferenceFTS(w *bufio.Writer, src *sql.DB) error {
	rows, err := src.Query(`
		SELECT id, headword, COALESCE(raw_text,'')
		FROM reference_entries
		ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	log.Print("writing reference_fts...")
	count := 0
	for rows.Next() {
		var id int
		var headword, rawText string
		if err := rows.Scan(&id, &headword, &rawText); err != nil {
			return err
		}
		fmt.Fprintf(w, "INSERT INTO reference_fts(rowid,headword,raw_text) VALUES(%d,%s,%s);\n",
			id, sqlStr(headword), sqlStr(rawText))
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	log.Printf("  wrote %d reference_fts rows", count)
	return nil
}

// sqlStr returns a SQL string literal with single-quote escaping.
func sqlStr(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// slugify converts a title like "Henry IV, Part I" to "henry-iv-part-i".
// Must stay in sync with the slugify function in internal/api/meta.go.
func slugify(title string) string {
	var b strings.Builder
	lastDash := true
	for _, r := range strings.ToLower(title) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
		} else if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}

// ── reference_spans ──────────────────────────────────────────────────────────

type spanJSON struct {
	Start    int     `json:"start"`
	End      int     `json:"end"`
	WorkSlug *string `json:"work_slug,omitempty"`
	Act      *int    `json:"act,omitempty"`
	Scene    *int    `json:"scene,omitempty"`
	Line     *int    `json:"line,omitempty"`
}

type refCitJSON struct {
	WorkTitle *string `json:"work_title"`
	Act       *int    `json:"act"`
	Scene     *int    `json:"scene"`
	Line      *int    `json:"line"`
	WorkSlug  *string `json:"work_slug"`
}

func writeReferenceSpans(w *bufio.Writer, src *sql.DB) error {
	// Build citations map: entry_id → []refCitJSON
	citMap := map[int][]refCitJSON{}
	citRows, err := src.Query(`
		SELECT rc.entry_id, w.title, rc.act, rc.scene, rc.line
		FROM reference_citations rc
		LEFT JOIN works w ON w.id = rc.work_id
		ORDER BY rc.entry_id, w.title, rc.act, rc.scene, rc.line`)
	if err != nil {
		return fmt.Errorf("reference_citations: %w", err)
	}
	defer citRows.Close()
	for citRows.Next() {
		var entryID int
		var c refCitJSON
		if err := citRows.Scan(&entryID, &c.WorkTitle, &c.Act, &c.Scene, &c.Line); err != nil {
			return err
		}
		if c.WorkTitle != nil {
			s := slugify(*c.WorkTitle)
			c.WorkSlug = &s
		}
		citMap[entryID] = append(citMap[entryID], c)
	}
	if err := citRows.Err(); err != nil {
		return err
	}

	rows, err := src.Query(`
		SELECT re.id, re.raw_text, s.short_code
		FROM reference_entries re
		JOIN sources s ON s.id = re.source_id
		ORDER BY re.id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	log.Print("writing reference_spans...")
	count := 0
	for rows.Next() {
		var id int
		var rawText, sourceCode string
		if err := rows.Scan(&id, &rawText, &sourceCode); err != nil {
			return err
		}

		spans := parser.LocateCitationSpans(sourceCode, rawText)
		abbrevMap := abbrevMapForSource(sourceCode)
		spansList := make([]spanJSON, 0, len(spans))
		for _, sp := range spans {
			sj := spanJSON{Start: sp.Start, End: sp.End, Act: sp.Act, Scene: sp.Scene, Line: sp.Line}
			if slug := resolveAbbrevToSlug(sp.WorkAbbrev, abbrevMap); slug != "" {
				s := slug
				sj.WorkSlug = &s
			}
			spansList = append(spansList, sj)
		}
		sort.Slice(spansList, func(i, j int) bool { return spansList[i].Start < spansList[j].Start })

		spansBytes, _ := json.Marshal(spansList)

		cits := citMap[id]
		if cits == nil {
			cits = []refCitJSON{}
		}
		citsBytes, _ := json.Marshal(cits)

		fmt.Fprintf(w, "INSERT INTO reference_spans(id,citation_spans,citations) VALUES(%d,%s,%s);\n",
			id, sqlStr(string(spansBytes)), sqlStr(string(citsBytes)))
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	log.Printf("  wrote %d reference_spans rows", count)
	return nil
}

func abbrevMapForSource(sourceCode string) map[string]string {
	switch sourceCode {
	case "abbott":
		return constants.AbbottAbbrevs
	case "onions":
		return constants.OnionsAbbrevs
	case "bartlett":
		return constants.BartlettAbbrevs
	case "henley_farmer":
		return constants.HenleyFarmerAbbrevs
	default:
		return nil
	}
}

func resolveAbbrevToSlug(abbrev string, abbrevMap map[string]string) string {
	schmidtAbbrev := abbrev
	if abbrevMap != nil {
		if mapped, ok := abbrevMap[abbrev]; ok {
			schmidtAbbrev = mapped
		}
	}
	if work, ok := constants.SchmidtWorks[schmidtAbbrev]; ok {
		return slugify(work.Title)
	}
	return ""
}

// ── lexicon_drawer ────────────────────────────────────────────────────────────

type drawerSenseJSON struct {
	ID             int     `json:"id"`
	EntryID        int     `json:"entry_id"`
	SenseNumber    int     `json:"sense_number"`
	SubSense       *string `json:"sub_sense"`
	DefinitionText *string `json:"definition_text"`
}

type drawerSubEntryJSON struct {
	ID          int               `json:"id"`
	Key         string            `json:"key"`
	EntryType   *string           `json:"entry_type"`
	FullText    *string           `json:"full_text"`
	Orthography *string           `json:"orthography"`
	Senses      []drawerSenseJSON `json:"senses"`
	Citations   []struct{}        `json:"citations"`
}

type drawerRefJSON struct {
	SourceName    string `json:"source_name"`
	SourceCode    string `json:"source_code"`
	EntryID       int    `json:"entry_id"`
	EntryHeadword string `json:"entry_headword"`
}

type lexiconDrawerJSON struct {
	ID          int                  `json:"id"`
	Key         string               `json:"key"`
	Orthography *string              `json:"orthography"`
	EntryType   *string              `json:"entry_type"`
	FullText    *string              `json:"full_text"`
	SubEntries  []drawerSubEntryJSON `json:"subEntries"`
	Senses      []drawerSenseJSON    `json:"senses"`
	Citations   []struct{}           `json:"citations"`
	References  []drawerRefJSON      `json:"references"`
}

func writeLexiconDrawer(w *bufio.Writer, src *sql.DB) error {
	sensesMap := map[int][]drawerSenseJSON{}
	senseRows, err := src.Query(
		`SELECT id, entry_id, sense_number, sub_sense, definition_text
		 FROM lexicon_senses
		 ORDER BY entry_id, sense_number, COALESCE(sub_sense,'')`)
	if err != nil {
		return fmt.Errorf("lexicon_senses: %w", err)
	}
	defer senseRows.Close()
	for senseRows.Next() {
		var s drawerSenseJSON
		if err := senseRows.Scan(&s.ID, &s.EntryID, &s.SenseNumber, &s.SubSense, &s.DefinitionText); err != nil {
			return err
		}
		sensesMap[s.EntryID] = append(sensesMap[s.EntryID], s)
	}
	if err := senseRows.Err(); err != nil {
		return err
	}

	refsMap := map[string][]drawerRefJSON{}
	refRows, err := src.Query(`
		SELECT LOWER(re.headword), src.name, src.short_code, re.id, re.headword
		FROM reference_entries re
		JOIN sources src ON src.id = re.source_id
		ORDER BY LOWER(re.headword), src.name, re.id`)
	if err != nil {
		return fmt.Errorf("reference_entries for drawer: %w", err)
	}
	defer refRows.Close()
	for refRows.Next() {
		var key string
		var r drawerRefJSON
		if err := refRows.Scan(&key, &r.SourceName, &r.SourceCode, &r.EntryID, &r.EntryHeadword); err != nil {
			return err
		}
		refsMap[key] = append(refsMap[key], r)
	}
	if err := refRows.Err(); err != nil {
		return err
	}

	type entryInfo struct {
		id          int
		key         string
		baseKey     string
		orthography *string
		entryType   *string
		fullText    *string
	}
	var allEntries []entryInfo
	entryRows, err := src.Query(
		`SELECT id, key, base_key, orthography, entry_type, full_text
		 FROM lexicon_entries
		 ORDER BY base_key, sense_group, id`)
	if err != nil {
		return fmt.Errorf("lexicon_entries: %w", err)
	}
	defer entryRows.Close()
	for entryRows.Next() {
		var e entryInfo
		if err := entryRows.Scan(&e.id, &e.key, &e.baseKey, &e.orthography, &e.entryType, &e.fullText); err != nil {
			return err
		}
		allEntries = append(allEntries, e)
	}
	if err := entryRows.Err(); err != nil {
		return err
	}

	type groupData struct {
		baseKey string
		entries []entryInfo
	}
	var groups []groupData
	groupIdx := map[string]int{}
	for _, e := range allEntries {
		if i, ok := groupIdx[e.baseKey]; ok {
			groups[i].entries = append(groups[i].entries, e)
		} else {
			groupIdx[e.baseKey] = len(groups)
			groups = append(groups, groupData{baseKey: e.baseKey, entries: []entryInfo{e}})
		}
	}

	log.Print("writing lexicon_drawer...")
	count := 0
	for _, g := range groups {
		subEntries := make([]drawerSubEntryJSON, len(g.entries))
		var allSenses []drawerSenseJSON
		for i, e := range g.entries {
			s := sensesMap[e.id]
			if s == nil {
				s = []drawerSenseJSON{}
			}
			allSenses = append(allSenses, s...)
			subEntries[i] = drawerSubEntryJSON{
				ID:          e.id,
				Key:         e.key,
				EntryType:   e.entryType,
				FullText:    e.fullText,
				Orthography: e.orthography,
				Senses:      s,
				Citations:   []struct{}{},
			}
		}
		if allSenses == nil {
			allSenses = []drawerSenseJSON{}
		}

		entryRefs := refsMap[strings.ToLower(g.baseKey)]
		if entryRefs == nil {
			entryRefs = []drawerRefJSON{}
		}

		primary := g.entries[0]
		drawer := lexiconDrawerJSON{
			ID:          primary.id,
			Key:         g.baseKey,
			Orthography: primary.orthography,
			EntryType:   primary.entryType,
			FullText:    primary.fullText,
			SubEntries:  subEntries,
			Senses:      allSenses,
			Citations:   []struct{}{},
			References:  entryRefs,
		}
		data, err := json.Marshal(drawer)
		if err != nil {
			return fmt.Errorf("marshal lexicon drawer %d: %w", primary.id, err)
		}
		const maxJSON = 90_000
		for len(data) > maxJSON && len(drawer.References) > 0 {
			drawer.References = drawer.References[:len(drawer.References)/2]
			data, _ = json.Marshal(drawer)
		}

		for _, e := range g.entries {
			fmt.Fprintf(w, "INSERT INTO lexicon_drawer(id,data) VALUES(%d,%s);\n",
				e.id, sqlStr(string(data)))
			count++
		}
	}

	log.Printf("  wrote %d lexicon_drawer rows", count)
	return nil
}

func openReadOnly(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?mode=ro")
	if err != nil {
		return nil, err
	}
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA cache_size=-65536",
		"PRAGMA mmap_size=268435456",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("pragma %q: %w", p, err)
		}
	}
	return db, nil
}
