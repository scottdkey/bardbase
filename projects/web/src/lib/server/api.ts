import type { D1Database } from '@cloudflare/workers-types';
import type {
    FooterAttribution,
    MultiEditionScene,
    SearchResult
} from '$lib/types';

export interface LexiconLetter {
    letter: string;
    count: number;
}

export interface CorrectionIssue {
    number: number;
    title: string;
    state: string;
    url: string;
    created_at: string;
    updated_at: string;
    labels: string[];
    body: string;
}

export interface Work {
    id: number;
    title: string;
    slug: string;
    work_type: string;
    date_composed: string | null;
}

export interface WorkEdition {
    id: number;
    name: string;
    short_code: string;
    year: number | null;
    source_name: string;
}

export interface LineReference {
    entry_id: number;
    entry_key: string;
    source: string;
    source_code: string;
    sense_id: number | null;
    definition: string | null;
    quote_text: string | null;
    line: number;
}

export interface ReferenceSource {
    code: string;
    name: string;
    count: number;
}

export interface CitationSpan {
    start: number;
    end: number;
    work_slug?: string;
    act?: number;
    scene?: number;
    line?: number;
}

export interface WorkDivision {
    act: number;
    scene: number;
    description: string | null;
    line_count: number;
}

export interface LexiconIndexEntry {
    id: number;
    key: string;
}

function slugify(title: string): string {
    let result = '';
    let lastDash = true;
    for (const ch of title.toLowerCase()) {
        const code = ch.codePointAt(0)!;
        const isAlnum =
            (code >= 97 && code <= 122) ||
            (code >= 48 && code <= 57);
        if (isAlnum) {
            result += ch;
            lastDash = false;
        } else if (!lastDash) {
            result += '-';
            lastDash = true;
        }
    }
    return result.replace(/-$/, '');
}

function placeholders(n: number): string {
    return Array(n).fill('?').join(',');
}

// ── Module-level caches ──────────────────────────────────────────────────────
// Worker isolates persist for minutes to hours. Every value cached below is
// derived from immutable DB content (the DB is rebuilt and redeployed together;
// nothing mutates at runtime), so cache-forever-per-isolate is correct. A new
// deploy recycles the isolate, which naturally invalidates.

let worksCache: { plays: Work[]; poetry: Work[] } | null = null;
let attributionsCache: FooterAttribution[] | null = null;
let referenceSourcesCache: ReferenceSource[] | null = null;
let lexiconKeysCache: string[] | null = null;
let lexiconLettersCache: LexiconLetter[] | null = null;
const workTOCCache = new Map<string | number, WorkDivision[]>();
const workBySlugCache = new Map<string, { id: number; title: string; slug: string }>();
const workEditionsCache = new Map<string | number, WorkEdition[]>();
const charactersCache = new Map<number, { name: string; description?: string; speech_count: number }[]>();

// slug → id lookup, built lazily from getWorks.
let slugToIdCache: Map<string, number> | null = null;

async function resolveWorkId(db: D1Database, idOrSlug: number | string): Promise<number | null> {
    if (typeof idOrSlug === 'number') return idOrSlug;
    const n = Number(idOrSlug);
    if (!isNaN(n) && String(n) === idOrSlug) return n;

    if (!slugToIdCache) {
        const works = await getWorks(db);
        slugToIdCache = new Map();
        for (const w of [...works.plays, ...works.poetry]) slugToIdCache.set(w.slug, w.id);
    }
    return slugToIdCache.get(idOrSlug) ?? null;
}

export async function getWorks(db: D1Database): Promise<{ plays: Work[]; poetry: Work[] }> {
    if (worksCache) return worksCache;

    const { results } = await db
        .prepare('SELECT id, title, work_type, date_composed FROM works ORDER BY title')
        .all<{ id: number; title: string; work_type: string; date_composed: string | null }>();

    const plays: Work[] = [];
    const poetry: Work[] = [];
    for (const row of results ?? []) {
        const work: Work = { ...row, slug: slugify(row.title) };
        if (row.work_type === 'comedy' || row.work_type === 'tragedy' || row.work_type === 'history') {
            plays.push(work);
        } else if (row.work_type === 'poem' || row.work_type === 'sonnet_sequence') {
            poetry.push(work);
        }
    }
    worksCache = { plays, poetry };
    return worksCache;
}

