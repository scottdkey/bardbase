#!/usr/bin/env python3
"""
Shakespeare Database — Master Build Script

Builds shakespeare.db from source data:
  1. Parse OSS/Moby MySQL dump → works, characters, text (Public Domain)
  2. Parse Schmidt Lexicon XMLs → entries, senses, citations (CC BY-SA 3.0)
  3. Download Standard Ebooks plays → text lines (CC0)
  4. Download Standard Ebooks poetry → sonnets, poems (CC0)
  5. Add Folger reference URLs
  6. Build full-text search indexes

Usage:
    python3 tools/build.py                    # Full build
    python3 tools/build.py --skip-download    # Skip SE downloads (use cache)
    python3 tools/build.py --output build/    # Custom output directory
    python3 tools/build.py --step oss         # Run only one step
"""

import sqlite3
import os
import sys
import re
import json
import glob
import time
import argparse
import urllib.request
import xml.etree.ElementTree as ET
from html.parser import HTMLParser
from pathlib import Path
from datetime import datetime

# ============================================================
# PATHS
# ============================================================

SCRIPT_DIR = Path(__file__).parent
REPO_ROOT = SCRIPT_DIR.parent
SOURCES_DIR = REPO_ROOT / "sources"
OSS_SQL_PATH = SOURCES_DIR / "oss" / "oss-db-full.sql"
LEXICON_DIR = SOURCES_DIR / "lexicon" / "entries"

# ============================================================
# CONSTANTS
# ============================================================

# Schmidt abbreviation → (full title, Perseus text ID, work_type)
SCHMIDT_WORKS = {
    "Tp.": ("The Tempest", "1999.03.0056", "comedy"),
    "Gent.": ("Two Gentlemen of Verona", "1999.03.0032", "comedy"),
    "Gentl.": ("Two Gentlemen of Verona", "1999.03.0032", "comedy"),
    "Wiv.": ("Merry Wives of Windsor", "1999.03.0059", "comedy"),
    "Meas.": ("Measure for Measure", "1999.03.0049", "comedy"),
    "Err.": ("Comedy of Errors", "1999.03.0039", "comedy"),
    "Ado": ("Much Ado About Nothing", "1999.03.0047", "comedy"),
    "LLL": ("Love's Labour's Lost", "1999.03.0048", "comedy"),
    "Mids.": ("Midsummer Night's Dream", "1999.03.0051", "comedy"),
    "Merch.": ("Merchant of Venice", "1999.03.0050", "comedy"),
    "As": ("As You Like It", "1999.03.0038", "comedy"),
    "Shr.": ("Taming of the Shrew", "1999.03.0054", "comedy"),
    "All's": ("All's Well That Ends Well", "1999.03.0036", "comedy"),
    "Alls": ("All's Well That Ends Well", "1999.03.0036", "comedy"),
    "Tw.": ("Twelfth Night", "1999.03.0057", "comedy"),
    "Wint.": ("Winter's Tale", "1999.03.0060", "comedy"),
    "John": ("King John", "1999.03.0033", "history"),
    "R2": ("Richard II", "1999.03.0052", "history"),
    "H4A": ("Henry IV Part 1", "1999.03.0041", "history"),
    "H4B": ("Henry IV Part 2", "1999.03.0042", "history"),
    "H5": ("Henry V", "1999.03.0043", "history"),
    "H6A": ("Henry VI Part 1", "1999.03.0044", "history"),
    "H6B": ("Henry VI Part 2", "1999.03.0045", "history"),
    "H6C": ("Henry VI Part 3", "1999.03.0046", "history"),
    "R3": ("Richard III", "1999.03.0035", "history"),
    "H8": ("Henry VIII", "1999.03.0074", "history"),
    "Troil.": ("Troilus and Cressida", "1999.03.0058", "comedy"),
    "Cor.": ("Coriolanus", "1999.03.0026", "tragedy"),
    "Tit.": ("Titus Andronicus", "1999.03.0037", "tragedy"),
    "Rom.": ("Romeo and Juliet", "1999.03.0053", "tragedy"),
    "Tim.": ("Timon of Athens", "1999.03.0055", "tragedy"),
    "Caes.": ("Julius Caesar", "1999.03.0027", "tragedy"),
    "Mcb.": ("Macbeth", "1999.03.0028", "tragedy"),
    "Hml.": ("Hamlet", "1999.03.0031", "tragedy"),
    "Lr.": ("King Lear", "1999.03.0029", "tragedy"),
    "Oth.": ("Othello", "1999.03.0034", "tragedy"),
    "Ant.": ("Antony and Cleopatra", "1999.03.0025", "tragedy"),
    "Cymb.": ("Cymbeline", "1999.03.0040", "comedy"),
    "Per.": ("Pericles", "1999.03.0030", "comedy"),
    "Ven.": ("Venus and Adonis", "1999.03.0061", "poem"),
    "Lucr.": ("Rape of Lucrece", "1999.03.0062", "poem"),
    "Sonn.": ("Sonnets", "1999.03.0064", "sonnet_sequence"),
    "Pilgr.": ("Passionate Pilgrim", "1999.03.0063", "poem"),
    "Phoen.": ("Phoenix and the Turtle", "1999.03.0066", "poem"),
    "Compl.": ("Lover's Complaint", "1999.03.0065", "poem"),
    # Aliases without periods
    "Tp": ("The Tempest", "1999.03.0056", "comedy"),
    "Wiv": ("Merry Wives of Windsor", "1999.03.0059", "comedy"),
    "Meas": ("Measure for Measure", "1999.03.0049", "comedy"),
    "Err": ("Comedy of Errors", "1999.03.0039", "comedy"),
    "Mids": ("Midsummer Night's Dream", "1999.03.0051", "comedy"),
    "Merch": ("Merchant of Venice", "1999.03.0050", "comedy"),
    "Shr": ("Taming of the Shrew", "1999.03.0054", "comedy"),
    "Tw": ("Twelfth Night", "1999.03.0057", "comedy"),
    "Wint": ("Winter's Tale", "1999.03.0060", "comedy"),
    "Troil": ("Troilus and Cressida", "1999.03.0058", "comedy"),
    "Cor": ("Coriolanus", "1999.03.0026", "tragedy"),
    "Tit": ("Titus Andronicus", "1999.03.0037", "tragedy"),
    "Rom": ("Romeo and Juliet", "1999.03.0053", "tragedy"),
    "Tim": ("Timon of Athens", "1999.03.0055", "tragedy"),
    "Caes": ("Julius Caesar", "1999.03.0027", "tragedy"),
    "Mcb": ("Macbeth", "1999.03.0028", "tragedy"),
    "Hml": ("Hamlet", "1999.03.0031", "tragedy"),
    "Lr": ("King Lear", "1999.03.0029", "tragedy"),
    "Oth": ("Othello", "1999.03.0034", "tragedy"),
    "Ant": ("Antony and Cleopatra", "1999.03.0025", "tragedy"),
    "Cymb": ("Cymbeline", "1999.03.0040", "comedy"),
    "Per": ("Pericles", "1999.03.0030", "comedy"),
    "Ven": ("Venus and Adonis", "1999.03.0061", "poem"),
    "Lucr": ("Rape of Lucrece", "1999.03.0062", "poem"),
    "Sonn": ("Sonnets", "1999.03.0064", "sonnet_sequence"),
    "Pilgr": ("Passionate Pilgrim", "1999.03.0063", "poem"),
    "Phoen": ("Phoenix and the Turtle", "1999.03.0066", "poem"),
    "Compl": ("Lover's Complaint", "1999.03.0065", "poem"),
}

PERSEUS_TO_SCHMIDT = {
    "tmp": "Tp.", "tgv": "Gentl.", "wiv": "Wiv.", "mm": "Meas.",
    "err": "Err.", "ado": "Ado", "lll": "LLL", "mnd": "Mids.",
    "mv": "Merch.", "ayl": "As", "shr": "Shr.", "aww": "All's",
    "tn": "Tw.", "wt": "Wint.", "jn": "John", "r2": "R2",
    "1h4": "H4A", "2h4": "H4B", "h5": "H5", "1h6": "H6A",
    "2h6": "H6B", "3h6": "H6C", "r3": "R3", "h8": "H8",
    "tro": "Troil.", "cor": "Cor.", "tit": "Tit.", "rom": "Rom.",
    "tim": "Tim.", "jc": "Caes.", "mac": "Mcb.", "ham": "Hml.",
    "lr": "Lr.", "oth": "Oth.", "ant": "Ant.", "cym": "Cymb.",
    "per": "Per.", "ven": "Ven.", "luc": "Lucr.", "son": "Sonn.",
    "pp": "Pilgr.", "phoe": "Phoen.", "lc": "Compl.",
}

