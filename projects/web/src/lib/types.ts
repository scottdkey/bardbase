/** Core types derived from the Shakespeare DB schema */

export interface Work {
	id: number;
	title: string;
	long_title: string;
	oss_id: number;
	schmidt_abbrev: string;
	work_type: string;
	year: string;
	genre_desc: string;
	total_scenes: number;
	total_paragraphs: number;
}

export interface Character {
	id: number;
	char_id: string;
	name: string;
	abbrev: string;
	work_id: number;
	description: string;
	speech_count: number;
}

export interface TextLine {
	id: number;
	work_id: number;
	edition_id: number;
	act: number;
	scene: number;
	line_number: number;
	character_id: number | null;
	content: string;
	is_stage_direction: boolean;
}

export interface Edition {
	id: number;
	name: string;
	short_code: string;
	source_id: number;
	year: string;
}

export interface LexiconEntry {
	id: number;
	key: string;
	letter: string;
	orthography: string;
	full_text: string;
}