export async function getWorkTOC(db: D1Database, idOrSlug: number | string): Promise<WorkDivision[]> {
    const cached = workTOCCache.get(idOrSlug);
    if (cached) return cached;

    const workId = await resolveWorkId(db, idOrSlug);
    if (workId === null) return [];
    const { results } = await db
        .prepare(
            `SELECT td.act, td.scene, td.description,
                COUNT(tl.id) AS line_count
            FROM text_divisions td
            LEFT JOIN text_lines tl
                ON tl.work_id = td.work_id
                AND tl.act = td.act
                AND tl.scene = td.scene
                AND tl.edition_id = td.edition_id
            WHERE td.work_id = ? AND td.edition_id = (
              SELECT MIN(edition_id) FROM text_divisions WHERE work_id = ?
            )
            GROUP BY td.act, td.scene
            ORDER BY td.act, td.scene`
        )
        .bind(workId, workId)
        .all<WorkDivision>();
    const toc = results ?? [];
    workTOCCache.set(idOrSlug, toc);
    return toc;
}

export async function getWorkBySlug(db: D1Database, slug: string): Promise<{ id: number; title: string; slug: string }> {
    const cached = workBySlugCache.get(slug);
    if (cached) return cached;

    // Resolve through the cached works list — no extra query.
    const works = await getWorks(db);
    for (const w of [...works.plays, ...works.poetry]) {
        if (w.slug === slug) {
            const out = { id: w.id, title: w.title, slug };
            workBySlugCache.set(slug, out);
            return out;
        }
    }
    throw new Error(`Work not found: ${slug}`);
}

export async function getWorkEditions(db: D1Database, idOrSlug: number | string): Promise<WorkEdition[]> {
    const cached = workEditionsCache.get(idOrSlug);
    if (cached) return cached;

    const workId = await resolveWorkId(db, idOrSlug);
    if (workId === null) return [];
    const { results } = await db
        .prepare(
            `SELECT e.id, e.name, e.short_code, e.year, s.name as source_name
            FROM editions e
            JOIN sources s ON s.id = e.source_id
            WHERE e.id IN (
              SELECT DISTINCT edition_id FROM text_lines WHERE work_id = ?
            )
            ORDER BY e.id`
        )
        .bind(workId)
        .all<WorkEdition>();
    const editions = results ?? [];
    workEditionsCache.set(idOrSlug, editions);
    return editions;
}

export async function getAttributions(db: D1Database): Promise<FooterAttribution[]> {
    if (attributionsCache) return attributionsCache;

    const { results } = await db
        .prepare(
            `SELECT s.name AS source_name, a.attribution_html, a.license_notice_text,
            COALESCE(a.display_priority, 0) AS display_priority,
            CASE WHEN a.required = 1 THEN 1 ELSE 0 END AS required
            FROM attributions a
            JOIN sources s ON s.id = a.source_id
            WHERE a.display_format = 'footer'
              AND (
                s.id IN (
                  SELECT DISTINCT e.source_id FROM editions e
                  WHERE EXISTS (SELECT 1 FROM text_lines tl WHERE tl.edition_id = e.id)
                )
                OR (s.short_code = 'perseus_schmidt'
                    AND EXISTS (SELECT 1 FROM lexicon_entries LIMIT 1))
              )
            ORDER BY a.display_priority DESC, s.name`
        )
        .all<{
            source_name: string;
            attribution_html: string;
            license_notice_text: string | null;
            display_priority: number;
            required: number;
        }>();
    attributionsCache = (results ?? []).map((r) => ({ ...r, required: r.required === 1 }));
    return attributionsCache;
}

export async function getLexiconKeys(db: D1Database): Promise<string[]> {
    if (lexiconKeysCache) return lexiconKeysCache;
    const { results } = await db
        .prepare('SELECT DISTINCT LOWER(base_key) AS key FROM lexicon_entries ORDER BY 1')
        .all<{ key: string }>();
    lexiconKeysCache = (results ?? []).map((r) => r.key);
    return lexiconKeysCache;
}