# OSS work ID → Schmidt abbreviation mapping
OSS_TO_SCHMIDT = {
    "tempest": "Tp.", "twogents": "Gent.", "merrywives": "Wiv.",
    "measure": "Meas.", "comedyerrors": "Err.", "muchado": "Ado",
    "loveslabours": "LLL", "midsummer": "Mids.", "merchantvenice": "Merch.",
    "asyoulikeit": "As", "tamingshrew": "Shr.", "allswell": "All's",
    "12night": "Tw.", "winterstale": "Wint.", "kingjohn": "John",
    "richard2": "R2", "henry4p1": "H4A", "henry4p2": "H4B",
    "henry5": "H5", "henry6p1": "H6A", "henry6p2": "H6B",
    "henry6p3": "H6C", "richard3": "R3", "henry8": "H8",
    "troilus": "Troil.", "coriolanus": "Cor.", "titus": "Tit.",
    "romeojuliet": "Rom.", "timonathens": "Tim.", "juliuscaesar": "Caes.",
    "macbeth": "Mcb.", "hamlet": "Hml.", "kinglear": "Lr.",
    "othello": "Oth.", "antonycleo": "Ant.", "cymbeline": "Cymb.",
    "pericles": "Per.", "venusadonis": "Ven.", "rapelucrece": "Lucr.",
    "sonnets": "Sonn.", "passionatepilgrim": "Pilgr.",
    "phoenixturtle": "Phoen.", "loverscomplaint": "Compl.",
    "mndream": "Mids.",
}

# Standard Ebooks repo → OSS ID mapping
SE_PLAY_REPOS = {
    'william-shakespeare_hamlet': 'hamlet',
    'william-shakespeare_romeo-and-juliet': 'romeojuliet',
    'william-shakespeare_the-tempest': 'tempest',
    'william-shakespeare_twelfth-night': '12night',
    'william-shakespeare_king-lear': 'kinglear',
    'william-shakespeare_julius-caesar': 'juliuscaesar',
    'william-shakespeare_antony-and-cleopatra': 'antonycleo',
    'william-shakespeare_henry-vi-part-ii': 'henry6p2',
    'william-shakespeare_the-merchant-of-venice': 'merchantvenice',
    'william-shakespeare_a-midsummer-nights-dream': 'midsummer',
    'william-shakespeare_othello': 'othello',
    'william-shakespeare_macbeth': 'macbeth',
    'william-shakespeare_cymbeline': 'cymbeline',
    'william-shakespeare_pericles': 'pericles',
    'william-shakespeare_coriolanus': 'coriolanus',
    'william-shakespeare_henry-viii': 'henry8',
    'william-shakespeare_richard-iii': 'richard3',
    'william-shakespeare_richard-ii': 'richard2',
    'william-shakespeare_henry-v': 'henry5',
    'william-shakespeare_titus-andronicus': 'titus',
    'william-shakespeare_king-john': 'kingjohn',
    'william-shakespeare_loves-labours-lost': 'loveslabours',
    'william-shakespeare_the-winters-tale': 'winterstale',
    'william-shakespeare_troilus-and-cressida': 'troilus',
    'william-shakespeare_timon-of-athens': 'timonathens',
    'william-shakespeare_measure-for-measure': 'measure',
    'william-shakespeare_henry-vi-part-iii': 'henry6p3',
    'william-shakespeare_henry-iv-part-i': 'henry4p1',
    'william-shakespeare_henry-vi-part-i': 'henry6p1',
    'william-shakespeare_henry-iv-part-ii': 'henry4p2',
    'william-shakespeare_the-comedy-of-errors': 'comedyerrors',
    'william-shakespeare_as-you-like-it': 'asyoulikeit',
    'william-shakespeare_much-ado-about-nothing': 'muchado',
    'william-shakespeare_the-two-gentlemen-of-verona': 'twogents',
    'william-shakespeare_the-merry-wives-of-windsor': 'merrywives',
    'william-shakespeare_alls-well-that-ends-well': 'allswell',
    'william-shakespeare_the-taming-of-the-shrew': 'tamingshrew',
}

SE_POETRY_MAP = {
    'venus-and-adonis': 'venusadonis',
    'the-rape-of-lucrece': 'rapelucrece',
    'the-passionate-pilgrim': 'passionatepilgrim',
    'the-pheonix-and-the-turtle': 'phoenixturtle',
}

FOLGER_SLUGS = {
    'tempest': 'the-tempest', 'twogents': 'the-two-gentlemen-of-verona',
    'merrywives': 'the-merry-wives-of-windsor', 'measure': 'measure-for-measure',
    'comedyerrors': 'the-comedy-of-errors', 'muchado': 'much-ado-about-nothing',
    'loveslabours': 'loves-labors-lost', 'midsummer': 'a-midsummer-nights-dream',
    'merchantvenice': 'the-merchant-of-venice', 'asyoulikeit': 'as-you-like-it',
    'tamingshrew': 'the-taming-of-the-shrew', 'allswell': 'alls-well-that-ends-well',
    '12night': 'twelfth-night', 'winterstale': 'the-winters-tale',
    'kingjohn': 'king-john', 'richard2': 'richard-ii',
    'henry4p1': 'henry-iv-part-1', 'henry4p2': 'henry-iv-part-2',
    'henry5': 'henry-v', 'henry6p1': 'henry-vi-part-1',
    'henry6p2': 'henry-vi-part-2', 'henry6p3': 'henry-vi-part-3',
    'richard3': 'richard-iii', 'henry8': 'henry-viii',
    'troilus': 'troilus-and-cressida', 'coriolanus': 'coriolanus',
    'titus': 'titus-andronicus', 'romeojuliet': 'romeo-and-juliet',
    'timonathens': 'timon-of-athens', 'juliuscaesar': 'julius-caesar',
    'macbeth': 'macbeth', 'hamlet': 'hamlet',
    'kinglear': 'king-lear', 'othello': 'othello',
    'antonycleo': 'antony-and-cleopatra', 'cymbeline': 'cymbeline',
    'pericles': 'pericles',
}

GENRE_MAP = {'c': 'comedy', 'h': 'history', 't': 'tragedy', 'p': 'poem', 's': 'sonnet_sequence'}


# ============================================================
# SCHEMA
# ============================================================

