#!/usr/bin/env python3
"""
Schmidt Shakespeare Lexicon — XML to SQLite importer.

Parses all downloaded lexicon XML files and imports them into a SQLite database
with full-text search support.

Usage:
    python3 tools/perseus-sqlite-builder.py [--db PATH] [--entries PATH] [--rebuild]

Options:
    --db PATH       SQLite database path (default: shakespeare_db/schmidt_lexicon.db)
    --entries PATH  Lexicon entries directory (default: perseus-schmidt/entries)
    --rebuild       Drop and recreate all tables before importing
"""

import sqlite3
import os
import sys
import re
import json
import glob
import argparse
import xml.etree.ElementTree as ET
from datetime import datetime
from pathlib import Path

# Schmidt abbreviation → (full title, Perseus text ID, work_type)
SCHMIDT_WORKS = {
    "Tp.":      ("The Tempest", "1999.03.0056", "comedy"),
    "Gent.":    ("Two Gentlemen of Verona", "1999.03.0032", "comedy"),
    "Gentl.":   ("Two Gentlemen of Verona", "1999.03.0032", "comedy"),
    "Wiv.":     ("Merry Wives of Windsor", "1999.03.0059", "comedy"),
    "Meas.":    ("Measure for Measure", "1999.03.0049", "comedy"),
    "Err.":     ("Comedy of Errors", "1999.03.0039", "comedy"),
    "Ado":      ("Much Ado About Nothing", "1999.03.0047", "comedy"),
    "LLL":      ("Love's Labour's Lost", "1999.03.0048", "comedy"),
    "Mids.":    ("Midsummer Night's Dream", "1999.03.0051", "comedy"),
    "Merch.":   ("Merchant of Venice", "1999.03.0050", "comedy"),
    "As":       ("As You Like It", "1999.03.0038", "comedy"),
    "Shr.":     ("Taming of the Shrew", "1999.03.0054", "comedy"),
    "All's":    ("All's Well That Ends Well", "1999.03.0036", "comedy"),
    "Alls":     ("All's Well That Ends Well", "1999.03.0036", "comedy"),
    "Tw.":      ("Twelfth Night", "1999.03.0057", "comedy"),
    "Wint.":    ("Winter's Tale", "1999.03.0060", "comedy"),
    "John":     ("King John", "1999.03.0033", "history"),
    "R2":       ("Richard II", "1999.03.0052", "history"),
    "H4A":      ("Henry IV Part 1", "1999.03.0041", "history"),
    "H4B":      ("Henry IV Part 2", "1999.03.0042", "history"),
    "H5":       ("Henry V", "1999.03.0043", "history"),
    "H6A":      ("Henry VI Part 1", "1999.03.0044", "history"),
    "H6B":      ("Henry VI Part 2", "1999.03.0045", "history"),
    "H6C":      ("Henry VI Part 3", "1999.03.0046", "history"),
    "R3":       ("Richard III", "1999.03.0035", "history"),
    "H8":       ("Henry VIII", "1999.03.0074", "history"),
    "Troil.":   ("Troilus and Cressida", "1999.03.0058", "comedy"),
    "Cor.":     ("Coriolanus", "1999.03.0026", "tragedy"),
    "Tit.":     ("Titus Andronicus", "1999.03.0037", "tragedy"),
    "Rom.":     ("Romeo and Juliet", "1999.03.0053", "tragedy"),
    "Tim.":     ("Timon of Athens", "1999.03.0055", "tragedy"),
    "Caes.":    ("Julius Caesar", "1999.03.0027", "tragedy"),
    "Mcb.":     ("Macbeth", "1999.03.0028", "tragedy"),
    "Hml.":     ("Hamlet", "1999.03.0031", "tragedy"),
    "Lr.":      ("King Lear", "1999.03.0029", "tragedy"),
    "Oth.":     ("Othello", "1999.03.0034", "tragedy"),
    "Ant.":     ("Antony and Cleopatra", "1999.03.0025", "tragedy"),
    "Cymb.":    ("Cymbeline", "1999.03.0040", "comedy"),
    "Per.":     ("Pericles", "1999.03.0030", "comedy"),
    "Ven.":     ("Venus and Adonis", "1999.03.0061", "poem"),
    "Lucr.":    ("Rape of Lucrece", "1999.03.0062", "poem"),
    "Sonn.":    ("Sonnets", "1999.03.0064", "sonnet_sequence"),
    "Pilgr.":   ("Passionate Pilgrim", "1999.03.0063", "poem"),
    "Phoen.":   ("Phoenix and the Turtle", "1999.03.0066", "poem"),
    "Compl.":   ("Lover's Complaint", "1999.03.0065", "poem"),
    # Additional abbreviations found in citations
    "Tp":       ("The Tempest", "1999.03.0056", "comedy"),
    "Wiv":      ("Merry Wives of Windsor", "1999.03.0059", "comedy"),
    "Meas":     ("Measure for Measure", "1999.03.0049", "comedy"),
    "Err":      ("Comedy of Errors", "1999.03.0039", "comedy"),
    "Mids":     ("Midsummer Night's Dream", "1999.03.0051", "comedy"),
    "Merch":    ("Merchant of Venice", "1999.03.0050", "comedy"),
    "Shr":      ("Taming of the Shrew", "1999.03.0054", "comedy"),
    "Tw":       ("Twelfth Night", "1999.03.0057", "comedy"),
    "Wint":     ("Winter's Tale", "1999.03.0060", "comedy"),
    "Troil":    ("Troilus and Cressida", "1999.03.0058", "comedy"),
    "Cor":      ("Coriolanus", "1999.03.0026", "tragedy"),
    "Tit":      ("Titus Andronicus", "1999.03.0037", "tragedy"),
    "Rom":      ("Romeo and Juliet", "1999.03.0053", "tragedy"),
    "Tim":      ("Timon of Athens", "1999.03.0055", "tragedy"),
    "Caes":     ("Julius Caesar", "1999.03.0027", "tragedy"),
    "Mcb":      ("Macbeth", "1999.03.0028", "tragedy"),
    "Hml":      ("Hamlet", "1999.03.0031", "tragedy"),
    "Lr":       ("King Lear", "1999.03.0029", "tragedy"),
    "Oth":      ("Othello", "1999.03.0034", "tragedy"),
    "Ant":      ("Antony and Cleopatra", "1999.03.0025", "tragedy"),
    "Cymb":     ("Cymbeline", "1999.03.0040", "comedy"),
    "Per":      ("Pericles", "1999.03.0030", "comedy"),
    "Ven":      ("Venus and Adonis", "1999.03.0061", "poem"),
    "Lucr":     ("Rape of Lucrece", "1999.03.0062", "poem"),
    "Sonn":     ("Sonnets", "1999.03.0064", "sonnet_sequence"),
    "Pilgr":    ("Passionate Pilgrim", "1999.03.0063", "poem"),
    "Phoen":    ("Phoenix and the Turtle", "1999.03.0066", "poem"),
    "Compl":    ("Lover's Complaint", "1999.03.0065", "poem"),
}

