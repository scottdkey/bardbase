#!/usr/bin/env python3
"""
Import Standard Ebooks Shakespeare poetry into shakespeare.db
- Sonnets (154)
- Venus and Adonis
- Rape of Lucrece
- The Passionate Pilgrim
- The Phoenix and the Turtle
- A Lover's Complaint (in sonnets.xhtml)
"""

import os
import sys
import re
import sqlite3
import urllib.request
from html.parser import HTMLParser

WORKSPACE = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DB_PATH = os.path.join(WORKSPACE, 'shakespeare_db', 'shakespeare.db')
CACHE_DIR = os.path.join(WORKSPACE, 'shakespeare_db', 'standard-ebooks-cache')

# Map SE article IDs to our work oss_ids
POETRY_MAP = {
    'venus-and-adonis': 'venusadonis',
    'the-rape-of-lucrece': 'rapelucrece',
    'the-passionate-pilgrim': 'passionatepilgrim',
    'the-pheonix-and-the-turtle': 'phoenixturtle',  # Note: SE has typo "pheonix"
}

# Sonnets and Lover's Complaint are in sonnets.xhtml
SONNETS_WORK = 'sonnets'
LOVERS_COMPLAINT_WORK = 'loverscomplaint'


class PoetryXHTMLParser(HTMLParser):
    """Parse SE poetry XHTML into structured data."""

    def __init__(self):
        super().__init__()
        self.poems = {}  # article_id -> list of lines
        self.current_article = None
        self.current_section = None
        self.buf = ''
        self.in_span = False
        self.in_header = False
        self.in_dedication = False
        self.in_p = False
        self.line_counter = 0
        self.stanza_counter = 0
        self.tag_stack = []

    def _epub_type(self, attrs):
        return dict(attrs).get('epub:type', '')

    def handle_starttag(self, tag, attrs):
        self.tag_stack.append(tag)
        ad = dict(attrs)
        et = self._epub_type(attrs)

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
            self.current_section = sid

        elif tag in ('h2', 'h3', 'h4', 'header', 'hgroup'):
            self.in_header = True

        elif tag == 'span' and self.current_article and not self.in_header and not self.in_dedication:
            self.in_span = True
            self.buf = ''

        elif tag == 'p' and self.current_article and not self.in_header and not self.in_dedication:
            self.in_p = True
            self.stanza_counter += 1

    def handle_endtag(self, tag):
        if self.tag_stack and self.tag_stack[-1] == tag:
            self.tag_stack.pop()

        if tag == 'span' and self.in_span:
            text = self.buf.strip()
            if text and self.current_article:
                self.line_counter += 1
                self.poems[self.current_article].append({
                    'text': text,
                    'line_number': self.line_counter,
                    'stanza': self.stanza_counter,
                })
            self.in_span = False
            self.buf = ''

        elif tag == 'p' and self.in_p:
            self.in_p = False

        elif tag == 'article':
            self.current_article = None
            self.current_section = None
            self.in_dedication = False

        elif tag == 'section':
            if self.in_dedication:
                self.in_dedication = False

        elif tag in ('h2', 'h3', 'h4', 'header', 'hgroup'):
            self.in_header = False

    def handle_data(self, data):
        if self.in_span:
            self.buf += data

    def handle_entityref(self, name):
        c = {'amp': '&', 'lt': '<', 'gt': '>', 'nbsp': ' ', 'quot': '"'}.get(name, f'&{name};')
        if self.in_span:
            self.buf += c

    def handle_charref(self, name):
        try:
            c = chr(int(name[1:], 16)) if name.startswith('x') else chr(int(name))
        except ValueError:
            c = f'&#{name};'
        if self.in_span:
            self.buf += c


