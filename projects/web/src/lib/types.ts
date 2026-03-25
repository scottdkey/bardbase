// Shared TypeScript types for the Bardbase API.
// These mirror the JSON shapes returned by the Go HTTP API.

export interface LexiconListItem {
	id: number;
	key: string;
	orthography: string | null;
}

export interface SearchResult {
	id: number;
	key: string;
	orthography: string | null;
}

export interface EditionLineRef {
	edition_id: number;
	edition_code: string;
	line_number: number | null;
}

export interface LexiconCitationDetail {
	id: number;
	entry_id: number;
	sense_id: number | null;
	work_id: number | null;
	work_abbrev: string | null;
	work_title: string | null;
	act: number | null;
	scene: number | null;
	line: number | null;
	quote_text: string | null;
	display_text: string | null;
	raw_bibl: string | null;
	matched_line: string | null;
	matched_line_number: number | null;
	matched_character: string | null;
	matched_edition_id: number | null;
	edition_lines: EditionLineRef[];
}

export interface LexiconSenseDetail {
	id: number;
	entry_id: number;
	sense_number: number;
	sub_sense: string | null;
	definition_text: string | null;
}

export interface LexiconSubEntryDetail {
	id: number;
	key: string;
	entry_type: string | null;
	full_text: string | null;
	orthography: string | null;
	senses: LexiconSenseDetail[];
	citations: LexiconCitationDetail[];
}

export interface ReferenceCitation {
	source_name: string;
	source_code: string;
	entry_id: number;
	entry_headword: string;
	work_title: string | null;
	work_abbrev: string | null;
	work_slug: string | null;
	act: number | null;
	scene: number | null;
	line: number | null;
	edition_lines: EditionLineRef[];
}

export interface LexiconEntryDetail {
	id: number;
	key: string;
	orthography: string | null;
	entry_type: string | null;
	full_text: string | null;
	subEntries: LexiconSubEntryDetail[];
	senses: LexiconSenseDetail[];
	citations: LexiconCitationDetail[];
	references: ReferenceCitation[];
}

export interface EditionInfo {
	id: number;
	code: string;
	name: string;
}

export interface AlignedEditionLine {
	line_number: number | null;
	content: string;
	content_type: string | null;
	character_name: string | null;
}

export interface AlignedSceneRow {
	editions: Record<number, AlignedEditionLine | null>;
}

export interface CharacterInfo {
	name: string;
	description?: string;
	speech_count: number;
}

export interface MultiEditionScene {
	work_title: string;
	act: number;
	scene: number;
	available_editions: EditionInfo[];
	characters: CharacterInfo[];
	rows: AlignedSceneRow[];
}

export interface FooterAttribution {
	source_name: string;
	attribution_html: string;
	license_notice_text: string | null;
	display_priority: number;
	required: boolean;
}

export interface DbStats {
	work_count: number;
	character_count: number;
	line_count: number;
	lexicon_count: number;
}