# Perseus work code → Schmidt abbreviation (for parsing bibl n= attributes)
PERSEUS_TO_SCHMIDT = {
    "tmp": "Tp.",
    "tgv": "Gentl.",
    "wiv": "Wiv.",
    "mm": "Meas.",
    "err": "Err.",
    "ado": "Ado",
    "lll": "LLL",
    "mnd": "Mids.",
    "mv": "Merch.",
    "ayl": "As",
    "shr": "Shr.",
    "aww": "All's",
    "tn": "Tw.",
    "wt": "Wint.",
    "jn": "John",
    "r2": "R2",
    "1h4": "H4A",
    "2h4": "H4B",
    "h5": "H5",
    "1h6": "H6A",
    "2h6": "H6B",
    "3h6": "H6C",
    "r3": "R3",
    "h8": "H8",
    "tro": "Troil.",
    "cor": "Cor.",
    "tit": "Tit.",
    "rom": "Rom.",
    "tim": "Tim.",
    "jc": "Caes.",
    "mac": "Mcb.",
    "ham": "Hml.",
    "lr": "Lr.",
    "oth": "Oth.",
    "ant": "Ant.",
    "cym": "Cymb.",
    "per": "Per.",
    "ven": "Ven.",
    "luc": "Lucr.",
    "son": "Sonn.",
    "pp": "Pilgr.",
    "phoe": "Phoen.",
    "lc": "Compl.",
}