export async function getLexiconLetters(db: D1Database): Promise<LexiconLetter[]> {
    if (lexiconLettersCache) return lexiconLettersCache;
    const { results } = await db
        .prepare('SELECT letter, COUNT(*) AS count FROM lexicon_entries GROUP BY letter ORDER BY letter')
        .all<LexiconLetter>();
    lexiconLettersCache = results ?? [];
    return lexiconLettersCache;
}

export async function getReferenceSources(db: D1Database): Promise<ReferenceSource[]> {
    if (referenceSourcesCache) return referenceSourcesCache;

    const result: ReferenceSource[] = [];

    const schmidtRow = await db
        .prepare('SELECT COUNT(DISTINCT base_key) AS cnt FROM lexicon_entries')
        .first<{ cnt: number }>();
    if (schmidtRow) {
        result.push({ code: 'schmidt', name: 'Schmidt Shakespeare Lexicon', count: schmidtRow.cnt });
    }

    const { results } = await db
        .prepare(
            `SELECT s.short_code, s.name, COUNT(re.id) as entry_count
            FROM sources s
            JOIN reference_entries re ON re.source_id = s.id
            GROUP BY s.id
            ORDER BY s.name`
        )
        .all<{ short_code: string; name: string; entry_count: number }>();
    for (const row of results ?? []) {
        result.push({ code: row.short_code, name: row.name, count: row.entry_count });
    }
    referenceSourcesCache = result;
    return result;
}

export async function getCorrections(state = 'all'): Promise<CorrectionIssue[]> {
    const url = `https://api.github.com/repos/scottdkey/bardbase/issues?labels=correction&state=${state}&per_page=100&sort=created&direction=desc`;
    const res = await fetch(url, {
        headers: {
            Accept: 'application/vnd.github.v3+json',
            'User-Agent': 'bardbase-web'
        }
    });
    if (!res.ok) throw new Error(`GitHub returned ${res.status}`);
    const issues = (await res.json()) as {
        number: number;
        title: string;
        state: string;
        html_url: string;
        created_at: string;
        updated_at: string;
        labels: { name: string }[];
        body: string;
    }[];
    return issues.map((i) => ({
        number: i.number,
        title: i.title,
        state: i.state,
        url: i.html_url,
        created_at: i.created_at,
        updated_at: i.updated_at,
        labels: i.labels.map((l) => l.name),
        body: i.body
    }));
}

type LineRow = {
    id: number;
    editionId: number;
    lineNumber: number | null;
    content: string;
    contentType: string | null;
    characterName: string | null;
};

type GapEntry = { edId: number; lineId: number };

type EditionInfo = { id: number; code: string; name: string };

type AlignedLine = {
    line_number: number | null;
    content: string;
    content_type: string | null;
    character_name: string | null;
};

// Character-name coalesce expression: prefer the FK-joined characters.name, fall
// back to a LIKE match against characters for the same work (catches abbreviations
// like "Ham." when the character name is "Hamlet"), final fallback is the raw tag.
const charCoalesceExpr = `COALESCE(
    c.name,
    (SELECT c2.name FROM characters c2
     WHERE c2.work_id = tl.work_id
       AND LOWER(c2.name) LIKE LOWER(REPLACE(REPLACE(REPLACE(tl.char_name, '.', ''), 'æ', 'ae'), 'Æ', 'Ae')) || '%'
     LIMIT 1),
    tl.char_name
)`;