SCHEMA_SQL = """
-- Sources: where data comes from
CREATE TABLE IF NOT EXISTS sources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    short_code TEXT NOT NULL UNIQUE,
    url TEXT,
    license TEXT,
    license_url TEXT,
    attribution_text TEXT,
    attribution_required BOOLEAN DEFAULT 0,
    notes TEXT,
    imported_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Works: all 43 Shakespeare works
CREATE TABLE IF NOT EXISTS works (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    oss_id TEXT UNIQUE,
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
CREATE INDEX IF NOT EXISTS idx_works_oss_id ON works(oss_id);
CREATE INDEX IF NOT EXISTS idx_works_schmidt ON works(schmidt_abbrev);

-- Characters
CREATE TABLE IF NOT EXISTS characters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    char_id TEXT UNIQUE,
    name TEXT NOT NULL,
    abbrev TEXT,
    work_id INTEGER REFERENCES works(id),
    oss_work_id TEXT,
    description TEXT,
    speech_count INTEGER
);
CREATE INDEX IF NOT EXISTS idx_characters_work ON characters(work_id);

-- Editions
CREATE TABLE IF NOT EXISTS editions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    short_code TEXT NOT NULL UNIQUE,
    source_id INTEGER REFERENCES sources(id),
    year INTEGER,
    editors TEXT,
    description TEXT,
    notes TEXT
);

-- Text lines: the actual play/poem text
CREATE TABLE IF NOT EXISTS text_lines (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    work_id INTEGER NOT NULL REFERENCES works(id),
    edition_id INTEGER NOT NULL REFERENCES editions(id),
    act INTEGER,
    scene INTEGER,
    paragraph_num INTEGER,
    character_id INTEGER REFERENCES characters(id),
    char_name TEXT,
    content TEXT NOT NULL,
    content_type TEXT DEFAULT 'speech',
    word_count INTEGER DEFAULT 0,
    oss_paragraph_id INTEGER,
    sonnet_number INTEGER,
    stanza INTEGER
);
CREATE INDEX IF NOT EXISTS idx_text_work_edition ON text_lines(work_id, edition_id);
CREATE INDEX IF NOT EXISTS idx_text_location ON text_lines(work_id, act, scene);

-- Structural divisions (acts/scenes)
CREATE TABLE IF NOT EXISTS text_divisions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    work_id INTEGER NOT NULL REFERENCES works(id),
    edition_id INTEGER NOT NULL REFERENCES editions(id),
    act INTEGER NOT NULL,
    scene INTEGER NOT NULL,
    description TEXT,
    line_count INTEGER DEFAULT 0,
    UNIQUE(work_id, edition_id, act, scene)
);

-- Lexicon entries (Schmidt)
CREATE TABLE IF NOT EXISTS lexicon_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    letter TEXT NOT NULL,
    orthography TEXT,
    entry_type TEXT DEFAULT 'main',
    full_text TEXT,
    raw_xml TEXT,
    source_file TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_lexicon_key ON lexicon_entries(key);
CREATE INDEX IF NOT EXISTS idx_lexicon_letter ON lexicon_entries(letter);

-- Lexicon senses
CREATE TABLE IF NOT EXISTS lexicon_senses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entry_id INTEGER NOT NULL REFERENCES lexicon_entries(id),
    sense_number INTEGER NOT NULL,
    definition_text TEXT,
    UNIQUE(entry_id, sense_number)
);
CREATE INDEX IF NOT EXISTS idx_senses_entry ON lexicon_senses(entry_id);

-- Lexicon citations
CREATE TABLE IF NOT EXISTS lexicon_citations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entry_id INTEGER NOT NULL REFERENCES lexicon_entries(id),
    sense_id INTEGER REFERENCES lexicon_senses(id),
    work_id INTEGER REFERENCES works(id),
    work_abbrev TEXT,
    perseus_ref TEXT,
    act INTEGER,
    scene INTEGER,
    line INTEGER,
    quote_text TEXT,
    display_text TEXT,
    raw_bibl TEXT
);
CREATE INDEX IF NOT EXISTS idx_citations_entry ON lexicon_citations(entry_id);
CREATE INDEX IF NOT EXISTS idx_citations_work ON lexicon_citations(work_id);
CREATE INDEX IF NOT EXISTS idx_citations_location ON lexicon_citations(work_abbrev, act, scene, line);

-- Citation-to-text resolved links (future)
CREATE TABLE IF NOT EXISTS citation_matches (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    citation_id INTEGER NOT NULL REFERENCES lexicon_citations(id),
    text_line_id INTEGER NOT NULL REFERENCES text_lines(id),
    edition_id INTEGER NOT NULL REFERENCES editions(id),
    match_type TEXT DEFAULT 'exact',
    confidence REAL DEFAULT 1.0,
    matched_text TEXT,
    notes TEXT
);

-- Cross-edition line concordance (future)
CREATE TABLE IF NOT EXISTS line_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    work_id INTEGER NOT NULL REFERENCES works(id),
    act INTEGER,
    scene INTEGER,
    globe_line INTEGER,
    f1_tln INTEGER,
    f1_line INTEGER,
    notes TEXT
);

-- Build tracking
CREATE TABLE IF NOT EXISTS import_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    phase TEXT NOT NULL,
    action TEXT NOT NULL,
    details TEXT,
    count INTEGER DEFAULT 0,
    duration_secs REAL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Full-text search: lexicon
CREATE VIRTUAL TABLE IF NOT EXISTS lexicon_fts USING fts5(
    key, orthography, full_text,
    content='lexicon_entries',
    content_rowid='id',
    tokenize='porter unicode61'
);

-- Full-text search: text
CREATE VIRTUAL TABLE IF NOT EXISTS text_fts USING fts5(
    content, char_name,
    content='text_lines',
    content_rowid='id',
    tokenize='porter unicode61'
);
"""


# ============================================================
# STEP 1: OSS/Moby Import
# ============================================================

def parse_mysql_values(line):
    """
    Parse a MySQL INSERT INTO ... VALUES (...), (...); line.
    Returns list of tuples of string values.
    """
    # Find the VALUES portion
    match = re.search(r'VALUES\s*', line, re.IGNORECASE)
    if not match:
        return []

    values_str = line[match.end():]
    rows = []
    i = 0
    length = len(values_str)

    while i < length:
        # Skip to opening paren
        while i < length and values_str[i] != '(':
            i += 1
        if i >= length:
            break
        i += 1  # skip '('

        # Parse one row
        fields = []
        while i < length and values_str[i] != ')':
            if values_str[i] == "'":
                # Quoted string
                i += 1
                val = []
                while i < length:
                    if values_str[i] == '\\' and i + 1 < length:
                        val.append(values_str[i + 1])
                        i += 2
                    elif values_str[i] == "'" and i + 1 < length and values_str[i + 1] == "'":
                        val.append("'")
                        i += 2
                    elif values_str[i] == "'":
                        i += 1
                        break
                    else:
                        val.append(values_str[i])
                        i += 1
                fields.append(''.join(val))
            elif values_str[i] in (' ', ','):
                i += 1
            elif values_str[i:i+4].upper() == 'NULL':
                fields.append(None)
                i += 4
            else:
                # Unquoted value (number)
                val = []
                while i < length and values_str[i] not in (',', ')'):
                    val.append(values_str[i])
                    i += 1
                fields.append(''.join(val).strip())

        rows.append(tuple(fields))
        i += 1  # skip ')'

    return rows


def decode_html_entities(text):
    """Decode HTML numeric entities like &#8217; to characters."""
    if not text:
        return text
    def replace_entity(m):
        try:
            return chr(int(m.group(1)))
        except (ValueError, OverflowError):
            return m.group(0)
    return re.sub(r'&#(\d+);', replace_entity, text)