class SonnetParser(HTMLParser):
    """Parse SE sonnets XHTML — each sonnet is an <article>."""

    def __init__(self):
        super().__init__()
        self.sonnets = {}  # sonnet_number -> list of lines
        self.lovers_complaint = []  # A Lover's Complaint lines
        self.current_article = None
        self.current_sonnet_num = None
        self.is_lovers_complaint = False
        self.buf = ''
        self.in_span = False
        self.in_header = False
        self.line_counter = 0
        self.stanza_counter = 0
        self.in_p = False
        self.tag_stack = []

    def handle_starttag(self, tag, attrs):
        self.tag_stack.append(tag)
        ad = dict(attrs)
        et = dict(attrs).get('epub:type', '')

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
                entry = {
                    'text': text,
                    'line_number': self.line_counter,
                    'stanza': self.stanza_counter,
                }

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
        c = {'amp': '&', 'lt': '<', 'gt': '>', 'nbsp': ' '}.get(name, f'&{name};')
        if self.in_span:
            self.buf += c

    def handle_charref(self, name):
        try:
            c = chr(int(name[1:], 16)) if name.startswith('x') else chr(int(name))
        except ValueError:
            c = f'&#{name};'
        if self.in_span:
            self.buf += c


def fetch_or_cache(url, cache_name):
    """Fetch URL with caching."""
    os.makedirs(CACHE_DIR, exist_ok=True)
    cache_path = os.path.join(CACHE_DIR, cache_name)

    if os.path.exists(cache_path):
        with open(cache_path) as f:
            return f.read()

    req = urllib.request.Request(url, headers={'User-Agent': 'Shakespeare-DB/1.0'})
    with urllib.request.urlopen(req, timeout=30) as resp:
        content = resp.read().decode('utf-8')

    with open(cache_path, 'w') as f:
        f.write(content)

    return content