async function mergeWorkLevelEditions(
    db: D1Database,
    workId: number,
    anchorId: number,
    anchorLines: LineRow[],
    rows: { editions: Record<number, AlignedLine> }[],
    availEditions: EditionInfo[]
): Promise<[{ editions: Record<number, AlignedLine> }[], EditionInfo[]]> {
    const editionsWithContent = new Set<number>();
    for (const row of rows) {
        for (const edId of Object.keys(row.editions)) {
            if (Number(edId) !== anchorId) editionsWithContent.add(Number(edId));
        }
    }

    const wlEditionsRes = await db
        .prepare(
            `SELECT DISTINCT
            CASE WHEN lm.edition_a_id = ? THEN lm.edition_b_id ELSE lm.edition_a_id END AS other_ed_id,
            e.short_code, e.name
            FROM line_mappings lm
            JOIN editions e ON e.id = CASE WHEN lm.edition_a_id = ? THEN lm.edition_b_id ELSE lm.edition_a_id END
            WHERE lm.work_id = ? AND lm.act = 0 AND lm.scene = 0
              AND (lm.edition_a_id = ? OR lm.edition_b_id = ?)
              AND e.id NOT IN (0, 10, 11)`
        )
        .bind(anchorId, anchorId, workId, anchorId, anchorId)
        .all<{ other_ed_id: number; short_code: string; name: string }>();
    const wlEditions = wlEditionsRes.results ?? [];

    const candidateEditions = wlEditions.filter((e) => !editionsWithContent.has(e.other_ed_id));
    if (candidateEditions.length === 0) return [rows, availEditions];

    const anchorIdToRow = new Map<number, number>();
    for (const al of anchorLines) {
        for (let i = 0; i < rows.length; i++) {
            const ed = rows[i].editions[anchorId];
            if (ed && ed.content === al.content) {
                anchorIdToRow.set(al.id, i);
                break;
            }
        }
    }

    for (const wlEd of candidateEditions) {
        const mapRowsRes = await db
            .prepare(
                `SELECT lm.line_a_id, lm.line_b_id
                FROM line_mappings lm
                WHERE lm.work_id = ? AND lm.act = 0 AND lm.scene = 0
                  AND ((lm.edition_a_id = ? AND lm.edition_b_id = ?)
                    OR (lm.edition_a_id = ? AND lm.edition_b_id = ?))`
            )
            .bind(workId, anchorId, wlEd.other_ed_id, wlEd.other_ed_id, anchorId)
            .all<{ line_a_id: number | null; line_b_id: number | null }>();
        const mapRows = mapRowsRes.results ?? [];

        const otherLineIds = new Set<number>();
        const anchorToOther = new Map<number, number>();
        for (const mr of mapRows) {
            let anchorLid: number | null = null;
            let otherLid: number | null = null;
            if (mr.line_a_id !== null && anchorIdToRow.has(mr.line_a_id)) {
                anchorLid = mr.line_a_id;
                otherLid = mr.line_b_id;
            } else {
                anchorLid = mr.line_b_id;
                otherLid = mr.line_a_id;
            }
            if (anchorLid !== null && otherLid !== null) {
                anchorToOther.set(anchorLid, otherLid);
                otherLineIds.add(otherLid);
            }
        }

        if (otherLineIds.size === 0) continue;

        const otherIds = [...otherLineIds];
        const olPh = placeholders(otherIds.length);
        const olRowsRes = await db
            .prepare(
                `SELECT tl.id, tl.line_number, tl.content, tl.content_type,
                ${charCoalesceExpr} AS character_name
                FROM text_lines tl
                LEFT JOIN characters c ON c.id = tl.character_id
                WHERE tl.id IN (${olPh})`
            )
            .bind(...otherIds)
            .all<{
                id: number;
                line_number: number | null;
                content: string;
                content_type: string | null;
                character_name: string | null;
            }>();
        const olRows = olRowsRes.results ?? [];

        const otherLines = new Map<number, typeof olRows[0]>();
        for (const ol of olRows) otherLines.set(ol.id, ol);

        let merged = false;
        for (const al of anchorLines) {
            const otherLid = anchorToOther.get(al.id);
            if (otherLid === undefined) continue;
            const ol = otherLines.get(otherLid);
            if (!ol) continue;
            const rowIdx = anchorIdToRow.get(al.id);
            if (rowIdx === undefined) continue;
            rows[rowIdx].editions[wlEd.other_ed_id] = {
                line_number: ol.line_number,
                content: ol.content,
                content_type: ol.content_type,
                character_name: ol.character_name
            };
            merged = true;
        }

        if (merged && !availEditions.some((e) => e.id === wlEd.other_ed_id)) {
            availEditions = [...availEditions, { id: wlEd.other_ed_id, code: wlEd.short_code, name: wlEd.name }];
        }
    }

    return [rows, availEditions];
}

