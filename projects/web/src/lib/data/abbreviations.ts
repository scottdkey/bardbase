/**
 * Schmidt/Onions lexicographic abbreviation glossary.
 * Used to explain abbreviated terms in lexicon definitions.
 */
export interface Abbreviation {
	abbrev: string;
	expansion: string;
	description?: string;
}

export const abbreviations: Abbreviation[] = [
	// ─── Grammatical terms ───
	{ abbrev: 'absol.', expansion: 'absolute(ly)', description: 'Used without the usual construction (e.g., a verb without its object)' },
	{ abbrev: 'adj.', expansion: 'adjective' },
	{ abbrev: 'adv.', expansion: 'adverb' },
	{ abbrev: 'advb.', expansion: 'adverbial(ly)' },
	{ abbrev: 'attrib.', expansion: 'attributive(ly)' },
	{ abbrev: 'comb.', expansion: 'in combination', description: 'Combined with another noun' },
	{ abbrev: 'comp.', expansion: 'compound' },
	{ abbrev: 'Comp.', expansion: 'comparative' },
	{ abbrev: 'concr.', expansion: 'concrete' },
	{ abbrev: 'conj.', expansion: 'conjunction; conjecture(s)' },
	{ abbrev: 'constr.', expansion: 'construed with; construction' },
	{ abbrev: 'def.', expansion: 'definition' },
	{ abbrev: 'dissyll.', expansion: 'dissyllable', description: 'Pronounced as two syllables' },
	{ abbrev: 'ellipt.', expansion: 'elliptical(ly)', description: 'With words omitted that are understood from context' },
	{ abbrev: 'expr.', expansion: 'expression' },
	{ abbrev: 'fig.', expansion: 'figurative(ly)' },
	{ abbrev: 'gen.', expansion: 'general(ly)' },
	{ abbrev: 'imper.', expansion: 'imperative' },
	{ abbrev: 'impers.', expansion: 'impersonal' },
	{ abbrev: 'indef.', expansion: 'indefinite' },
	{ abbrev: 'indic.', expansion: 'indicative' },
	{ abbrev: 'inf.', expansion: 'infinitive' },
	{ abbrev: 'interj.', expansion: 'interjection' },
	{ abbrev: 'intr.', expansion: 'intransitive' },
	{ abbrev: 'Irreg.', expansion: 'irregular' },
	{ abbrev: 'lit.', expansion: 'literal(ly)' },
	{ abbrev: 'monosyll.', expansion: 'monosyllable', description: 'Pronounced as one syllable' },
	{ abbrev: 'obj.', expansion: 'object' },
	{ abbrev: 'obs.', expansion: 'obsolete' },
	{ abbrev: 'pa. pple.', expansion: 'past participle' },
	{ abbrev: 'pa. t.', expansion: 'past tense' },
	{ abbrev: 'partic.', expansion: 'participle; particularly' },
	{ abbrev: 'Partic.', expansion: 'participle; particularly' },
	{ abbrev: 'pass.', expansion: 'passive' },
	{ abbrev: 'phr.', expansion: 'phrase(s)' },
	{ abbrev: 'pl.', expansion: 'plural' },
	{ abbrev: 'ppl. adj.', expansion: 'participial adjective' },
	{ abbrev: 'pple.', expansion: 'participle' },
	{ abbrev: 'prep.', expansion: 'preposition' },
	{ abbrev: 'pron.', expansion: 'pronoun' },
	{ abbrev: 'refl.', expansion: 'reflexive' },
	{ abbrev: 'Refl.', expansion: 'reflexive' },
	{ abbrev: 'sb.', expansion: 'substantive (noun)' },
	{ abbrev: 'sing.', expansion: 'singular' },
	{ abbrev: 'subst.', expansion: 'substantive (noun)' },
	{ abbrev: 'syll.', expansion: 'syllable(s)' },
	{ abbrev: 'tr.', expansion: 'transitive' },
	{ abbrev: 'trans.', expansion: 'transitive' },
	{ abbrev: 'transf.', expansion: 'in a transferred sense' },
	{ abbrev: 'trisyll.', expansion: 'trisyllable', description: 'Pronounced as three syllables' },
	{ abbrev: 'vb.', expansion: 'verb' },
	{ abbrev: 'voc.', expansion: 'vocative' },

	// ─── Reference/scholarly terms ───
	{ abbrev: 'app.', expansion: 'apparently' },
	{ abbrev: 'arch.', expansion: 'archaic' },
	{ abbrev: 'cf.', expansion: 'compare (Latin: confer)' },
	{ abbrev: 'corr.', expansion: 'corruption' },
	{ abbrev: 'dial.', expansion: 'dialect(s), dialectal(ly)' },
	{ abbrev: 'e.g.', expansion: 'for example (Latin: exempli gratia)' },
	{ abbrev: 'esp.', expansion: 'especially' },
	{ abbrev: 'etym.', expansion: 'etymology, etymological' },
	{ abbrev: 'exx.', expansion: 'examples' },
	{ abbrev: 'foll.', expansion: 'following' },
	{ abbrev: 'freq.', expansion: 'frequent(ly)' },
	{ abbrev: 'i.e.', expansion: 'that is (Latin: id est)' },
	{ abbrev: 'occas.', expansion: 'occasional(ly)' },
	{ abbrev: 'orig.', expansion: 'original(ly)' },
	{ abbrev: 'prob.', expansion: 'probably' },
	{ abbrev: 'q.v.', expansion: 'which see (Latin: quod vide)' },
	{ abbrev: 'ref.', expansion: 'reference; referred; referring' },
	{ abbrev: 'scil.', expansion: 'that is to say (Latin: scilicet)' },
	{ abbrev: 'spec.', expansion: 'specific(ally)' },
	{ abbrev: 's.v.', expansion: 'under the word (Latin: sub verbo)' },
	{ abbrev: 'usu.', expansion: 'usual(ly)' },
	{ abbrev: 'viz.', expansion: 'namely (Latin: videlicet)' },

	// ─── Edition/textual references ───
	{ abbrev: 'Ff', expansion: 'Folios', description: 'All four Shakespeare Folio editions (F1 1623, F2 1632, F3 1663, F4 1685)' },
	{ abbrev: 'F1', expansion: 'First Folio (1623)', description: 'The first collected edition of Shakespeare\'s plays' },
	{ abbrev: 'F2', expansion: 'Second Folio (1632)' },
	{ abbrev: 'F3', expansion: 'Third Folio (1663)' },
	{ abbrev: 'F4', expansion: 'Fourth Folio (1685)' },
	{ abbrev: 'Qq', expansion: 'Quartos', description: 'All Quarto editions of a play' },
	{ abbrev: 'Q1', expansion: 'First Quarto' },
	{ abbrev: 'Q2', expansion: 'Second Quarto' },
	{ abbrev: 'Q3', expansion: 'Third Quarto' },
	{ abbrev: 'M. Edd.', expansion: 'Modern Editors/Editions', description: 'Editions from Rowe (1709) onwards' },
	{ abbrev: 'O. Edd.', expansion: 'Old Editions', description: 'Pre-modern editions (Folios and Quartos)' },
	{ abbrev: 'Edd.', expansion: 'Editions' },
	{ abbrev: 'edd.', expansion: 'editions' },

	// ─── Languages ───
	{ abbrev: 'Fr.', expansion: 'French' },
	{ abbrev: 'It.', expansion: 'Italian' },
	{ abbrev: 'L.', expansion: 'Latin' },
	{ abbrev: 'O.Fr.', expansion: 'Old French' },
	{ abbrev: 'Span.', expansion: 'Spanish' },
	{ abbrev: 'Germ.', expansion: 'German' },

	// ─── Schmidt-specific compound abbreviations ───
	{ abbrev: 'Comp. Irreg. expr.', expansion: 'Comparative Irregular expression', description: 'An irregular grammatical form used in comparison' },
	{ abbrev: 'Sup. Irreg. expr.', expansion: 'Superlative Irregular expression', description: 'An irregular grammatical form used in the superlative' },
	{ abbrev: 'ind. art.', expansion: 'indefinite article' },
	{ abbrev: 'def. art.', expansion: 'definite article' },

	// ─── Scholars referenced ───
	{ abbrev: 'Cotgr.', expansion: 'Cotgrave', description: 'Randle Cotgrave\'s French-English dictionary (1611)' },
	{ abbrev: 'J.', expansion: 'Johnson', description: 'Samuel Johnson\'s edition of Shakespeare (1765)' },
	{ abbrev: 'Palsgr.', expansion: 'Palsgrave', description: 'John Palsgrave\'s French grammar (1530)' },
	{ abbrev: 'S.', expansion: 'Shakespeare; Shakespearian' },
	{ abbrev: 'pre-S.', expansion: 'pre-Shakespearian' },
	{ abbrev: 'post-S.', expansion: 'post-Shakespearian' },
	{ abbrev: 'Eliz.', expansion: 'Elizabethan' },
	{ abbrev: 'pre-Eliz.', expansion: 'pre-Elizabethan' },
];

// Build a lookup map for fast access
const abbrevMap = new Map<string, Abbreviation>();
for (const a of abbreviations) {
	abbrevMap.set(a.abbrev.toLowerCase(), a);
}

/**
 * Look up an abbreviation. Case-insensitive.
 */
export function lookupAbbreviation(abbrev: string): Abbreviation | undefined {
	return abbrevMap.get(abbrev.toLowerCase());
}

/**
 * Find all abbreviations that appear in a text string.
 * Returns matches sorted by position in the text.
 */
export function findAbbreviationsInText(text: string): { abbrev: Abbreviation; index: number }[] {
	const results: { abbrev: Abbreviation; index: number }[] = [];
	for (const a of abbreviations) {
		let idx = 0;
		while ((idx = text.indexOf(a.abbrev, idx)) !== -1) {
			results.push({ abbrev: a, index: idx });
			idx += a.abbrev.length;
		}
	}
	results.sort((a, b) => a.index - b.index);
	return results;
}