def create_tables(conn, rebuild=False):
    """Create the database schema."""
    c = conn.cursor()
    
    if rebuild:
        c.executescript("""
            DROP TABLE IF EXISTS lexicon_fts;
            DROP TABLE IF EXISTS citation_matches;
            DROP TABLE IF EXISTS lexicon_citations;
            DROP TABLE IF EXISTS lexicon_senses;
            DROP TABLE IF EXISTS lexicon_entries;
            DROP TABLE IF EXISTS works;
            DROP TABLE IF EXISTS editions;
            DROP TABLE IF EXISTS text_lines;
            DROP TABLE IF EXISTS text_fts;
            DROP TABLE IF EXISTS text_divisions;
            DROP TABLE IF EXISTS line_mappings;
            DROP TABLE IF EXISTS citation_corrections;
            DROP TABLE IF EXISTS import_log;
        """)
    
    c.executescript("""
        -- Core lexicon entry
        CREATE TABLE IF NOT EXISTS lexicon_entries (
            id INTEGER PRIMARY KEY,
            key TEXT NOT NULL,
            letter TEXT NOT NULL,
            entry_type TEXT DEFAULT 'main',
            orthography TEXT,
            full_text TEXT NOT NULL,
            raw_xml TEXT NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
        CREATE INDEX IF NOT EXISTS idx_lexicon_key ON lexicon_entries(key);
        CREATE INDEX IF NOT EXISTS idx_lexicon_letter ON lexicon_entries(letter);

        -- Individual senses/definitions within an entry
        CREATE TABLE IF NOT EXISTS lexicon_senses (
            id INTEGER PRIMARY KEY,
            entry_id INTEGER REFERENCES lexicon_entries(id) ON DELETE CASCADE,
            sense_number INTEGER,
            definition_text TEXT NOT NULL,
            UNIQUE(entry_id, sense_number)
        );
        CREATE INDEX IF NOT EXISTS idx_senses_entry ON lexicon_senses(entry_id);

        -- Every citation in the lexicon
        CREATE TABLE IF NOT EXISTS lexicon_citations (
            id INTEGER PRIMARY KEY,
            entry_id INTEGER REFERENCES lexicon_entries(id) ON DELETE CASCADE,
            sense_id INTEGER REFERENCES lexicon_senses(id),
            work_abbrev TEXT,
            work_id INTEGER,
            act INTEGER,
            scene INTEGER,
            line INTEGER,
            raw_citation TEXT NOT NULL,
            perseus_ref TEXT,
            quote_text TEXT,
            display_text TEXT
        );
        CREATE INDEX IF NOT EXISTS idx_citations_entry ON lexicon_citations(entry_id);
        CREATE INDEX IF NOT EXISTS idx_citations_work ON lexicon_citations(work_abbrev);
        CREATE INDEX IF NOT EXISTS idx_citations_location ON lexicon_citations(work_abbrev, act, scene, line);

        -- Shakespeare works master list
        CREATE TABLE IF NOT EXISTS works (
            id INTEGER PRIMARY KEY,
            title TEXT NOT NULL,
            full_title TEXT,
            schmidt_abbrev TEXT NOT NULL UNIQUE,
            perseus_id TEXT,
            work_type TEXT NOT NULL,
            date_composed TEXT,
            date_first_published TEXT,
            folger_url TEXT,
            notes TEXT
        );

        -- Edition/version tracking
        CREATE TABLE IF NOT EXISTS editions (
            id INTEGER PRIMARY KEY,
            name TEXT NOT NULL,
            short_code TEXT NOT NULL UNIQUE,
            year INTEGER,
            editors TEXT,
            source TEXT NOT NULL,
            source_id TEXT,
            notes TEXT
        );

        -- Actual text content, line by line (Phase 2)
        CREATE TABLE IF NOT EXISTS text_lines (
            id INTEGER PRIMARY KEY,
            work_id INTEGER REFERENCES works(id),
            edition_id INTEGER REFERENCES editions(id),
            act INTEGER,
            scene INTEGER,
            line_number INTEGER NOT NULL,
            speaker TEXT,
            content TEXT NOT NULL,
            content_type TEXT DEFAULT 'verse',
            is_stage_direction BOOLEAN DEFAULT 0,
            sonnet_number INTEGER,
            stanza INTEGER,
            raw_xml TEXT,
            UNIQUE(work_id, edition_id, act, scene, line_number)
        );

        -- Cross-edition line concordance (Phase 3)
        CREATE TABLE IF NOT EXISTS line_mappings (
            id INTEGER PRIMARY KEY,
            work_id INTEGER REFERENCES works(id),
            act INTEGER,
            scene INTEGER,
            globe_line INTEGER,
            f1_line INTEGER,
            tln INTEGER,
            notes TEXT,
            UNIQUE(work_id, act, scene, globe_line)
        );

        -- Resolved citation-to-text links (Phase 3)
        CREATE TABLE IF NOT EXISTS citation_matches (
            id INTEGER PRIMARY KEY,
            citation_id INTEGER REFERENCES lexicon_citations(id),
            text_line_id INTEGER REFERENCES text_lines(id),
            edition_id INTEGER REFERENCES editions(id),
            match_type TEXT NOT NULL,
            confidence REAL DEFAULT 1.0,
            matched_text TEXT,
            notes TEXT
        );

        -- Citation corrections for edition differences (Phase 5)
        CREATE TABLE IF NOT EXISTS citation_corrections (
            id INTEGER PRIMARY KEY,
            citation_id INTEGER REFERENCES lexicon_citations(id),
            edition_id INTEGER REFERENCES editions(id),
            original_line INTEGER,
            corrected_line INTEGER,
            correction_type TEXT,
            notes TEXT,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );

        -- Structural divisions (Phase 2)
        CREATE TABLE IF NOT EXISTS text_divisions (
            id INTEGER PRIMARY KEY,
            work_id INTEGER REFERENCES works(id),
            edition_id INTEGER REFERENCES editions(id),
            act INTEGER NOT NULL,
            scene INTEGER,
            title TEXT,
            first_line_id INTEGER,
            last_line_id INTEGER,
            line_count INTEGER
        );

        -- Import tracking
        CREATE TABLE IF NOT EXISTS import_log (
            id INTEGER PRIMARY KEY,
            phase TEXT NOT NULL,
            action TEXT NOT NULL,
            details TEXT,
            count INTEGER DEFAULT 0,
            timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
    """)
    
    # FTS tables
    c.executescript("""
        CREATE VIRTUAL TABLE IF NOT EXISTS lexicon_fts USING fts5(
            key,
            full_text,
            content=lexicon_entries,
            content_rowid=id,
            tokenize='porter unicode61'
        );

        CREATE VIRTUAL TABLE IF NOT EXISTS text_fts USING fts5(
            content,
            speaker,
            content=text_lines,
            content_rowid=id,
            tokenize='porter unicode61'
        );
    """)
    
    conn.commit()