export async function getScene(
    db: D1Database,
    workIdOrSlug: number | string,
    act: number,
    scene: number
): Promise<MultiEditionScene> {
    const workId = await resolveWorkId(db, workIdOrSlug);
    if (workId === null) throw new Error(`Work not found: ${workIdOrSlug}`);

    const workRow = await db
        .prepare('SELECT title, work_type FROM works WHERE id = ?')
        .bind(workId)
        .first<{ title: string; work_type: string }>();
    if (!workRow) throw new Error(`Work not found: ${workId}`);

    const edRowsRes = await db
        .prepare(
            `SELECT DISTINCT e.id, e.short_code, e.name
            FROM editions e JOIN text_lines tl ON tl.edition_id = e.id
            WHERE tl.work_id = ? AND tl.act = ? AND tl.scene = ?
              AND e.id IN (1,2,3,4,5)
            ORDER BY e.id`
        )
        .bind(workId, act, scene)
        .all<{ id: number; short_code: string; name: string }>();
    const edRows = edRowsRes.results ?? [];

    if (edRows.length === 0) throw new Error(`Scene not found: ${workIdOrSlug} ${act}.${scene}`);

    let availEditions: EditionInfo[] = edRows.map((e) => ({ id: e.id, code: e.short_code, name: e.name }));
    const editionIds = availEditions.map((e) => e.id);

    const edPh = placeholders(editionIds.length);
    const rawLineRowsRes = await db
        .prepare(
            `SELECT tl.id, tl.edition_id, tl.line_number, tl.content, tl.content_type,
            ${charCoalesceExpr} AS character_name
            FROM text_lines tl
            LEFT JOIN characters c ON c.id = tl.character_id
            WHERE tl.work_id = ? AND tl.edition_id IN (${edPh})
              AND tl.act = ? AND tl.scene = ?
            ORDER BY tl.edition_id, tl.line_number, tl.id`
        )
        .bind(workId, ...editionIds, act, scene)
        .all<{
            id: number;
            edition_id: number;
            line_number: number | null;
            content: string;
            content_type: string | null;
            character_name: string | null;
        }>();
    const rawLineRows = rawLineRowsRes.results ?? [];

    const linesByEdition = new Map<number, LineRow[]>();
    const lineById = new Map<number, LineRow>();
    for (const r of rawLineRows) {
        const lr: LineRow = {
            id: r.id,
            editionId: r.edition_id,
            lineNumber: r.line_number,
            content: r.content,
            contentType: r.content_type,
            characterName: r.character_name
        };
        if (!linesByEdition.has(lr.editionId)) linesByEdition.set(lr.editionId, []);
        linesByEdition.get(lr.editionId)!.push(lr);
        lineById.set(lr.id, lr);
    }

    let anchorId = editionIds[0];
    if (editionIds.includes(1)) anchorId = 1;
    const anchorLines = linesByEdition.get(anchorId) ?? [];

    if (availEditions.length <= 1) {
        const rows = anchorLines.map((l) => ({
            editions: {
                [anchorId]: {
                    line_number: l.lineNumber,
                    content: l.content,
                    content_type: l.contentType,
                    character_name: l.characterName
                }
            }
        }));
        const characters = await loadCharacters(db, workId);
        return {
            work_title: workRow.title,
            act,
            scene,
            available_editions: availEditions,
            characters,
            rows
        };
    }

    const otherIds = editionIds.filter((id) => id !== anchorId);
    const otherPh = placeholders(otherIds.length);

    const mappingRowsRes = await db
        .prepare(
            `SELECT lm.edition_a_id, lm.edition_b_id, lm.line_a_id, lm.line_b_id
            FROM line_mappings lm
            WHERE lm.work_id = ? AND lm.act = ? AND lm.scene = ?
              AND (
                (lm.edition_a_id = ? AND lm.edition_b_id IN (${otherPh}))
                OR (lm.edition_b_id = ? AND lm.edition_a_id IN (${otherPh}))
              )
            ORDER BY lm.edition_b_id, lm.align_order`
        )
        .bind(workId, act, scene, anchorId, ...otherIds, anchorId, ...otherIds)
        .all<{
            edition_a_id: number;
            edition_b_id: number;
            line_a_id: number | null;
            line_b_id: number | null;
        }>();
    const mappingRows = mappingRowsRes.results ?? [];

    const anchorToOther = new Map<number, Map<number, number>>();
    const gapsAfterAnchor = new Map<number, GapEntry[]>();
    const lastAnchorByEdition = new Map<number, number>();

    for (const mr of mappingRows) {
        let anchorLineId: number | null = null;
        let otherLineId: number | null = null;
        let otherEdId: number;

        if (mr.edition_a_id === anchorId) {
            anchorLineId = mr.line_a_id;
            otherLineId = mr.line_b_id;
            otherEdId = mr.edition_b_id;
        } else {
            anchorLineId = mr.line_b_id;
            otherLineId = mr.line_a_id;
            otherEdId = mr.edition_a_id;
        }

        if (anchorLineId !== null && otherLineId !== null) {
            if (!anchorToOther.has(anchorLineId)) anchorToOther.set(anchorLineId, new Map());
            anchorToOther.get(anchorLineId)!.set(otherEdId, otherLineId);
            lastAnchorByEdition.set(otherEdId, anchorLineId);
        } else if (anchorLineId === null && otherLineId !== null) {
            const afterId = lastAnchorByEdition.get(otherEdId) ?? 0;
            if (!gapsAfterAnchor.has(afterId)) gapsAfterAnchor.set(afterId, []);
            gapsAfterAnchor.get(afterId)!.push({ edId: otherEdId, lineId: otherLineId });
        } else if (anchorLineId !== null) {
            lastAnchorByEdition.set(otherEdId!, anchorLineId);
        }
    }

    const usedLines = new Set<number>();
    let rows: { editions: Record<number, AlignedLine> }[] = [];

    for (const g of gapsAfterAnchor.get(0) ?? []) {
        const l = lineById.get(g.lineId);
        if (l) {
            rows.push({
                editions: {
                    [g.edId]: {
                        line_number: l.lineNumber,
                        content: l.content,
                        content_type: l.contentType,
                        character_name: l.characterName
                    }
                }
            });
            usedLines.add(g.lineId);
        }
    }

    for (const al of anchorLines) {
        const row: { editions: Record<number, AlignedLine> } = {
            editions: {
                [anchorId]: {
                    line_number: al.lineNumber,
                    content: al.content,
                    content_type: al.contentType,
                    character_name: al.characterName
                }
            }
        };

        const edMap = anchorToOther.get(al.id);
        if (edMap) {
            for (const [otherEdId, otherLineId] of edMap) {
                const ol = lineById.get(otherLineId);
                if (ol) {
                    row.editions[otherEdId] = {
                        line_number: ol.lineNumber,
                        content: ol.content,
                        content_type: ol.contentType,
                        character_name: ol.characterName
                    };
                    usedLines.add(otherLineId);
                }
            }
        }
        rows.push(row);

        for (const g of gapsAfterAnchor.get(al.id) ?? []) {
            if (usedLines.has(g.lineId)) continue;
            const l = lineById.get(g.lineId);
            if (l) {
                rows.push({
                    editions: {
                        [g.edId]: {
                            line_number: l.lineNumber,
                            content: l.content,
                            content_type: l.contentType,
                            character_name: l.characterName
                        }
                    }
                });
                usedLines.add(g.lineId);
            }
        }
    }

    [rows, availEditions] = await mergeWorkLevelEditions(db, workId, anchorId, anchorLines, rows, availEditions);

    // Drop any edition that produced no visible rows (e.g. Perseus Globe on
    // poetry that has text_lines but no line_mappings connecting them).
    const editionsInRows = new Set<number>();
    for (const row of rows) {
        for (const edId of Object.keys(row.editions)) {
            editionsInRows.add(Number(edId));
        }
    }
    availEditions = availEditions.filter((e) => editionsInRows.has(e.id));

    const characters = await loadCharacters(db, workId);
    return {
        work_title: workRow.title,
        act,
        scene,
        available_editions: availEditions,
        characters,
        rows
    };
}