def main():
    conn = sqlite3.connect(DB_PATH)
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA foreign_keys=ON")

    # Get edition ID for se_modern
    edition_id = conn.execute("SELECT id FROM editions WHERE short_code = 'se_modern'").fetchone()
    if not edition_id:
        print("ERROR: se_modern edition not found. Run standard-ebooks-importer.py first.")
        sys.exit(1)
    edition_id = edition_id[0]

    # Get works map
    works = {}
    for row in conn.execute("SELECT id, oss_id, title FROM works").fetchall():
        works[row[1]] = (row[0], row[2])

    base = 'https://raw.githubusercontent.com/standardebooks/william-shakespeare_poetry/master/src/epub/text'

    # ========== POETRY.XHTML ==========
    print("=== Downloading poetry.xhtml ===")
    poetry_html = fetch_or_cache(f"{base}/poetry.xhtml", "se-poetry.xhtml")

    parser = PoetryXHTMLParser()
    parser.feed(poetry_html)

    total_imported = 0

    for article_id, lines in parser.poems.items():
        oss_id = POETRY_MAP.get(article_id)
        if not oss_id or oss_id not in works:
            print(f"  [SKIP] {article_id} → {oss_id} (not mapped)")
            continue

        work_db_id, title = works[oss_id]
        print(f"\n  {title}: {len(lines)} lines")

        # Clear existing
        conn.execute("DELETE FROM text_lines WHERE work_id = ? AND edition_id = ?", (work_db_id, edition_id))

        for i, line in enumerate(lines):
            conn.execute("""
                INSERT INTO text_lines (work_id, edition_id, act, scene, paragraph_num, char_name, content, content_type, word_count, stanza)
                VALUES (?, ?, NULL, NULL, ?, NULL, ?, 'verse', ?, ?)
            """, (
                work_db_id, edition_id,
                line['line_number'],
                line['text'],
                len(line['text'].split()),
                line['stanza']
            ))
        total_imported += len(lines)
        conn.commit()
        print(f"    ✓ {len(lines)} lines, {max(l['stanza'] for l in lines)} stanzas")

    # ========== SONNETS.XHTML ==========
    print("\n=== Downloading sonnets.xhtml ===")
    sonnets_html = fetch_or_cache(f"{base}/sonnets.xhtml", "se-sonnets.xhtml")

    sparser = SonnetParser()
    sparser.feed(sonnets_html)

    print(f"\n  Found {len(sparser.sonnets)} sonnets, Lover's Complaint: {len(sparser.lovers_complaint)} lines")

    # Import sonnets
    if 'sonnets' in works:
        work_db_id, title = works['sonnets']
        conn.execute("DELETE FROM text_lines WHERE work_id = ? AND edition_id = ?", (work_db_id, edition_id))

        sort_order = 0
        for sonnet_num in sorted(sparser.sonnets.keys()):
            lines = sparser.sonnets[sonnet_num]
            for line in lines:
                sort_order += 1
                conn.execute("""
                    INSERT INTO text_lines (work_id, edition_id, act, scene, paragraph_num, char_name, content, content_type, word_count, sonnet_number, stanza)
                    VALUES (?, ?, NULL, ?, ?, NULL, ?, 'verse', ?, ?, ?)
                """, (
                    work_db_id, edition_id,
                    sonnet_num,
                    line['line_number'],
                    line['text'],
                    len(line['text'].split()),
                    sonnet_num,
                    line['stanza']
                ))

        total_imported += sort_order
        conn.commit()
        print(f"  ✓ Sonnets: {sort_order} lines across {len(sparser.sonnets)} sonnets")

    # Import Lover's Complaint
    if 'loverscomplaint' in works and sparser.lovers_complaint:
        work_db_id, title = works['loverscomplaint']
        conn.execute("DELETE FROM text_lines WHERE work_id = ? AND edition_id = ?", (work_db_id, edition_id))

        for line in sparser.lovers_complaint:
            conn.execute("""
                INSERT INTO text_lines (work_id, edition_id, act, scene, paragraph_num, char_name, content, content_type, word_count, stanza)
                VALUES (?, ?, NULL, NULL, ?, NULL, ?, 'verse', ?, ?)
            """, (
                work_db_id, edition_id,
                line['line_number'],
                line['text'],
                len(line['text'].split()),
                line['stanza']
            ))

        total_imported += len(sparser.lovers_complaint)
        conn.commit()
        print(f"  ✓ A Lover's Complaint: {len(sparser.lovers_complaint)} lines")

    # ========== FOLGER REFERENCE URLs ==========
    print("\n=== Building Folger Reference URLs ===")

    # Folger URL patterns - map oss_id to Folger slug
    folger_slugs = {
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

    updated = 0
    for oss_id, slug in folger_slugs.items():
        url = f'https://www.folger.edu/explore/shakespeares-works/{slug}/'
        result = conn.execute("UPDATE works SET folger_url = ? WHERE oss_id = ?", (url, oss_id))
        if result.rowcount > 0:
            updated += 1

    conn.commit()
    print(f"  ✓ {updated} Folger reference URLs set")

    # ========== SUMMARY ==========
    print("\n" + "=" * 60)
    se_total = conn.execute("SELECT COUNT(*) FROM text_lines WHERE edition_id = ?", (edition_id,)).fetchone()[0]
    all_total = conn.execute("SELECT COUNT(*) FROM text_lines").fetchone()[0]
    db_size = os.path.getsize(DB_PATH) / 1024 / 1024

    print(f"Poetry import complete: {total_imported} new lines")
    print(f"\nTotal SE lines: {se_total:,}")
    print(f"Total all lines: {all_total:,}")
    print(f"Database size: {db_size:.1f} MB")

    # Show all works with SE text
    print(f"\nWorks with SE text ({conn.execute('SELECT COUNT(DISTINCT work_id) FROM text_lines WHERE edition_id = ?', (edition_id,)).fetchone()[0]}):")
    for row in conn.execute("""
        SELECT w.title, COUNT(*) as lines
        FROM text_lines tl JOIN works w ON tl.work_id = w.id
        WHERE tl.edition_id = ?
        GROUP BY w.id
        ORDER BY w.title
    """, (edition_id,)).fetchall():
        print(f"  {row[0]:40s} {row[1]:>6,} lines")

    conn.close()


if __name__ == '__main__':
    main()
