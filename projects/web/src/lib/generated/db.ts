// AUTO-GENERATED — do not edit manually.
// Regenerate with: npm run generate:types
// Source: sqlite schema introspected from build/bardbase.db

import { z } from 'zod';

export interface Attributions {
	id: number;
	source_id: number;
	required: number | null;
	attribution_text: string;
	attribution_html: string | null;
	display_format: string | null;
	display_context: string | null;
	display_priority: number | null;
	requires_link_back: number | null;
	link_back_url: string | null;
	requires_license_notice: number | null;
	license_notice_text: string | null;
	requires_author_credit: number | null;
	author_credit_text: string | null;
	share_alike_required: number | null;
	commercial_use_allowed: number | null;
	modification_allowed: number | null;
	notes: string | null;
	created_at: string | null;
}

export const AttributionsSchema = z.object({
	id: z.number(),
	source_id: z.number(),
	required: z.number().nullable(),
	attribution_text: z.string(),
	attribution_html: z.string().nullable(),
	display_format: z.string().nullable(),
	display_context: z.string().nullable(),
	display_priority: z.number().nullable(),
	requires_link_back: z.number().nullable(),
	link_back_url: z.string().nullable(),
	requires_license_notice: z.number().nullable(),
	license_notice_text: z.string().nullable(),
	requires_author_credit: z.number().nullable(),
	author_credit_text: z.string().nullable(),
	share_alike_required: z.number().nullable(),
	commercial_use_allowed: z.number().nullable(),
	modification_allowed: z.number().nullable(),
	notes: z.string().nullable(),
	created_at: z.string().nullable(),
});

export type AttributionsRow = z.infer<typeof AttributionsSchema>;

export interface Characters {
	id: number;
	char_id: string | null;
	name: string;
	abbrev: string | null;
	work_id: number | null;
	oss_work_id: string | null;
	description: string | null;
	speech_count: number | null;
}

export const CharactersSchema = z.object({
	id: z.number(),
	char_id: z.string().nullable(),
	name: z.string(),
	abbrev: z.string().nullable(),
	work_id: z.number().nullable(),
	oss_work_id: z.string().nullable(),
	description: z.string().nullable(),
	speech_count: z.number().nullable(),
});

export type CharactersRow = z.infer<typeof CharactersSchema>;

export interface CitationMatches {
	id: number;
	citation_id: number;
	text_line_id: number;
	edition_id: number;
	match_type: string | null;
	confidence: number | null;
	matched_text: string | null;
	notes: string | null;
}

export const CitationMatchesSchema = z.object({
	id: z.number(),
	citation_id: z.number(),
	text_line_id: z.number(),
	edition_id: z.number(),
	match_type: z.string().nullable(),
	confidence: z.number().nullable(),
	matched_text: z.string().nullable(),
	notes: z.string().nullable(),
});

export type CitationMatchesRow = z.infer<typeof CitationMatchesSchema>;

export interface Editions {
	id: number;
	name: string;
	short_code: string;
	source_id: number | null;
	year: number | null;
	editors: string | null;
	description: string | null;
	notes: string | null;
}

export const EditionsSchema = z.object({
	id: z.number(),
	name: z.string(),
	short_code: z.string(),
	source_id: z.number().nullable(),
	year: z.number().nullable(),
	editors: z.string().nullable(),
	description: z.string().nullable(),
	notes: z.string().nullable(),
});

export type EditionsRow = z.infer<typeof EditionsSchema>;

export interface ImportLog {
	id: number;
	phase: string;
	action: string;
	details: string | null;
	count: number | null;
	duration_secs: number | null;
	timestamp: string | null;
}

export const ImportLogSchema = z.object({
	id: z.number(),
	phase: z.string(),
	action: z.string(),
	details: z.string().nullable(),
	count: z.number().nullable(),
	duration_secs: z.number().nullable(),
	timestamp: z.string().nullable(),
});

export type ImportLogRow = z.infer<typeof ImportLogSchema>;

export interface LexiconCitations {
	id: number;
	entry_id: number;
	sense_id: number | null;
	work_id: number | null;
	work_abbrev: string | null;
	perseus_ref: string | null;
	act: number | null;
	scene: number | null;
	line: number | null;
	quote_text: string | null;
	display_text: string | null;
	raw_bibl: string | null;
}

export const LexiconCitationsSchema = z.object({
	id: z.number(),
	entry_id: z.number(),
	sense_id: z.number().nullable(),
	work_id: z.number().nullable(),
	work_abbrev: z.string().nullable(),
	perseus_ref: z.string().nullable(),
	act: z.number().nullable(),
	scene: z.number().nullable(),
	line: z.number().nullable(),
	quote_text: z.string().nullable(),
	display_text: z.string().nullable(),
	raw_bibl: z.string().nullable(),
});

export type LexiconCitationsRow = z.infer<typeof LexiconCitationsSchema>;

export interface LexiconEntries {
	id: number;
	key: string;
	letter: string;
	orthography: string | null;
	entry_type: string | null;
	full_text: string | null;
	raw_xml: string | null;
	source_file: string | null;
	created_at: string | null;
}

