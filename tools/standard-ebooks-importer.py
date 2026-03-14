#!/usr/bin/env python3
"""
Standard Ebooks Shakespeare Importer
Downloads all Shakespeare plays from Standard Ebooks (CC0) and imports into shakespeare.db

Standard Ebooks format:
- XHTML files per act (src/epub/text/act-N.xhtml)
- Semantic markup: z3998:persona, z3998:verse, z3998:stage-direction
- Table-based layout: character in col 1, speech in col 2
- Verse lines as <span> inside <p>, prose as plain <td>
"""

import os
import sys
import json
import time
import sqlite3
import urllib.request
from html.parser import HTMLParser

WORKSPACE = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DB_PATH = os.path.join(WORKSPACE, 'shakespeare_db', 'shakespeare.db')
CACHE_DIR = os.path.join(WORKSPACE, 'shakespeare_db', 'standard-ebooks-cache')

# Repo name → oss_id mapping
SE_REPOS = {
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


class SEPlayParser(HTMLParser):
    """Parse Standard Ebooks play XHTML into structured lines."""

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
        # Track nested stage dirs inside speech cells
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
                # Stage direction inside a speech cell
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
            line_text = self.buf.strip()
            if line_text:
                self.verse_lines.append(line_text)
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
                self.lines.append({
                    'act': self.current_act,
                    'scene': self.current_scene,
                    'character': None,
                    'text': sd,
                    'is_stage_direction': True,
                    'line_in_scene': self.scene_line_counter,
                })
            self.in_stage_dir = False
            self.buf = ''

        elif tag == 'td' and self.td_is_speech:
            # Flush speech lines
            if self.verse_lines:
                for item in self.verse_lines:
                    if isinstance(item, tuple) and item[0] == 'SD':
                        self.scene_line_counter += 1
                        self.lines.append({
                            'act': self.current_act,
                            'scene': self.current_scene,
                            'character': None,
                            'text': item[1],
                            'is_stage_direction': True,
                            'line_in_scene': self.scene_line_counter,
                        })
                    else:
                        self.scene_line_counter += 1
                        self.lines.append({
                            'act': self.current_act,
                            'scene': self.current_scene,
                            'character': self.current_character,
                            'text': item,
                            'is_stage_direction': False,
                            'line_in_scene': self.scene_line_counter,
                        })
            else:
                # Prose — single block
                prose = self.buf.strip()
                if prose:
                    self.scene_line_counter += 1
                    self.lines.append({
                        'act': self.current_act,
                        'scene': self.current_scene,
                        'character': self.current_character,
                        'text': prose,
                        'is_stage_direction': False,
                        'line_in_scene': self.scene_line_counter,
                    })
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


def fetch_url(url, retries=3):
    for attempt in range(retries):
        try:
            req = urllib.request.Request(url, headers={
                'User-Agent': 'Shakespeare-DB-Builder/1.0 (academic research)',
            })
            with urllib.request.urlopen(req, timeout=30) as resp:
                return resp.read().decode('utf-8')
        except Exception as e:
            if attempt < retries - 1:
                time.sleep(2)
            else:
                print(f"  ERROR: {e}")
                return None


def download_play(repo_name):
    """Download all act files, with local caching."""
    os.makedirs(CACHE_DIR, exist_ok=True)
    cache_file = os.path.join(CACHE_DIR, f'{repo_name}.json')

    if os.path.exists(cache_file):
        with open(cache_file) as f:
            return json.load(f)

    api_url = f"https://api.github.com/repos/standardebooks/{repo_name}/contents/src/epub/text"
    listing = fetch_url(api_url)
    if not listing:
        return None

    files = json.loads(listing)
    act_files = sorted([f['name'] for f in files
                       if f['name'].startswith('act-') and f['name'].endswith('.xhtml')])

    if not act_files:
        print("  No act files found!")
        return None

    acts_data = {}
    for fname in act_files:
        url = f"https://raw.githubusercontent.com/standardebooks/{repo_name}/master/src/epub/text/{fname}"
        print(f"    ↓ {fname}")
        content = fetch_url(url)
        if content:
            acts_data[fname] = content
        time.sleep(0.5)

    with open(cache_file, 'w') as f:
        json.dump(acts_data, f)

    return acts_data


def parse_play(acts_data):
    """Parse XHTML files into list of line dicts."""
    all_lines = []
    sort_order = 0

    for fname in sorted(acts_data.keys()):
        content = acts_data[fname]
        parser = SEPlayParser()
        try:
            parser.feed(content)
        except Exception as e:
            print(f"  Parse error in {fname}: {e}")
            continue

        for line in parser.lines:
            sort_order += 1
            line['sort_order'] = sort_order
            all_lines.append(line)

    return all_lines


def ensure_source_and_edition(conn):
    """Create the SE source and edition if they don't exist."""
    # Check if source exists
    row = conn.execute("SELECT id FROM sources WHERE short_code = 'standard_ebooks'").fetchone()
    if not row:
        conn.execute("""
            INSERT INTO sources (name, short_code, url, license, license_url, attribution_text, attribution_required, notes)
            VALUES (
                'Standard Ebooks',
                'standard_ebooks',
                'https://standardebooks.org',
                'CC0 1.0 Universal',
                'https://creativecommons.org/publicdomain/zero/1.0/',
                'Standard Ebooks — Free and liberated ebooks, carefully produced for the true book lover. standardebooks.org',
                0,
                'Public domain dedication. No attribution legally required. Based on public domain source texts.'
            )
        """)
        source_id = conn.execute("SELECT id FROM sources WHERE short_code = 'standard_ebooks'").fetchone()[0]
    else:
        source_id = row[0]

    # Check if edition exists
    row = conn.execute("SELECT id FROM editions WHERE short_code = 'se_modern'").fetchone()
    if not row:
        conn.execute("""
            INSERT INTO editions (name, short_code, source_id, year, editors, description, notes)
            VALUES (
                'Standard Ebooks Modern Edition',
                'se_modern',
                ?,
                2024,
                'Standard Ebooks editorial team',
                'Carefully produced modern-spelling editions. CC0 licensed.',
                'Based on public domain source texts with editorial corrections and semantic markup.'
            )
        """, (source_id,))
        edition_id = conn.execute("SELECT id FROM editions WHERE short_code = 'se_modern'").fetchone()[0]
    else:
        edition_id = row[0]

    conn.commit()
    return source_id, edition_id


def import_lines(conn, work_db_id, edition_id, lines):
    """Import parsed lines into text_lines."""
    # Clear existing SE lines for this work
    conn.execute("DELETE FROM text_lines WHERE work_id = ? AND edition_id = ?", (work_db_id, edition_id))

    for line in lines:
        char_name = line.get('character')

        # Try to match character to characters table
        char_id = None
        if char_name:
            # Try exact match first
            row = conn.execute(
                "SELECT id FROM characters WHERE work_id = ? AND UPPER(name) = UPPER(?)",
                (work_db_id, char_name)
            ).fetchone()
            if not row:
                # Try abbreviation
                row = conn.execute(
                    "SELECT id FROM characters WHERE work_id = ? AND UPPER(abbrev) = UPPER(?)",
                    (work_db_id, char_name)
                ).fetchone()
            if row:
                char_id = row[0]

        content_type = 'stage_direction' if line.get('is_stage_direction') else 'speech'
        text = line['text']

        conn.execute("""
            INSERT INTO text_lines (work_id, edition_id, act, scene, paragraph_num, character_id, char_name, content, content_type, word_count)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """, (
            work_db_id,
            edition_id,
            line.get('act'),
            line.get('scene'),
            line.get('line_in_scene'),
            char_id,
            char_name,
            text,
            content_type,
            len(text.split())
        ))

    # Also create text_divisions entries
    conn.execute("DELETE FROM text_divisions WHERE work_id = ? AND edition_id = ?", (work_db_id, edition_id))
    scenes = {}
    for line in lines:
        key = (line.get('act'), line.get('scene'))
        if key[0] is not None and key not in scenes:
            scenes[key] = 0
        if key in scenes:
            scenes[key] += 1

    for (act, scene), count in sorted(scenes.items()):
        conn.execute("""
            INSERT OR IGNORE INTO text_divisions (work_id, edition_id, act, scene, line_count)
            VALUES (?, ?, ?, ?, ?)
        """, (work_db_id, edition_id, act, scene, count))

    conn.commit()


def main():
    if not os.path.exists(DB_PATH):
        print(f"ERROR: Database not found at {DB_PATH}")
        sys.exit(1)

    conn = sqlite3.connect(DB_PATH)
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA foreign_keys=ON")

    source_id, edition_id = ensure_source_and_edition(conn)

    # Build oss_id → db id map
    works_map = {}
    for row in conn.execute("SELECT id, oss_id, title FROM works").fetchall():
        works_map[row[1]] = (row[0], row[2])

    print("=" * 60)
    print("Standard Ebooks Shakespeare Importer")
    print("=" * 60)
    print(f"Database: {DB_PATH}")
    print(f"Edition ID: {edition_id} (se_modern)")
    print(f"Works in DB: {len(works_map)}")
    print()

    force = '--force' in sys.argv
    total_lines = 0
    total_plays = 0
    errors = []

    for repo_name in sorted(SE_REPOS.keys()):
        oss_id = SE_REPOS[repo_name]
        if oss_id not in works_map:
            print(f"[SKIP] {repo_name} → {oss_id} (not in works table)")
            continue

        work_db_id, title = works_map[oss_id]
        total_plays += 1
        print(f"[{total_plays:2d}/37] {title}")

        # Check existing
        existing = conn.execute(
            "SELECT COUNT(*) FROM text_lines WHERE work_id = ? AND edition_id = ?",
            (work_db_id, edition_id)
        ).fetchone()[0]

        if existing > 0 and not force:
            print(f"  Already imported ({existing} lines)")
            total_lines += existing
            continue

        # Download
        acts_data = download_play(repo_name)
        if not acts_data:
            errors.append(f"Download failed: {repo_name}")
            continue

        # Parse
        lines = parse_play(acts_data)
        if not lines:
            errors.append(f"Parse failed: {repo_name}")
            continue

        # Import
        import_lines(conn, work_db_id, edition_id, lines)

        acts = len(set(l['act'] for l in lines if l.get('act')))
        scenes = len(set((l['act'], l['scene']) for l in lines if l.get('act') and l.get('scene')))
        sd = sum(1 for l in lines if l.get('is_stage_direction'))
        speech = len(lines) - sd
        total_lines += len(lines)

        print(f"  ✓ {len(lines)} lines ({acts} acts, {scenes} scenes, {speech} speech, {sd} stage dirs)")
        time.sleep(0.3)

    # Summary
    print()
    print("=" * 60)
    se_total = conn.execute("SELECT COUNT(*) FROM text_lines WHERE edition_id = ?", (edition_id,)).fetchone()[0]
    oss_total = conn.execute("SELECT COUNT(*) FROM text_lines WHERE edition_id = 1").fetchone()[0]
    all_total = conn.execute("SELECT COUNT(*) FROM text_lines").fetchone()[0]
    db_size = os.path.getsize(DB_PATH) / 1024 / 1024

    print(f"DONE: {total_plays} plays processed")
    print(f"\nText lines in DB:")
    print(f"  OSS/Moby:         {oss_total:>8,}")
    print(f"  Standard Ebooks:  {se_total:>8,}")
    print(f"  Total:            {all_total:>8,}")
    print(f"\nDatabase size: {db_size:.1f} MB")

    if errors:
        print(f"\nErrors ({len(errors)}):")
        for e in errors:
            print(f"  ✗ {e}")

    conn.close()


if __name__ == '__main__':
    main()