def import_oss(conn, sql_path):
    """Import OSS/Moby Shakespeare data from MySQL dump."""
    print("=" * 60)
    print("STEP 1: Import OSS/Moby Shakespeare")
    print("=" * 60)

    if not sql_path.exists():
        print(f"  ERROR: OSS SQL dump not found at {sql_path}")
        return False

    start = time.time()

    # Create OSS source
    conn.execute("""
        INSERT OR IGNORE INTO sources (name, short_code, url, license, attribution_text, attribution_required, notes)
        VALUES ('Open Source Shakespeare / Moby', 'oss_moby', 'https://www.opensourceshakespeare.org',
                'Public Domain', NULL, 0,
                'Globe-based modern spelling text. Originally from Moby project.')
    """)
    source_id = conn.execute("SELECT id FROM sources WHERE short_code = 'oss_moby'").fetchone()[0]

    # Create OSS edition
    conn.execute("""
        INSERT OR IGNORE INTO editions (name, short_code, source_id, year, editors, description)
        VALUES ('Open Source Shakespeare (Globe)', 'oss_globe', ?, 2003,
                'George Mason University', 'Globe-based text via Moby project')
    """, (source_id,))
    edition_id = conn.execute("SELECT id FROM editions WHERE short_code = 'oss_globe'").fetchone()[0]

    # Read the SQL file
    with open(sql_path, 'r', encoding='latin-1') as f:
        sql_content = f.read()

    # Parse works
    works_data = {}
    chapters_data = {}
    characters_data = []
    paragraphs_data = []

    # Extract complete SQL statements (quote-aware — handles semicolons inside strings)
    all_statements = []
    i = 0
    length = len(sql_content)
    while i < length:
        while i < length and sql_content[i] in (' ', '\t', '\n', '\r'):
            i += 1
        if i >= length:
            break
        if sql_content[i:i+2] == '--':
            while i < length and sql_content[i] != '\n':
                i += 1
            continue
        if sql_content[i:i+2] == '/*':
            end = sql_content.find('*/', i + 2)
            i = end + 2 if end != -1 else length
            continue
        stmt_start = i
        in_quote = False
        while i < length:
            if sql_content[i] == "'" and not in_quote:
                in_quote = True
                i += 1
            elif sql_content[i] == "'" and in_quote:
                if i + 1 < length and sql_content[i + 1] == "'":
                    i += 2
                elif i > 0 and sql_content[i - 1] == '\\':
                    i += 1
                else:
                    in_quote = False
                    i += 1
            elif sql_content[i] == ';' and not in_quote:
                all_statements.append(sql_content[stmt_start:i + 1])
                i += 1
                break
            else:
                i += 1
        else:
            remaining = sql_content[stmt_start:].strip()
            if remaining:
                all_statements.append(remaining)

    for stmt in all_statements:
        if 'INSERT INTO' not in stmt.upper():
            continue

        if 'INSERT INTO `Works`' in stmt:
            rows = parse_mysql_values(stmt)
            for row in rows:
                if len(row) >= 10:
                    wid, title, long_title, short_title, date, genre, notes, source, total_words, total_paras = row[:10]
                    works_data[wid] = {
                        'oss_id': wid,
                        'title': decode_html_entities(title),
                        'full_title': decode_html_entities(long_title),
                        'short_title': short_title,
                        'date_composed': int(date) if date else None,
                        'genre_type': genre,
                        'work_type': GENRE_MAP.get(genre, genre),
                        'total_words': int(total_words) if total_words else None,
                        'total_paragraphs': int(total_paras) if total_paras else None,
                        'source_text': source,
                    }

        elif 'INSERT INTO `Chapters`' in stmt:
            rows = parse_mysql_values(stmt)
            for row in rows:
                if len(row) >= 5:
                    work_id, chapter_id, section, chapter, desc = row[:5]
                    chapters_data[int(chapter_id)] = {
                        'work_id': work_id,
                        'section': int(section),
                        'chapter': int(chapter),
                        'description': decode_html_entities(desc),
                    }

        elif 'INSERT INTO `Characters`' in stmt:
            rows = parse_mysql_values(stmt)
            for row in rows:
                if len(row) >= 6:
                    char_id, name, abbrev, work_id, desc, speech_count = row[:6]
                    characters_data.append({
                        'char_id': char_id,
                        'name': decode_html_entities(name),
                        'abbrev': abbrev,
                        'oss_work_id': work_id,
                        'description': decode_html_entities(desc),
                        'speech_count': int(speech_count) if speech_count else None,
                    })

        elif 'INSERT INTO `Paragraphs`' in stmt:
            rows = parse_mysql_values(stmt)
            for row in rows:
                if len(row) >= 12:
                    work_id, para_id, para_num, char_id, plain_text = row[0], row[1], row[2], row[3], row[4]
                    para_type, section, chapter, char_count, word_count = row[7], row[8], row[9], row[10], row[11]
                    paragraphs_data.append({
                        'work_id': work_id,
                        'paragraph_id': int(para_id),
                        'paragraph_num': int(para_num),
                        'char_id': char_id,
                        'text': decode_html_entities(plain_text),
                        'type': para_type,
                        'section': int(section),
                        'chapter': int(chapter),
                        'word_count': int(word_count) if word_count else 0,
                    })

    # Insert works
    print(f"  Works: {len(works_data)}")
    for wid, w in works_data.items():
        schmidt = OSS_TO_SCHMIDT.get(wid)
        perseus_id = SCHMIDT_WORKS.get(schmidt, (None, None, None))[1] if schmidt else None
        conn.execute("""
            INSERT OR IGNORE INTO works (oss_id, title, full_title, short_title, schmidt_abbrev,
                work_type, date_composed, genre_type, total_words, total_paragraphs, source_text, perseus_id)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """, (wid, w['title'], w['full_title'], w['short_title'], schmidt,
              w['work_type'], w['date_composed'], w['genre_type'],
              w['total_words'], w['total_paragraphs'], w['source_text'], perseus_id))

    # Build work ID map
    work_id_map = {}
    for row in conn.execute("SELECT id, oss_id FROM works"):
        work_id_map[row[1]] = row[0]

    # Insert characters
    print(f"  Characters: {len(characters_data)}")
    for c in characters_data:
        db_work_id = work_id_map.get(c['oss_work_id'])
        conn.execute("""
            INSERT OR IGNORE INTO characters (char_id, name, abbrev, work_id, oss_work_id, description, speech_count)
            VALUES (?, ?, ?, ?, ?, ?, ?)
        """, (c['char_id'], c['name'], c['abbrev'], db_work_id, c['oss_work_id'],
              c['description'], c['speech_count']))

    # Build char ID maps
    char_id_map = {}
    for row in conn.execute("SELECT id, char_id FROM characters"):
        char_id_map[row[1]] = row[0]
    char_name_map = {c['char_id']: c['name'] for c in characters_data}

    # Insert text_divisions from chapters
    print(f"  Chapters (divisions): {len(chapters_data)}")
    for cid, ch in chapters_data.items():
        db_work_id = work_id_map.get(ch['work_id'])
        if db_work_id:
            conn.execute("""
                INSERT OR IGNORE INTO text_divisions (work_id, edition_id, act, scene, description)
                VALUES (?, ?, ?, ?, ?)
            """, (db_work_id, edition_id, ch['section'], ch['chapter'], ch['description']))

    # Insert text_lines from paragraphs
    print(f"  Paragraphs (text lines): {len(paragraphs_data)}")
    for p in paragraphs_data:
        db_work_id = work_id_map.get(p['work_id'])
        char_db_id = char_id_map.get(p['char_id'])
        char_name = char_name_map.get(p['char_id'])

        content_type = 'stage_direction' if p['type'] == 'd' else 'speech'

        if db_work_id:
            conn.execute("""
                INSERT INTO text_lines (work_id, edition_id, act, scene, paragraph_num,
                    character_id, char_name, content, content_type, word_count, oss_paragraph_id)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            """, (db_work_id, edition_id, p['section'], p['chapter'], p['paragraph_num'],
                  char_db_id, char_name, p['text'], content_type, p['word_count'], p['paragraph_id']))

    conn.commit()

    elapsed = time.time() - start
    line_count = conn.execute("SELECT COUNT(*) FROM text_lines WHERE edition_id = ?", (edition_id,)).fetchone()[0]
    conn.execute("INSERT INTO import_log (phase, action, details, count, duration_secs) VALUES (?, ?, ?, ?, ?)",
                 ('oss', 'import_complete', f'{len(works_data)} works, {len(characters_data)} chars', line_count, elapsed))
    conn.commit()

    print(f"  ✓ {line_count:,} text lines imported in {elapsed:.1f}s")
    return True


# ============================================================
# STEP 2: Schmidt Lexicon Import
# ============================================================

def get_element_text(elem):
    """Recursively get all text from an XML element."""
    parts = []
    if elem.text:
        parts.append(elem.text)
    for child in elem:
        parts.append(get_element_text(child))
        if child.tail:
            parts.append(child.tail)
    return ''.join(parts)


def parse_perseus_ref(bibl_n):
    """Parse Perseus bibl n= attribute into components."""
    if not bibl_n or not bibl_n.startswith('shak.'):
        return None
    parts = bibl_n.replace('shak. ', '').strip().split()
    if len(parts) < 2:
        return None
    work_code = parts[0]
    numbers = parts[1] if len(parts) > 1 else ""
    schmidt_abbrev = PERSEUS_TO_SCHMIDT.get(work_code)
    if not schmidt_abbrev:
        return None
    num_parts = numbers.split('.')
    act = scene = line = None
    try:
        if len(num_parts) == 3:
            act, scene, line = int(num_parts[0]), int(num_parts[1]), int(num_parts[2])
        elif len(num_parts) == 2:
            act, line = int(num_parts[0]), int(num_parts[1])
        elif len(num_parts) == 1 and num_parts[0]:
            line = int(num_parts[0])
    except ValueError:
        pass
    return (schmidt_abbrev, act, scene, line, bibl_n)


