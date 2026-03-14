package constants

// OSSToSchmidt maps OSS/Moby work IDs to Schmidt abbreviations.
var OSSToSchmidt = map[string]string{
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

// SEPlayRepos maps Standard Ebooks repo names to OSS work IDs.
var SEPlayRepos = map[string]string{
	"william-shakespeare_hamlet":                         "hamlet",
	"william-shakespeare_romeo-and-juliet":               "romeojuliet",
	"william-shakespeare_the-tempest":                    "tempest",
	"william-shakespeare_twelfth-night":                  "12night",
	"william-shakespeare_king-lear":                      "kinglear",
	"william-shakespeare_julius-caesar":                  "juliuscaesar",
	"william-shakespeare_antony-and-cleopatra":           "antonycleo",
	"william-shakespeare_henry-vi-part-ii":               "henry6p2",
	"william-shakespeare_the-merchant-of-venice":         "merchantvenice",
	"william-shakespeare_a-midsummer-nights-dream":       "midsummer",
	"william-shakespeare_othello":                        "othello",
	"william-shakespeare_macbeth":                        "macbeth",
	"william-shakespeare_cymbeline":                      "cymbeline",
	"william-shakespeare_pericles":                       "pericles",
	"william-shakespeare_coriolanus":                     "coriolanus",
	"william-shakespeare_henry-viii":                     "henry8",
	"william-shakespeare_richard-iii":                    "richard3",
	"william-shakespeare_richard-ii":                     "richard2",
	"william-shakespeare_henry-v":                        "henry5",
	"william-shakespeare_titus-andronicus":               "titus",
	"william-shakespeare_king-john":                      "kingjohn",
	"william-shakespeare_loves-labours-lost":             "loveslabours",
	"william-shakespeare_the-winters-tale":               "winterstale",
	"william-shakespeare_troilus-and-cressida":           "troilus",
	"william-shakespeare_timon-of-athens":                "timonathens",
	"william-shakespeare_measure-for-measure":            "measure",
	"william-shakespeare_henry-vi-part-iii":              "henry6p3",
	"william-shakespeare_henry-iv-part-i":                "henry4p1",
	"william-shakespeare_henry-vi-part-i":                "henry6p1",
	"william-shakespeare_henry-iv-part-ii":               "henry4p2",
	"william-shakespeare_the-comedy-of-errors":           "comedyerrors",
	"william-shakespeare_as-you-like-it":                 "asyoulikeit",
	"william-shakespeare_much-ado-about-nothing":         "muchado",
	"william-shakespeare_the-two-gentlemen-of-verona":    "twogents",
	"william-shakespeare_the-merry-wives-of-windsor":     "merrywives",
	"william-shakespeare_alls-well-that-ends-well":       "allswell",
	"william-shakespeare_the-taming-of-the-shrew":        "tamingshrew",
}

// SEPoetryMap maps SE poetry article IDs to OSS work IDs.
var SEPoetryMap = map[string]string{
	"venus-and-adonis":            "venusadonis",
	"the-rape-of-lucrece":         "rapelucrece",
	"the-passionate-pilgrim":      "passionatepilgrim",
	"the-pheonix-and-the-turtle":  "phoenixturtle",
}

// FolgerSlugs maps OSS work IDs to Folger Shakespeare Library URL slugs.
var FolgerSlugs = map[string]string{
	"tempest": "the-tempest", "twogents": "the-two-gentlemen-of-verona",
	"merrywives": "the-merry-wives-of-windsor", "measure": "measure-for-measure",
	"comedyerrors": "the-comedy-of-errors", "muchado": "much-ado-about-nothing",
	"loveslabours": "loves-labors-lost", "midsummer": "a-midsummer-nights-dream",
	"merchantvenice": "the-merchant-of-venice", "asyoulikeit": "as-you-like-it",
	"tamingshrew": "the-taming-of-the-shrew", "allswell": "alls-well-that-ends-well",
	"12night": "twelfth-night", "winterstale": "the-winters-tale",
	"kingjohn": "king-john", "richard2": "richard-ii",
	"henry4p1": "henry-iv-part-1", "henry4p2": "henry-iv-part-2",
	"henry5": "henry-v", "henry6p1": "henry-vi-part-1",
	"henry6p2": "henry-vi-part-2", "henry6p3": "henry-vi-part-3",
	"richard3": "richard-iii", "henry8": "henry-viii",
	"troilus": "troilus-and-cressida", "coriolanus": "coriolanus",
	"titus": "titus-andronicus", "romeojuliet": "romeo-and-juliet",
	"timonathens": "timon-of-athens", "juliuscaesar": "julius-caesar",
	"macbeth": "macbeth", "hamlet": "hamlet",
	"kinglear": "king-lear", "othello": "othello",
	"antonycleo": "antony-and-cleopatra", "cymbeline": "cymbeline",
	"pericles": "pericles",
}

// GenreMap maps OSS single-letter genre codes to full work type names.
var GenreMap = map[string]string{
	"c": "comedy",
	"h": "history",
	"t": "tragedy",
	"p": "poem",
	"s": "sonnet_sequence",
}