async function loadCharacters(db: D1Database, workId: number) {
    const cached = charactersCache.get(workId);
    if (cached) return cached;

    const { results } = await db
        .prepare(
            `SELECT name, description, COALESCE(speech_count, 0) AS speech_count
            FROM characters WHERE work_id = ? ORDER BY name`
        )
        .bind(workId)
        .all<{ name: string; description: string | null; speech_count: number }>();
    const characters = (results ?? []).map((r) => ({ ...r, description: r.description ?? undefined }));
    charactersCache.set(workId, characters);
    return characters;
}

export async function getSceneReferences(
    db: D1Database,
    workIdOrSlug: number | string,
    act: number,
    scene: number
): Promise<Record<string, LineReference[]>> {
    const workId = await resolveWorkId(db, workIdOrSlug);
    if (workId === null) return {};

    const result: Record<string, LineReference[]> = {};

    const schmidtRowsRes = await db
        .prepare(
            `SELECT tl.line_number, le.id, le.base_key, lc.sense_id, ls.definition_text, lc.quote_text,
               cm.edition_id, cm.confidence
            FROM citation_matches cm
            JOIN lexicon_citations lc ON lc.id = cm.citation_id
            JOIN lexicon_entries le ON le.id = lc.entry_id
            JOIN text_lines tl ON tl.id = cm.text_line_id
            LEFT JOIN lexicon_senses ls ON ls.id = lc.sense_id
            WHERE lc.work_id = ? AND tl.act = ? AND tl.scene = ?
              AND tl.line_number IS NOT NULL
            GROUP BY tl.line_number, le.id, cm.edition_id
            ORDER BY tl.line_number, le.base_key`
        )
        .bind(workId, act, scene)
        .all<{
            line_number: number;
            id: number;
            base_key: string;
            sense_id: number | null;
            definition_text: string | null;
            quote_text: string | null;
            edition_id: number;
            confidence: number;
        }>();

    for (const row of schmidtRowsRes.results ?? []) {
        const key = String(row.line_number);
        if (!result[key]) result[key] = [];
        result[key].push({
            entry_id: row.id,
            entry_key: row.base_key,
            source: 'Schmidt Shakespeare Lexicon',
            source_code: 'schmidt',
            sense_id: row.sense_id,
            definition: row.definition_text,
            quote_text: row.quote_text,
            line: row.line_number
        });
    }

    const refRowsRes = await db
        .prepare(
            `SELECT tl.line_number, re.id, re.headword, s.name, s.short_code, re.raw_text,
               rcm.edition_id, rcm.confidence
            FROM reference_citation_matches rcm
            JOIN reference_citations rc ON rc.id = rcm.ref_citation_id
            JOIN reference_entries re ON re.id = rc.entry_id
            JOIN text_lines tl ON tl.id = rcm.text_line_id
            JOIN sources s ON s.id = rc.source_id
            WHERE rc.work_id = ? AND tl.act = ? AND tl.scene = ?
              AND tl.line_number IS NOT NULL
            GROUP BY tl.line_number, re.id, rcm.edition_id
            ORDER BY tl.line_number, s.short_code, re.headword`
        )
        .bind(workId, act, scene)
        .all<{
            line_number: number;
            id: number;
            headword: string;
            name: string;
            short_code: string;
            raw_text: string;
            edition_id: number;
            confidence: number;
        }>();

    for (const row of refRowsRes.results ?? []) {
        const key = String(row.line_number);
        if (!result[key]) result[key] = [];
        let def = row.raw_text;
        if (def.length > 300) def = def.slice(0, 300) + '…';
        result[key].push({
            entry_id: row.id,
            entry_key: row.headword,
            source: row.name,
            source_code: row.short_code,
            sense_id: null,
            definition: def,
            quote_text: null,
            line: row.line_number
        });
    }

    return result;
}

export async function search(q: string, limit = 20): Promise<SearchResult[]> {
    const res = await fetch(`/api/search?q=${encodeURIComponent(q)}&limit=${limit}`);
    if (!res.ok) throw new Error(`Search failed: ${res.status}`);
    return res.json() as Promise<SearchResult[]>;
}