export const LexiconEntriesSchema = z.object({
	id: z.number(),
	key: z.string(),
	letter: z.string(),
	orthography: z.string().nullable(),
	entry_type: z.string().nullable(),
	full_text: z.string().nullable(),
	raw_xml: z.string().nullable(),
	source_file: z.string().nullable(),
	created_at: z.string().nullable(),
});

export type LexiconEntriesRow = z.infer<typeof LexiconEntriesSchema>;

export interface LexiconSenses {
	id: number;
	entry_id: number;
	sense_number: number;
	definition_text: string | null;
}

export const LexiconSensesSchema = z.object({
	id: z.number(),
	entry_id: z.number(),
	sense_number: z.number(),
	definition_text: z.string().nullable(),
});

export type LexiconSensesRow = z.infer<typeof LexiconSensesSchema>;

export interface LineMappings {
	id: number;
	work_id: number;
	act: number;
	scene: number;
	align_order: number;
	edition_a_id: number;
	edition_b_id: number;
	line_a_id: number | null;
	line_b_id: number | null;
	match_type: string | null;
	similarity: number | null;
}

export const LineMappingsSchema = z.object({
	id: z.number(),
	work_id: z.number(),
	act: z.number(),
	scene: z.number(),
	align_order: z.number(),
	edition_a_id: z.number(),
	edition_b_id: z.number(),
	line_a_id: z.number().nullable(),
	line_b_id: z.number().nullable(),
	match_type: z.string().nullable(),
	similarity: z.number().nullable(),
});

export type LineMappingsRow = z.infer<typeof LineMappingsSchema>;

export interface Sources {
	id: number;
	name: string;
	short_code: string;
	url: string | null;
	license: string | null;
	license_url: string | null;
	attribution_text: string | null;
	attribution_required: number | null;
	notes: string | null;
	imported_at: string | null;
}

export const SourcesSchema = z.object({
	id: z.number(),
	name: z.string(),
	short_code: z.string(),
	url: z.string().nullable(),
	license: z.string().nullable(),
	license_url: z.string().nullable(),
	attribution_text: z.string().nullable(),
	attribution_required: z.number().nullable(),
	notes: z.string().nullable(),
	imported_at: z.string().nullable(),
});

export type SourcesRow = z.infer<typeof SourcesSchema>;

export interface TextDivisions {
	id: number;
	work_id: number;
	edition_id: number;
	act: number;
	scene: number;
	description: string | null;
	line_count: number | null;
}

export const TextDivisionsSchema = z.object({
	id: z.number(),
	work_id: z.number(),
	edition_id: z.number(),
	act: z.number(),
	scene: z.number(),
	description: z.string().nullable(),
	line_count: z.number().nullable(),
});

export type TextDivisionsRow = z.infer<typeof TextDivisionsSchema>;

export interface TextLines {
	id: number;
	work_id: number;
	edition_id: number;
	act: number | null;
	scene: number | null;
	paragraph_num: number | null;
	line_number: number | null;
	character_id: number | null;
	char_name: string | null;
	content: string;
	content_type: string | null;
	word_count: number | null;
	oss_paragraph_id: number | null;
	sonnet_number: number | null;
	stanza: number | null;
	line_type: string | null;
}

export const TextLinesSchema = z.object({
	id: z.number(),
	work_id: z.number(),
	edition_id: z.number(),
	act: z.number().nullable(),
	scene: z.number().nullable(),
	paragraph_num: z.number().nullable(),
	line_number: z.number().nullable(),
	character_id: z.number().nullable(),
	char_name: z.string().nullable(),
	content: z.string(),
	content_type: z.string().nullable(),
	word_count: z.number().nullable(),
	oss_paragraph_id: z.number().nullable(),
	sonnet_number: z.number().nullable(),
	stanza: z.number().nullable(),
	line_type: z.string().nullable(),
});

export type TextLinesRow = z.infer<typeof TextLinesSchema>;

export interface Works {
	id: number;
	oss_id: string | null;
	title: string;
	full_title: string | null;
	short_title: string | null;
	schmidt_abbrev: string | null;
	work_type: string | null;
	date_composed: number | null;
	genre_type: string | null;
	total_words: number | null;
	total_paragraphs: number | null;
	source_text: string | null;
	folger_url: string | null;
	perseus_id: string | null;
	notes: string | null;
}

export const WorksSchema = z.object({
	id: z.number(),
	oss_id: z.string().nullable(),
	title: z.string(),
	full_title: z.string().nullable(),
	short_title: z.string().nullable(),
	schmidt_abbrev: z.string().nullable(),
	work_type: z.string().nullable(),
	date_composed: z.number().nullable(),
	genre_type: z.string().nullable(),
	total_words: z.number().nullable(),
	total_paragraphs: z.number().nullable(),
	source_text: z.string().nullable(),
	folger_url: z.string().nullable(),
	perseus_id: z.string().nullable(),
	notes: z.string().nullable(),
});

export type WorksRow = z.infer<typeof WorksSchema>;
