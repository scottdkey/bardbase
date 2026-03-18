-- Word-level linguistic annotations (Folger TEIsimple MorphAdorner POS tagging)
--
-- One row per word token per text line. Currently populated only for the
-- Folger Shakespeare edition (source_key='folger'), which encodes every word
-- as <w lemma="…" ana="#…"> in its TEIsimple XML.
--
-- pos values use the MorphAdorner tagset (e.g. 'j'=adjective, 'vvz'=verb
-- 3rd-person singular present, 'n1'=common noun singular, 'av'=adverb).
-- The leading '#' from the XML ana attribute is stripped on import.
--
-- Expected volume: ~740k rows for 37 Folger plays (~8 words × ~2,500 lines/play).
CREATE TABLE IF NOT EXISTS word_annotations (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    line_id  INTEGER NOT NULL REFERENCES text_lines(id),
    position INTEGER NOT NULL,  -- 1-indexed word position within the line
    word     TEXT    NOT NULL,  -- surface form (as it appears in the text)
    lemma    TEXT,              -- dictionary headword form (from lemma="" attr)
    pos      TEXT               -- MorphAdorner POS tag (from ana="" attr, # stripped)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_word_ann_pk    ON word_annotations(line_id, position);
CREATE INDEX        IF NOT EXISTS idx_word_ann_lemma ON word_annotations(lemma);