def populate_works(conn):
    """Insert the Shakespeare works master list."""
    c = conn.cursor()
    
    # Deduplicate — some abbreviations map to the same work
    seen = set()
    works = []
    for abbrev, (title, perseus_id, work_type) in SCHMIDT_WORKS.items():
        # Use the "canonical" abbreviation (with period if applicable)
        if title not in seen:
            seen.add(title)
            # Find the canonical abbreviation (prefer with period)
            canonical = abbrev
            for alt, (t, _, _) in SCHMIDT_WORKS.items():
                if t == title and '.' in alt:
                    canonical = alt
                    break
            works.append((title, canonical, perseus_id, work_type))
    
    for title, abbrev, perseus_id, work_type in works:
        c.execute("""
            INSERT OR IGNORE INTO works (title, schmidt_abbrev, perseus_id, work_type)
            VALUES (?, ?, ?, ?)
        """, (title, abbrev, perseus_id, work_type))
    
    conn.commit()
    count = c.execute("SELECT COUNT(*) FROM works").fetchone()[0]
    print(f"  Works table: {count} entries")
    return count


def populate_editions(conn):
    """Insert known editions."""
    c = conn.cursor()
    editions = [
        ("Globe Edition", "Globe", 1864, "Clark & Wright", "perseus", None,
         "Standard Victorian reference edition. Schmidt's citation base."),
        ("First Folio", "F1", 1623, "Heminge & Condell", "perseus", None,
         "Original collected works. Perseus has dual numbering."),
        ("Second Quarto", "Q2", None, None, "reference", None,
         "Various quarto editions referenced by Schmidt."),
        ("First Quarto", "Q1", None, None, "reference", None,
         "Various first quarto editions."),
    ]
    for name, code, year, editors, source, source_id, notes in editions:
        c.execute("""
            INSERT OR IGNORE INTO editions (name, short_code, year, editors, source, source_id, notes)
            VALUES (?, ?, ?, ?, ?, ?, ?)
        """, (name, code, year, editors, source, source_id, notes))
    conn.commit()