def parse_senses(full_text):
    """Split entry text into numbered senses."""
    sense_pattern = re.compile(r'(?:^|\s)(\d+)\)\s')
    matches = list(sense_pattern.finditer(full_text))
    if not matches:
        return [{'number': 1, 'text': full_text}]
    senses = []
    for i, match in enumerate(matches):
        num = int(match.group(1))
        start = match.end()
        end = matches[i + 1].start() if i + 1 < len(matches) else len(full_text)
        senses.append({'number': num, 'text': full_text[start:end].strip()})
    return senses


def parse_entry_xml(xml_path):
    """Parse a single lexicon entry XML file."""
    try:
        with open(xml_path, 'r', encoding='utf-8') as f:
            raw_xml = f.read()
        root = ET.fromstring(raw_xml)
    except Exception:
        try:
            raw_xml_fixed = raw_xml.replace('&', '&amp;')
            root = ET.fromstring(raw_xml_fixed)
        except Exception:
            return None

    entry_free = root.find('.//entryFree')
    if entry_free is None:
        return None

    key = entry_free.get('key', '')
    entry_type = entry_free.get('type', 'main')
    div1 = root.find('.//div1')
    letter = div1.get('n', '') if div1 is not None else os.path.basename(os.path.dirname(xml_path))
    orth_elem = entry_free.find('orth')
    orthography = get_element_text(orth_elem) if orth_elem is not None else key
    full_text = re.sub(r'\s+', ' ', get_element_text(entry_free).strip())
    senses = parse_senses(full_text)

    citations = []
    for bibl in entry_free.iter('bibl'):
        bibl_n = bibl.get('n', '')
        display_text = get_element_text(bibl).strip()
        quote_text = None
        for cit in entry_free.iter('cit'):
            if bibl in list(cit.iter('bibl')):
                quote_elem = cit.find('quote')
                if quote_elem is not None:
                    quote_text = get_element_text(quote_elem).strip()
                break
        parsed = parse_perseus_ref(bibl_n)
        if parsed:
            s_abbrev, act, scene, line, p_ref = parsed
            citations.append({'work_abbrev': s_abbrev, 'act': act, 'scene': scene, 'line': line,
                              'raw_bibl': display_text, 'perseus_ref': p_ref, 'quote_text': quote_text,
                              'display_text': display_text})
        elif display_text:
            citations.append({'work_abbrev': None, 'act': None, 'scene': None, 'line': None,
                              'raw_bibl': display_text, 'perseus_ref': bibl_n or None,
                              'quote_text': quote_text, 'display_text': display_text})

    return {'key': key, 'letter': letter, 'entry_type': entry_type, 'orthography': orthography,
            'full_text': full_text, 'raw_xml': raw_xml, 'senses': senses, 'citations': citations,
            'source_file': os.path.basename(xml_path)}


def import_lexicon(conn, entries_dir):
    """Import Schmidt lexicon XML entries."""
    print("\n" + "=" * 60)
    print("STEP 2: Import Schmidt Lexicon")
    print("=" * 60)

    if not entries_dir.exists():
        print(f"  WARNING: Lexicon entries not found at {entries_dir}")
        print("  Skipping lexicon import (scraper may still be running)")
        return True

    # Create Perseus source
    conn.execute("""
        INSERT OR IGNORE INTO sources (name, short_code, url, license, license_url,
            attribution_text, attribution_required, notes)
        VALUES ('Perseus Digital Library — Schmidt Shakespeare Lexicon', 'perseus_schmidt',
                'http://www.perseus.tufts.edu', 'CC BY-SA 3.0',
                'https://creativecommons.org/licenses/by-sa/3.0/',
                'Alexander Schmidt, Shakespeare Lexicon and Quotation Dictionary. Provided by the Perseus Digital Library, Tufts University. Licensed under CC BY-SA 3.0.',
                1, 'Schmidt lexicon entries scraped from Perseus TEI XML.')
    """)

    start = time.time()
    xml_files = sorted(glob.glob(str(entries_dir / '*' / '*.xml')))
    print(f"  Found {len(xml_files)} XML files")

    if not xml_files:
        print("  No XML files found, skipping")
        return True

    # Build work abbrev → id map
    work_map = {}
    for row in conn.execute("SELECT id, schmidt_abbrev FROM works WHERE schmidt_abbrev IS NOT NULL"):
        work_map[row[1]] = row[0]
        work_map[row[1].rstrip('.')] = row[0]
    for abbrev in SCHMIDT_WORKS:
        title = SCHMIDT_WORKS[abbrev][0]
        for db_abbrev, db_id in list(work_map.items()):
            if SCHMIDT_WORKS.get(db_abbrev, (None,))[0] == title:
                work_map[abbrev] = db_id

    total_entries = 0
    total_citations = 0
    total_senses = 0
    errors = 0

    # Group by letter
    letter_dirs = sorted(set(os.path.dirname(f) for f in xml_files))

    for letter_dir in letter_dirs:
        letter = os.path.basename(letter_dir)
        letter_files = [f for f in xml_files if os.path.dirname(f) == letter_dir]
        letter_entries = 0

        for xml_path in letter_files:
            entry = parse_entry_xml(xml_path)
            if entry is None:
                errors += 1
                continue

            conn.execute("""
                INSERT OR IGNORE INTO lexicon_entries (key, letter, orthography, entry_type, full_text, raw_xml, source_file)
                VALUES (?, ?, ?, ?, ?, ?, ?)
            """, (entry['key'], entry['letter'], entry['orthography'], entry['entry_type'],
                  entry['full_text'], entry['raw_xml'], entry['source_file']))
            entry_id = conn.execute("SELECT id FROM lexicon_entries WHERE key = ?", (entry['key'],)).fetchone()
            if not entry_id:
                continue
            entry_id = entry_id[0]

            for sense in entry['senses']:
                conn.execute("INSERT OR IGNORE INTO lexicon_senses (entry_id, sense_number, definition_text) VALUES (?, ?, ?)",
                             (entry_id, sense['number'], sense['text']))
                total_senses += 1

            for cit in entry['citations']:
                work_id = work_map.get(cit['work_abbrev'])
                conn.execute("""
                    INSERT INTO lexicon_citations (entry_id, work_id, work_abbrev, perseus_ref,
                        act, scene, line, quote_text, display_text, raw_bibl)
                    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                """, (entry_id, work_id, cit['work_abbrev'], cit['perseus_ref'],
                      cit['act'], cit['scene'], cit['line'], cit['quote_text'],
                      cit['display_text'], cit['raw_bibl']))
                total_citations += 1

            letter_entries += 1

        if letter_entries > 0:
            conn.commit()
            total_entries += letter_entries
            print(f"  {letter}: {letter_entries} entries")

    elapsed = time.time() - start
    conn.execute("INSERT INTO import_log (phase, action, details, count, duration_secs) VALUES (?, ?, ?, ?, ?)",
                 ('lexicon', 'import_complete', f'{total_entries} entries, {total_citations} citations, {errors} errors',
                  total_entries, elapsed))
    conn.commit()
    print(f"  ✓ {total_entries:,} entries, {total_citations:,} citations, {total_senses:,} senses in {elapsed:.1f}s")
    return True


# ============================================================
# STEP 3: Standard Ebooks Plays
# ============================================================