def get_element_text(elem):
    """Recursively get all text content from an element and its children."""
    parts = []
    if elem.text:
        parts.append(elem.text)
    for child in elem:
        parts.append(get_element_text(child))
        if child.tail:
            parts.append(child.tail)
    return ''.join(parts)


def parse_perseus_ref(bibl_n):
    """
    Parse a Perseus bibl n= attribute like 'shak. ham 3.1.56' into components.
    Returns (schmidt_abbrev, act, scene, line, perseus_ref) or None.
    """
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
    
    # Parse numbers — could be "3.1.56" or "312" or "18.1"
    num_parts = numbers.split('.')
    act = scene = line = None
    
    try:
        if len(num_parts) == 3:
            act = int(num_parts[0])
            scene = int(num_parts[1])
            line = int(num_parts[2])
        elif len(num_parts) == 2:
            # Could be sonnet.line or act.scene
            act = int(num_parts[0])
            line = int(num_parts[1])
        elif len(num_parts) == 1 and num_parts[0]:
            line = int(num_parts[0])
    except ValueError:
        pass
    
    return (schmidt_abbrev, act, scene, line, bibl_n)


def parse_entry_xml(xml_path):
    """
    Parse a single lexicon entry XML file.
    Returns dict with parsed data or None on failure.
    """
    try:
        with open(xml_path, 'r', encoding='utf-8') as f:
            raw_xml = f.read()
    except Exception as e:
        return None
    
    try:
        root = ET.fromstring(raw_xml)
    except ET.ParseError:
        # Try to fix common XML issues
        try:
            # Wrap in a root if needed
            raw_xml_fixed = raw_xml.replace('&', '&amp;') if '&' in raw_xml and '&amp;' not in raw_xml else raw_xml
            root = ET.fromstring(raw_xml_fixed)
        except:
            return None
    
    # Find the entryFree element
    entry_free = root.find('.//entryFree')
    if entry_free is None:
        return None
    
    key = entry_free.get('key', '')
    entry_type = entry_free.get('type', 'main')
    
    # Get letter from parent div1
    div1 = root.find('.//div1')
    letter = div1.get('n', '') if div1 is not None else ''
    if not letter:
        letter = os.path.basename(os.path.dirname(xml_path))
    
    # Get orthography
    orth_elem = entry_free.find('orth')
    orthography = get_element_text(orth_elem) if orth_elem is not None else key
    
    # Get full text (all text content, stripped of tags)
    full_text = get_element_text(entry_free).strip()
    # Clean up whitespace
    full_text = re.sub(r'\s+', ' ', full_text)
    
    # Parse senses (split on numbered patterns after <lb/> elements)
    senses = parse_senses(full_text)
    
    # Parse all citations
    citations = []
    for bibl in entry_free.iter('bibl'):
        bibl_n = bibl.get('n', '')
        display_text = get_element_text(bibl).strip()
        
        # Check if this bibl is inside a <cit> (has associated quote)
        quote_text = None
        parent_cit = None
        # Walk up to find parent cit — ET doesn't support parent traversal easily,
        # so we search for cit elements containing this bibl
        for cit in entry_free.iter('cit'):
            if bibl in list(cit.iter('bibl')):
                quote_elem = cit.find('quote')
                if quote_elem is not None:
                    quote_text = get_element_text(quote_elem).strip()
                break
        
        parsed = parse_perseus_ref(bibl_n)
        if parsed:
            schmidt_abbrev, act, scene, line, perseus_ref = parsed
            citations.append({
                'work_abbrev': schmidt_abbrev,
                'act': act,
                'scene': scene,
                'line': line,
                'raw_citation': display_text,
                'perseus_ref': perseus_ref,
                'quote_text': quote_text,
                'display_text': display_text,
            })
        elif display_text:
            # Citation without Perseus ref — still record it
            citations.append({
                'work_abbrev': None,
                'act': None,
                'scene': None,
                'line': None,
                'raw_citation': display_text,
                'perseus_ref': bibl_n if bibl_n else None,
                'quote_text': quote_text,
                'display_text': display_text,
            })
    
    return {
        'key': key,
        'letter': letter,
        'entry_type': entry_type,
        'orthography': orthography,
        'full_text': full_text,
        'raw_xml': raw_xml,
        'senses': senses,
        'citations': citations,
    }