class SEPlayParser(HTMLParser):
    """Parse Standard Ebooks play XHTML."""
    def __init__(self):
        super().__init__()
        self.lines = []
        self.current_act = 0
        self.current_scene = 0
        self.current_character = None
        self.buf = ''
        self.in_persona = False
        self.in_stage_dir = False
        self.in_verse_span = False
        self.in_td = False
        self.in_header = False
        self.td_is_speech = False
        self.verse_lines = []
        self.scene_line_counter = 0
        self.tag_stack = []
        self.nested_stage_dir = False

    def _epub_type(self, attrs):
        return dict(attrs).get('epub:type', '')

    def handle_starttag(self, tag, attrs):
        self.tag_stack.append(tag)
        et = self._epub_type(attrs)
        ad = dict(attrs)
        if tag == 'section':
            sid = ad.get('id', '')
            if sid.startswith('scene-'):
                parts = sid.split('-')
                if len(parts) >= 3:
                    try:
                        self.current_act = int(parts[1])
                        self.current_scene = int(parts[2])
                    except ValueError:
                        pass
                self.scene_line_counter = 0
            elif sid.startswith('act-'):
                try:
                    self.current_act = int(sid.split('-')[1])
                except (ValueError, IndexError):
                    pass
            if 'prologue' in et:
                self.current_scene = 0
            elif 'epilogue' in et:
                self.current_scene = 99
        elif tag in ('h2', 'h3', 'h4', 'hgroup'):
            self.in_header = True
        elif tag == 'td':
            self.in_td = True
            if 'z3998:persona' in et:
                self.in_persona = True
                self.buf = ''
            else:
                self.td_is_speech = True
                self.buf = ''
                self.verse_lines = []
        elif tag == 'i' and 'z3998:stage-direction' in et:
            if self.td_is_speech:
                self.nested_stage_dir = True
                self.buf = ''
            else:
                self.in_stage_dir = True
                self.buf = ''
        elif tag == 'span' and self.td_is_speech and not self.nested_stage_dir:
            self.in_verse_span = True
            self.buf = ''

    def handle_endtag(self, tag):
        if self.tag_stack and self.tag_stack[-1] == tag:
            self.tag_stack.pop()
        if tag == 'td' and self.in_persona:
            self.current_character = self.buf.strip()
            self.in_persona = False
            self.in_td = False
        elif tag == 'span' and self.in_verse_span:
            t = self.buf.strip()
            if t:
                self.verse_lines.append(t)
            self.in_verse_span = False
            self.buf = ''
        elif tag == 'i' and self.nested_stage_dir:
            sd = self.buf.strip()
            if sd:
                self.verse_lines.append(('SD', sd))
            self.nested_stage_dir = False
            self.buf = ''
        elif tag == 'i' and self.in_stage_dir:
            sd = self.buf.strip()
            if sd:
                self.scene_line_counter += 1
                self.lines.append({'act': self.current_act, 'scene': self.current_scene,
                                   'character': None, 'text': sd, 'is_stage_direction': True,
                                   'line_in_scene': self.scene_line_counter})
            self.in_stage_dir = False
            self.buf = ''
        elif tag == 'td' and self.td_is_speech:
            if self.verse_lines:
                for item in self.verse_lines:
                    if isinstance(item, tuple) and item[0] == 'SD':
                        self.scene_line_counter += 1
                        self.lines.append({'act': self.current_act, 'scene': self.current_scene,
                                           'character': None, 'text': item[1], 'is_stage_direction': True,
                                           'line_in_scene': self.scene_line_counter})
                    else:
                        self.scene_line_counter += 1
                        self.lines.append({'act': self.current_act, 'scene': self.current_scene,
                                           'character': self.current_character, 'text': item,
                                           'is_stage_direction': False, 'line_in_scene': self.scene_line_counter})
            else:
                prose = self.buf.strip()
                if prose:
                    self.scene_line_counter += 1
                    self.lines.append({'act': self.current_act, 'scene': self.current_scene,
                                       'character': self.current_character, 'text': prose,
                                       'is_stage_direction': False, 'line_in_scene': self.scene_line_counter})
            self.verse_lines = []
            self.td_is_speech = False
            self.in_td = False
            self.buf = ''
        elif tag in ('h2', 'h3', 'h4', 'hgroup'):
            self.in_header = False

    def handle_data(self, data):
        if self.in_header:
            return
        if self.in_persona or self.in_stage_dir or self.in_verse_span or self.nested_stage_dir:
            self.buf += data
        elif self.td_is_speech and self.in_td:
            self.buf += data

    def handle_entityref(self, name):
        c = {'amp': '&', 'lt': '<', 'gt': '>', 'quot': '"', 'apos': "'", 'nbsp': ' '}.get(name, f'&{name};')
        self.handle_data(c)

    def handle_charref(self, name):
        try:
            c = chr(int(name[1:], 16)) if name.startswith('x') else chr(int(name))
            self.handle_data(c)
        except ValueError:
            self.handle_data(f'&#{name};')


class PoetryXHTMLParser(HTMLParser):
    """Parse SE poetry XHTML."""
    def __init__(self):
        super().__init__()
        self.poems = {}
        self.current_article = None
        self.buf = ''
        self.in_span = False
        self.in_header = False
        self.in_dedication = False
        self.line_counter = 0
        self.stanza_counter = 0
        self.tag_stack = []

    def handle_starttag(self, tag, attrs):
        self.tag_stack.append(tag)
        ad = dict(attrs)
        et = ad.get('epub:type', '')
        if tag == 'article':
            aid = ad.get('id', '')
            self.current_article = aid
            self.poems[aid] = []
            self.line_counter = 0
            self.stanza_counter = 0
        elif tag == 'section':
            sid = ad.get('id', '')
            if 'dedication' in sid or 'dedication' in et:
                self.in_dedication = True
        elif tag in ('h2', 'h3', 'h4', 'header', 'hgroup'):
            self.in_header = True
        elif tag == 'span' and self.current_article and not self.in_header and not self.in_dedication:
            self.in_span = True
            self.buf = ''
        elif tag == 'p' and self.current_article and not self.in_header and not self.in_dedication:
            self.stanza_counter += 1

    def handle_endtag(self, tag):
        if self.tag_stack and self.tag_stack[-1] == tag:
            self.tag_stack.pop()
        if tag == 'span' and self.in_span:
            text = self.buf.strip()
            if text and self.current_article:
                self.line_counter += 1
                self.poems[self.current_article].append({
                    'text': text, 'line_number': self.line_counter, 'stanza': self.stanza_counter})
            self.in_span = False
            self.buf = ''
        elif tag == 'article':
            self.current_article = None
            self.in_dedication = False
        elif tag == 'section' and self.in_dedication:
            self.in_dedication = False
        elif tag in ('h2', 'h3', 'h4', 'header', 'hgroup'):
            self.in_header = False

    def handle_data(self, data):
        if self.in_span:
            self.buf += data

    def handle_entityref(self, name):
        if self.in_span:
            self.buf += {'amp': '&', 'lt': '<', 'gt': '>', 'nbsp': ' '}.get(name, f'&{name};')

    def handle_charref(self, name):
        if self.in_span:
            try:
                self.buf += chr(int(name[1:], 16)) if name.startswith('x') else chr(int(name))
            except ValueError:
                self.buf += f'&#{name};'


class SonnetParser(HTMLParser):
    """Parse SE sonnets XHTML."""
    def __init__(self):
        super().__init__()
        self.sonnets = {}
        self.lovers_complaint = []
        self.current_article = None
        self.current_sonnet_num = None
        self.is_lovers_complaint = False
        self.buf = ''
        self.in_span = False
        self.in_header = False
        self.line_counter = 0
        self.stanza_counter = 0
        self.tag_stack = []

    def handle_starttag(self, tag, attrs):
        self.tag_stack.append(tag)
        ad = dict(attrs)
        if tag == 'article':
            aid = ad.get('id', '')
            self.current_article = aid
            self.line_counter = 0
            self.stanza_counter = 0
            if aid.startswith('sonnet-'):
                try:
                    self.current_sonnet_num = int(aid.split('-')[1])
                    self.sonnets[self.current_sonnet_num] = []
                except (ValueError, IndexError):
                    pass
            elif 'lover' in aid.lower() or 'complaint' in aid.lower():
                self.is_lovers_complaint = True
        elif tag in ('h2', 'h3', 'h4', 'header', 'hgroup'):
            self.in_header = True
        elif tag == 'span' and self.current_article and not self.in_header:
            self.in_span = True
            self.buf = ''
        elif tag == 'p' and self.current_article and not self.in_header:
            self.stanza_counter += 1

    def handle_endtag(self, tag):
        if self.tag_stack and self.tag_stack[-1] == tag:
            self.tag_stack.pop()
        if tag == 'span' and self.in_span:
            text = self.buf.strip()
            if text:
                self.line_counter += 1
                entry = {'text': text, 'line_number': self.line_counter, 'stanza': self.stanza_counter}
                if self.is_lovers_complaint:
                    self.lovers_complaint.append(entry)
                elif self.current_sonnet_num is not None:
                    self.sonnets[self.current_sonnet_num].append(entry)
            self.in_span = False
            self.buf = ''
        elif tag == 'article':
            self.current_article = None
            self.current_sonnet_num = None
            self.is_lovers_complaint = False
        elif tag in ('h2', 'h3', 'h4', 'header', 'hgroup'):
            self.in_header = False

    def handle_data(self, data):
        if self.in_span:
            self.buf += data

    def handle_entityref(self, name):
        if self.in_span:
            self.buf += {'amp': '&', 'lt': '<', 'gt': '>', 'nbsp': ' '}.get(name, f'&{name};')

    def handle_charref(self, name):
        if self.in_span:
            try:
                self.buf += chr(int(name[1:], 16)) if name.startswith('x') else chr(int(name))
            except ValueError:
                self.buf += f'&#{name};'