def parse_senses(full_text):
    """
    Split the full text into numbered senses.
    Schmidt uses patterns like: 1) definition... 2) definition...
    Separated by line breaks in the XML (which become spaces in full_text).
    """
    senses = []
    
    # Pattern: digit + closing paren, e.g. "1)" "2)" "12)"
    sense_pattern = re.compile(r'(?:^|\s)(\d+)\)\s')
    matches = list(sense_pattern.finditer(full_text))
    
    if not matches:
        # Single sense entry — the whole text is one sense
        return [{'number': 1, 'text': full_text}]
    
    for i, match in enumerate(matches):
        sense_num = int(match.group(1))
        start = match.end()
        end = matches[i + 1].start() if i + 1 < len(matches) else len(full_text)
        sense_text = full_text[start:end].strip()
        senses.append({'number': sense_num, 'text': sense_text})
    
    return senses


def import_entries(conn, entries_dir):
    """Import all XML entries from the entries directory."""
    c = conn.cursor()
    
    total_entries = 0
    total_citations = 0
    total_senses = 0
    errors = 0
    
    # Process each letter directory
    letter_dirs = sorted(glob.glob(os.path.join(entries_dir, '*')))
    
    for letter_dir in letter_dirs:
        if not os.path.isdir(letter_dir):
            continue
        
        letter = os.path.basename(letter_dir)
        xml_files = sorted(glob.glob(os.path.join(letter_dir, '*.xml')))
        
        if not xml_files:
            continue
        
        print(f"  Processing letter {letter}: {len(xml_files)} files...")
        
        letter_entries = 0
        letter_citations = 0
        
        for xml_path in xml_files:
            entry = parse_entry_xml(xml_path)
            if entry is None:
                errors += 1
                continue
            
            # Insert entry
            c.execute("""
                INSERT INTO lexicon_entries (key, letter, entry_type, orthography, full_text, raw_xml)
                VALUES (?, ?, ?, ?, ?, ?)
            """, (
                entry['key'], entry['letter'], entry['entry_type'],
                entry['orthography'], entry['full_text'], entry['raw_xml']
            ))
            entry_id = c.lastrowid
            
            # Insert senses
            for sense in entry['senses']:
                c.execute("""
                    INSERT OR IGNORE INTO lexicon_senses (entry_id, sense_number, definition_text)
                    VALUES (?, ?, ?)
                """, (entry_id, sense['number'], sense['text']))
                total_senses += 1
            
            # Insert citations
            for cit in entry['citations']:
                c.execute("""
                    INSERT INTO lexicon_citations 
                    (entry_id, work_abbrev, act, scene, line, raw_citation, perseus_ref, quote_text, display_text)
                    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
                """, (
                    entry_id, cit['work_abbrev'], cit['act'], cit['scene'], cit['line'],
                    cit['raw_citation'], cit['perseus_ref'], cit['quote_text'], cit['display_text']
                ))
                letter_citations += 1
            
            letter_entries += 1
        
        # Commit after each letter
        conn.commit()
        total_entries += letter_entries
        total_citations += letter_citations
        print(f"    → {letter_entries} entries, {letter_citations} citations")
    
    return total_entries, total_citations, total_senses, errors


def rebuild_fts(conn):
    """Rebuild the full-text search index."""
    c = conn.cursor()
    print("  Rebuilding FTS index...")
    c.execute("INSERT INTO lexicon_fts(lexicon_fts) VALUES('rebuild')")
    conn.commit()


def link_citations_to_works(conn):
    """Set work_id on citations based on work_abbrev."""
    c = conn.cursor()
    
    # Build abbrev → work_id mapping
    works = c.execute("SELECT id, schmidt_abbrev FROM works").fetchall()
    abbrev_to_id = {}
    for work_id, abbrev in works:
        abbrev_to_id[abbrev] = work_id
        # Also map without period
        abbrev_to_id[abbrev.rstrip('.')] = work_id
    
    # Also add all SCHMIDT_WORKS keys
    for abbrev, (title, _, _) in SCHMIDT_WORKS.items():
        for wid, wabbrev in works:
            if SCHMIDT_WORKS.get(wabbrev, (None,))[0] == title:
                abbrev_to_id[abbrev] = wid
                break
    
    updated = 0
    for abbrev, work_id in abbrev_to_id.items():
        result = c.execute("""
            UPDATE lexicon_citations SET work_id = ? WHERE work_abbrev = ? AND work_id IS NULL
        """, (work_id, abbrev))
        updated += result.rowcount
    
    conn.commit()
    print(f"  Linked {updated} citations to works")
    return updated