def fetch_url(url, retries=3):
    """Fetch URL with retries."""
    for attempt in range(retries):
        try:
            req = urllib.request.Request(url, headers={
                'User-Agent': 'Shakespeare-DB-Builder/2.0 (academic research)'})
            with urllib.request.urlopen(req, timeout=30) as resp:
                return resp.read().decode('utf-8')
        except Exception as e:
            if attempt < retries - 1:
                time.sleep(2)
            else:
                print(f"    ERROR: {e}")
                return None


def import_standard_ebooks(conn, cache_dir, skip_download=False):
    """Import Standard Ebooks Shakespeare plays."""
    print("\n" + "=" * 60)
    print("STEP 3: Import Standard Ebooks Plays")
    print("=" * 60)

    start = time.time()
    cache_dir = Path(cache_dir)
    cache_dir.mkdir(parents=True, exist_ok=True)

    # Create SE source + edition
    conn.execute("""
        INSERT OR IGNORE INTO sources (name, short_code, url, license, license_url,
            attribution_text, attribution_required, notes)
        VALUES ('Standard Ebooks', 'standard_ebooks', 'https://standardebooks.org',
                'CC0 1.0 Universal', 'https://creativecommons.org/publicdomain/zero/1.0/',
                'Standard Ebooks — Free and liberated ebooks. standardebooks.org', 0,
                'Public domain dedication. Based on public domain source texts.')
    """)
    source_id = conn.execute("SELECT id FROM sources WHERE short_code = 'standard_ebooks'").fetchone()[0]

    conn.execute("""
        INSERT OR IGNORE INTO editions (name, short_code, source_id, year, editors, description)
        VALUES ('Standard Ebooks Modern Edition', 'se_modern', ?, 2024,
                'Standard Ebooks editorial team', 'Carefully produced modern-spelling editions. CC0.')
    """, (source_id,))
    edition_id = conn.execute("SELECT id FROM editions WHERE short_code = 'se_modern'").fetchone()[0]
    conn.commit()

    # Build work map
    works_map = {}
    for row in conn.execute("SELECT id, oss_id, title FROM works"):
        works_map[row[1]] = (row[0], row[2])

    total_lines = 0
    total_plays = 0

    for repo_name in sorted(SE_PLAY_REPOS.keys()):
        oss_id = SE_PLAY_REPOS[repo_name]
        if oss_id not in works_map:
            continue

        work_db_id, title = works_map[oss_id]
        total_plays += 1

        cache_file = cache_dir / f'{repo_name}.json'

        # Download or use cache
        if cache_file.exists():
            with open(cache_file) as f:
                acts_data = json.load(f)
        elif skip_download:
            print(f"  [{total_plays:2d}/37] {title} — SKIPPED (no cache, --skip-download)")
            continue
        else:
            print(f"  [{total_plays:2d}/37] {title} — downloading...")
            api_url = f"https://api.github.com/repos/standardebooks/{repo_name}/contents/src/epub/text"
            listing = fetch_url(api_url)
            if not listing:
                continue
            files = json.loads(listing)
            act_files = sorted([f['name'] for f in files if f['name'].startswith('act-') and f['name'].endswith('.xhtml')])
            acts_data = {}
            for fname in act_files:
                url = f"https://raw.githubusercontent.com/standardebooks/{repo_name}/master/src/epub/text/{fname}"
                content = fetch_url(url)
                if content:
                    acts_data[fname] = content
                time.sleep(0.5)
            with open(cache_file, 'w') as f:
                json.dump(acts_data, f)

        # Parse
        all_lines = []
        for fname in sorted(acts_data.keys()):
            parser = SEPlayParser()
            try:
                parser.feed(acts_data[fname])
            except Exception as e:
                print(f"    Parse error in {fname}: {e}")
                continue
            all_lines.extend(parser.lines)

        if not all_lines:
            continue

        # Import
        conn.execute("DELETE FROM text_lines WHERE work_id = ? AND edition_id = ?", (work_db_id, edition_id))
        conn.execute("DELETE FROM text_divisions WHERE work_id = ? AND edition_id = ?", (work_db_id, edition_id))

        char_cache = {}
        for line in all_lines:
            char_name = line.get('character')
            char_id = None
            if char_name:
                if char_name not in char_cache:
                    row = conn.execute("SELECT id FROM characters WHERE work_id = ? AND UPPER(name) = UPPER(?)",
                                       (work_db_id, char_name)).fetchone()
                    if not row:
                        row = conn.execute("SELECT id FROM characters WHERE work_id = ? AND UPPER(abbrev) = UPPER(?)",
                                           (work_db_id, char_name)).fetchone()
                    char_cache[char_name] = row[0] if row else None
                char_id = char_cache[char_name]

            ct = 'stage_direction' if line.get('is_stage_direction') else 'speech'
            conn.execute("""
                INSERT INTO text_lines (work_id, edition_id, act, scene, paragraph_num,
                    character_id, char_name, content, content_type, word_count)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            """, (work_db_id, edition_id, line['act'], line['scene'], line['line_in_scene'],
                  char_id, char_name, line['text'], ct, len(line['text'].split())))

        # Divisions
        scenes = {}
        for line in all_lines:
            key = (line.get('act'), line.get('scene'))
            if key[0] is not None:
                scenes[key] = scenes.get(key, 0) + 1
        for (act, scene), count in sorted(scenes.items()):
            conn.execute("INSERT OR IGNORE INTO text_divisions (work_id, edition_id, act, scene, line_count) VALUES (?, ?, ?, ?, ?)",
                         (work_db_id, edition_id, act, scene, count))

        conn.commit()
        total_lines += len(all_lines)
        print(f"  [{total_plays:2d}/37] {title}: {len(all_lines)} lines")

    elapsed = time.time() - start
    conn.execute("INSERT INTO import_log (phase, action, details, count, duration_secs) VALUES (?, ?, ?, ?, ?)",
                 ('se_plays', 'import_complete', f'{total_plays} plays', total_lines, elapsed))
    conn.commit()
    print(f"  ✓ {total_lines:,} lines from {total_plays} plays in {elapsed:.1f}s")
    return True


def import_se_poetry(conn, cache_dir, skip_download=False):
    """Import Standard Ebooks poetry + Folger URLs."""
    print("\n" + "=" * 60)
    print("STEP 4: Import Poetry + Folger URLs")
    print("=" * 60)

    start = time.time()
    cache_dir = Path(cache_dir)
    cache_dir.mkdir(parents=True, exist_ok=True)

    edition_id = conn.execute("SELECT id FROM editions WHERE short_code = 'se_modern'").fetchone()
    if not edition_id:
        print("  ERROR: se_modern edition not found. Run step 3 first.")
        return False
    edition_id = edition_id[0]

    works = {}
    for row in conn.execute("SELECT id, oss_id, title FROM works"):
        works[row[1]] = (row[0], row[2])

    base = 'https://raw.githubusercontent.com/standardebooks/william-shakespeare_poetry/master/src/epub/text'
    total_imported = 0

    # Poetry
    poetry_cache = cache_dir / 'se-poetry.xhtml'
    if poetry_cache.exists():
        with open(poetry_cache) as f:
            poetry_html = f.read()
    elif skip_download:
        print("  Poetry — SKIPPED (no cache)")
        poetry_html = None
    else:
        print("  Downloading poetry.xhtml...")
        poetry_html = fetch_url(f"{base}/poetry.xhtml")
        if poetry_html:
            with open(poetry_cache, 'w') as f:
                f.write(poetry_html)

    if poetry_html:
        parser = PoetryXHTMLParser()
        parser.feed(poetry_html)
        for article_id, lines in parser.poems.items():
            oss_id = SE_POETRY_MAP.get(article_id)
            if not oss_id or oss_id not in works:
                continue
            work_db_id, title = works[oss_id]
            conn.execute("DELETE FROM text_lines WHERE work_id = ? AND edition_id = ?", (work_db_id, edition_id))
            for line in lines:
                conn.execute("""
                    INSERT INTO text_lines (work_id, edition_id, paragraph_num, content, content_type, word_count, stanza)
                    VALUES (?, ?, ?, ?, 'verse', ?, ?)
                """, (work_db_id, edition_id, line['line_number'], line['text'],
                      len(line['text'].split()), line['stanza']))
            total_imported += len(lines)
            conn.commit()
            print(f"  {title}: {len(lines)} lines")

    # Sonnets
    sonnets_cache = cache_dir / 'se-sonnets.xhtml'
    if sonnets_cache.exists():
        with open(sonnets_cache) as f:
            sonnets_html = f.read()
    elif skip_download:
        print("  Sonnets — SKIPPED (no cache)")
        sonnets_html = None
    else:
        print("  Downloading sonnets.xhtml...")
        sonnets_html = fetch_url(f"{base}/sonnets.xhtml")
        if sonnets_html:
            with open(sonnets_cache, 'w') as f:
                f.write(sonnets_html)

    if sonnets_html:
        sparser = SonnetParser()
        sparser.feed(sonnets_html)

        if 'sonnets' in works:
            work_db_id, title = works['sonnets']
            conn.execute("DELETE FROM text_lines WHERE work_id = ? AND edition_id = ?", (work_db_id, edition_id))
            sort_order = 0
            for snum in sorted(sparser.sonnets.keys()):
                for line in sparser.sonnets[snum]:
                    sort_order += 1
                    conn.execute("""
                        INSERT INTO text_lines (work_id, edition_id, scene, paragraph_num, content,
                            content_type, word_count, sonnet_number, stanza)
                        VALUES (?, ?, ?, ?, ?, 'verse', ?, ?, ?)
                    """, (work_db_id, edition_id, snum, line['line_number'], line['text'],
                          len(line['text'].split()), snum, line['stanza']))
            total_imported += sort_order
            conn.commit()
            print(f"  Sonnets: {sort_order} lines across {len(sparser.sonnets)} sonnets")

        if 'loverscomplaint' in works and sparser.lovers_complaint:
            work_db_id, title = works['loverscomplaint']
            conn.execute("DELETE FROM text_lines WHERE work_id = ? AND edition_id = ?", (work_db_id, edition_id))
            for line in sparser.lovers_complaint:
                conn.execute("""
                    INSERT INTO text_lines (work_id, edition_id, paragraph_num, content, content_type, word_count, stanza)
                    VALUES (?, ?, ?, ?, 'verse', ?, ?)
                """, (work_db_id, edition_id, line['line_number'], line['text'],
                      len(line['text'].split()), line['stanza']))
            total_imported += len(sparser.lovers_complaint)
            conn.commit()
            print(f"  A Lover's Complaint: {len(sparser.lovers_complaint)} lines")

    # Folger URLs
    updated = 0
    for oss_id, slug in FOLGER_SLUGS.items():
        url = f'https://www.folger.edu/explore/shakespeares-works/{slug}/'
        result = conn.execute("UPDATE works SET folger_url = ? WHERE oss_id = ?", (url, oss_id))
        if result.rowcount > 0:
            updated += 1
    conn.commit()
    print(f"  Folger URLs: {updated} works")

    elapsed = time.time() - start
    conn.execute("INSERT INTO import_log (phase, action, details, count, duration_secs) VALUES (?, ?, ?, ?, ?)",
                 ('se_poetry', 'import_complete', f'{total_imported} lines', total_imported, elapsed))
    conn.commit()
    print(f"  ✓ {total_imported:,} poetry lines in {elapsed:.1f}s")
    return True


# ============================================================
# STEP 5: Full-Text Search Indexes
# ============================================================

def build_fts(conn):
    """Rebuild all FTS indexes."""
    print("\n" + "=" * 60)
    print("STEP 5: Build Full-Text Search Indexes")
    print("=" * 60)

    start = time.time()

    # Lexicon FTS
    lexicon_count = conn.execute("SELECT COUNT(*) FROM lexicon_entries").fetchone()[0]
    if lexicon_count > 0:
        print(f"  Lexicon FTS: {lexicon_count:,} entries...")
        conn.execute("INSERT INTO lexicon_fts(lexicon_fts) VALUES('rebuild')")
        conn.commit()

    # Text FTS
    text_count = conn.execute("SELECT COUNT(*) FROM text_lines").fetchone()[0]
    if text_count > 0:
        print(f"  Text FTS: {text_count:,} lines...")
        conn.execute("INSERT INTO text_fts(text_fts) VALUES('rebuild')")
        conn.commit()

    elapsed = time.time() - start
    print(f"  ✓ FTS indexes built in {elapsed:.1f}s")


# ============================================================
# SUMMARY
# ============================================================

def print_summary(conn, db_path):
    """Print final database summary."""
    print("\n" + "=" * 60)
    print("BUILD COMPLETE")
    print("=" * 60)

    tables = [
        ('works', 'Works'),
        ('characters', 'Characters'),
        ('text_lines', 'Text lines'),
        ('text_divisions', 'Text divisions'),
        ('lexicon_entries', 'Lexicon entries'),
        ('lexicon_senses', 'Lexicon senses'),
        ('lexicon_citations', 'Lexicon citations'),
        ('sources', 'Sources'),
        ('editions', 'Editions'),
    ]

    for table, label in tables:
        count = conn.execute(f"SELECT COUNT(*) FROM [{table}]").fetchone()[0]
        print(f"  {label:25s} {count:>10,}")

    print()

    # Lines by edition
    for row in conn.execute("""
        SELECT e.name, COUNT(*) FROM text_lines t
        JOIN editions e ON t.edition_id = e.id
        GROUP BY e.id ORDER BY e.id
    """):
        print(f"  {row[0]:35s} {row[1]:>10,} lines")

    db_size = os.path.getsize(db_path)
    print(f"\n  Database: {db_path}")
    print(f"  Size: {db_size / 1024 / 1024:.1f} MB")
    print(f"  Built: {datetime.now().strftime('%Y-%m-%d %H:%M:%S UTC')}")
    print("=" * 60)


# ============================================================
# MAIN
# ============================================================

def main():
    parser = argparse.ArgumentParser(description='Build the Shakespeare database from sources')
    parser.add_argument('--output', default='build', help='Output directory (default: build/)')
    parser.add_argument('--skip-download', action='store_true', help='Skip Standard Ebooks downloads (use cache only)')
    parser.add_argument('--step', choices=['oss', 'lexicon', 'se', 'poetry', 'fts'], help='Run only one step')
    args = parser.parse_args()

    # Setup paths
    output_dir = REPO_ROOT / args.output
    output_dir.mkdir(parents=True, exist_ok=True)
    db_path = output_dir / 'shakespeare.db'
    cache_dir = output_dir / 'se-cache'

    # Remove existing DB for clean build (unless running single step)
    if not args.step and db_path.exists():
        os.remove(db_path)

    print(f"Shakespeare Database Builder")
    print(f"  Output:  {db_path}")
    print(f"  Sources: {SOURCES_DIR}")
    print()

    # Connect
    conn = sqlite3.connect(str(db_path))
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA synchronous=NORMAL")
    conn.execute("PRAGMA foreign_keys=ON")

    # Create schema
    print("Creating schema...")
    conn.executescript(SCHEMA_SQL)
    conn.commit()

    steps = {
        'oss': lambda: import_oss(conn, OSS_SQL_PATH),
        'lexicon': lambda: import_lexicon(conn, LEXICON_DIR),
        'se': lambda: import_standard_ebooks(conn, cache_dir, args.skip_download),
        'poetry': lambda: import_se_poetry(conn, cache_dir, args.skip_download),
        'fts': lambda: build_fts(conn),
    }

    if args.step:
        steps[args.step]()
    else:
        for name, fn in steps.items():
            fn()

    print_summary(conn, db_path)
    conn.close()


if __name__ == '__main__':
    main()