def print_stats(conn):
    """Print database statistics."""
    c = conn.cursor()
    
    print("\n" + "="*60)
    print("DATABASE STATISTICS")
    print("="*60)
    
    entries = c.execute("SELECT COUNT(*) FROM lexicon_entries").fetchone()[0]
    senses = c.execute("SELECT COUNT(*) FROM lexicon_senses").fetchone()[0]
    citations = c.execute("SELECT COUNT(*) FROM lexicon_citations").fetchone()[0]
    linked = c.execute("SELECT COUNT(*) FROM lexicon_citations WHERE work_id IS NOT NULL").fetchone()[0]
    unlinked = c.execute("SELECT COUNT(*) FROM lexicon_citations WHERE work_id IS NULL").fetchone()[0]
    works = c.execute("SELECT COUNT(*) FROM works").fetchone()[0]
    
    print(f"  Lexicon entries:    {entries:,}")
    print(f"  Senses:             {senses:,}")
    print(f"  Citations:          {citations:,}")
    print(f"    Linked to works:  {linked:,}")
    print(f"    Unlinked:         {unlinked:,}")
    print(f"  Works:              {works:,}")
    
    # Citations per work
    print(f"\n  Citations by work:")
    rows = c.execute("""
        SELECT w.schmidt_abbrev, w.title, COUNT(lc.id) as cnt
        FROM lexicon_citations lc
        JOIN works w ON lc.work_id = w.id
        GROUP BY w.id
        ORDER BY cnt DESC
        LIMIT 15
    """).fetchall()
    for abbrev, title, cnt in rows:
        print(f"    {abbrev:<8} {title:<35} {cnt:>6}")
    
    # Entries per letter
    print(f"\n  Entries by letter:")
    rows = c.execute("""
        SELECT letter, COUNT(*) as cnt FROM lexicon_entries GROUP BY letter ORDER BY letter
    """).fetchall()
    for letter, cnt in rows:
        print(f"    {letter}: {cnt}")
    
    # FTS test
    print(f"\n  FTS test — searching 'love':")
    rows = c.execute("""
        SELECT le.key, substr(le.full_text, 1, 80)
        FROM lexicon_fts fts
        JOIN lexicon_entries le ON le.id = fts.rowid
        WHERE fts.key MATCH 'love'
        LIMIT 5
    """).fetchall()
    for key, text in rows:
        print(f"    {key}: {text}...")
    
    print("="*60)
    
    # Log the import
    c.execute("""
        INSERT INTO import_log (phase, action, details, count)
        VALUES ('phase1', 'import_complete', ?, ?)
    """, (json.dumps({
        'entries': entries, 'senses': senses, 'citations': citations,
        'linked': linked, 'unlinked': unlinked
    }), entries))
    conn.commit()


def main():
    parser = argparse.ArgumentParser(description='Import Schmidt Lexicon XMLs into SQLite')
    parser.add_argument('--db', default='shakespeare_db/schmidt_lexicon.db',
                        help='SQLite database path')
    parser.add_argument('--entries', default='perseus-schmidt/entries',
                        help='Lexicon entries directory')
    parser.add_argument('--rebuild', action='store_true',
                        help='Drop and recreate all tables')
    args = parser.parse_args()
    
    # Ensure output directory exists
    os.makedirs(os.path.dirname(args.db) or '.', exist_ok=True)
    
    # Check entries directory
    if not os.path.isdir(args.entries):
        print(f"Error: entries directory not found: {args.entries}")
        sys.exit(1)
    
    xml_count = len(glob.glob(os.path.join(args.entries, '*', '*.xml')))
    print(f"Schmidt Shakespeare Lexicon — SQLite Builder")
    print(f"  Database: {args.db}")
    print(f"  Entries:  {args.entries} ({xml_count} XML files)")
    print(f"  Rebuild:  {args.rebuild}")
    print()
    
    conn = sqlite3.connect(args.db)
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA synchronous=NORMAL")
    conn.execute("PRAGMA foreign_keys=ON")
    
    print("Creating tables...")
    create_tables(conn, rebuild=args.rebuild)
    
    print("Populating works table...")
    populate_works(conn)
    
    print("Populating editions table...")
    populate_editions(conn)
    
    print(f"\nImporting {xml_count} lexicon entries...")
    entries, citations, senses, errors = import_entries(conn, args.entries)
    print(f"\n  Total: {entries} entries, {citations} citations, {senses} senses, {errors} errors")
    
    print("\nLinking citations to works...")
    link_citations_to_works(conn)
    
    print("\nRebuilding full-text search index...")
    rebuild_fts(conn)
    
    print_stats(conn)
    
    conn.close()
    
    db_size = os.path.getsize(args.db)
    print(f"\nDatabase size: {db_size / 1024 / 1024:.1f} MB")
    print("Done.")


if __name__ == '__main__':
    main()
