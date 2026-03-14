// Package constants contains all reference data for the Shakespeare database build.
package constants

// SchmidtWork maps a Schmidt abbreviation to its full title, Perseus text ID, and work type.
type SchmidtWork struct {
	Title     string
	PerseusID string
	WorkType  string
}

// SchmidtWorks maps Schmidt abbreviations (with and without periods) to work metadata.
var SchmidtWorks = map[string]SchmidtWork{
	// With periods
	"Tp.":    {Title: "The Tempest", PerseusID: "1999.03.0056", WorkType: "comedy"},
	"Gent.":  {Title: "Two Gentlemen of Verona", PerseusID: "1999.03.0032", WorkType: "comedy"},
	"Gentl.": {Title: "Two Gentlemen of Verona", PerseusID: "1999.03.0032", WorkType: "comedy"},
	"Wiv.":   {Title: "Merry Wives of Windsor", PerseusID: "1999.03.0059", WorkType: "comedy"},
	"Meas.":  {Title: "Measure for Measure", PerseusID: "1999.03.0049", WorkType: "comedy"},
	"Err.":   {Title: "Comedy of Errors", PerseusID: "1999.03.0039", WorkType: "comedy"},
	"Ado":    {Title: "Much Ado About Nothing", PerseusID: "1999.03.0047", WorkType: "comedy"},
	"LLL":    {Title: "Love's Labour's Lost", PerseusID: "1999.03.0048", WorkType: "comedy"},
	"Mids.":  {Title: "Midsummer Night's Dream", PerseusID: "1999.03.0051", WorkType: "comedy"},
	"Merch.": {Title: "Merchant of Venice", PerseusID: "1999.03.0050", WorkType: "comedy"},
	"As":     {Title: "As You Like It", PerseusID: "1999.03.0038", WorkType: "comedy"},
	"Shr.":   {Title: "Taming of the Shrew", PerseusID: "1999.03.0054", WorkType: "comedy"},
	"All's":  {Title: "All's Well That Ends Well", PerseusID: "1999.03.0036", WorkType: "comedy"},
	"Alls":   {Title: "All's Well That Ends Well", PerseusID: "1999.03.0036", WorkType: "comedy"},
	"Tw.":    {Title: "Twelfth Night", PerseusID: "1999.03.0057", WorkType: "comedy"},
	"Wint.":  {Title: "Winter's Tale", PerseusID: "1999.03.0060", WorkType: "comedy"},
	"John":   {Title: "King John", PerseusID: "1999.03.0033", WorkType: "history"},
	"R2":     {Title: "Richard II", PerseusID: "1999.03.0052", WorkType: "history"},
	"H4A":    {Title: "Henry IV Part 1", PerseusID: "1999.03.0041", WorkType: "history"},
	"H4B":    {Title: "Henry IV Part 2", PerseusID: "1999.03.0042", WorkType: "history"},
	"H5":     {Title: "Henry V", PerseusID: "1999.03.0043", WorkType: "history"},
	"H6A":    {Title: "Henry VI Part 1", PerseusID: "1999.03.0044", WorkType: "history"},
	"H6B":    {Title: "Henry VI Part 2", PerseusID: "1999.03.0045", WorkType: "history"},
	"H6C":    {Title: "Henry VI Part 3", PerseusID: "1999.03.0046", WorkType: "history"},
	"R3":     {Title: "Richard III", PerseusID: "1999.03.0035", WorkType: "history"},
	"H8":     {Title: "Henry VIII", PerseusID: "1999.03.0074", WorkType: "history"},
	"Troil.": {Title: "Troilus and Cressida", PerseusID: "1999.03.0058", WorkType: "comedy"},
	"Cor.":   {Title: "Coriolanus", PerseusID: "1999.03.0026", WorkType: "tragedy"},
	"Tit.":   {Title: "Titus Andronicus", PerseusID: "1999.03.0037", WorkType: "tragedy"},
	"Rom.":   {Title: "Romeo and Juliet", PerseusID: "1999.03.0053", WorkType: "tragedy"},
	"Tim.":   {Title: "Timon of Athens", PerseusID: "1999.03.0055", WorkType: "tragedy"},
	"Caes.":  {Title: "Julius Caesar", PerseusID: "1999.03.0027", WorkType: "tragedy"},
	"Mcb.":   {Title: "Macbeth", PerseusID: "1999.03.0028", WorkType: "tragedy"},
	"Hml.":   {Title: "Hamlet", PerseusID: "1999.03.0031", WorkType: "tragedy"},
	"Lr.":    {Title: "King Lear", PerseusID: "1999.03.0029", WorkType: "tragedy"},
	"Oth.":   {Title: "Othello", PerseusID: "1999.03.0034", WorkType: "tragedy"},
	"Ant.":   {Title: "Antony and Cleopatra", PerseusID: "1999.03.0025", WorkType: "tragedy"},
	"Cymb.":  {Title: "Cymbeline", PerseusID: "1999.03.0040", WorkType: "comedy"},
	"Per.":   {Title: "Pericles", PerseusID: "1999.03.0030", WorkType: "comedy"},
	"Ven.":   {Title: "Venus and Adonis", PerseusID: "1999.03.0061", WorkType: "poem"},
	"Lucr.":  {Title: "Rape of Lucrece", PerseusID: "1999.03.0062", WorkType: "poem"},
	"Sonn.":  {Title: "Sonnets", PerseusID: "1999.03.0064", WorkType: "sonnet_sequence"},
	"Pilgr.": {Title: "Passionate Pilgrim", PerseusID: "1999.03.0063", WorkType: "poem"},
	"Phoen.": {Title: "Phoenix and the Turtle", PerseusID: "1999.03.0066", WorkType: "poem"},
	"Compl.": {Title: "Lover's Complaint", PerseusID: "1999.03.0065", WorkType: "poem"},

	// Without periods (aliases)
	"Tp":    {Title: "The Tempest", PerseusID: "1999.03.0056", WorkType: "comedy"},
	"Wiv":   {Title: "Merry Wives of Windsor", PerseusID: "1999.03.0059", WorkType: "comedy"},
	"Meas":  {Title: "Measure for Measure", PerseusID: "1999.03.0049", WorkType: "comedy"},
	"Err":   {Title: "Comedy of Errors", PerseusID: "1999.03.0039", WorkType: "comedy"},
	"Mids":  {Title: "Midsummer Night's Dream", PerseusID: "1999.03.0051", WorkType: "comedy"},
	"Merch": {Title: "Merchant of Venice", PerseusID: "1999.03.0050", WorkType: "comedy"},
	"Shr":   {Title: "Taming of the Shrew", PerseusID: "1999.03.0054", WorkType: "comedy"},
	"Tw":    {Title: "Twelfth Night", PerseusID: "1999.03.0057", WorkType: "comedy"},
	"Wint":  {Title: "Winter's Tale", PerseusID: "1999.03.0060", WorkType: "comedy"},
	"Troil": {Title: "Troilus and Cressida", PerseusID: "1999.03.0058", WorkType: "comedy"},
	"Cor":   {Title: "Coriolanus", PerseusID: "1999.03.0026", WorkType: "tragedy"},
	"Tit":   {Title: "Titus Andronicus", PerseusID: "1999.03.0037", WorkType: "tragedy"},
	"Rom":   {Title: "Romeo and Juliet", PerseusID: "1999.03.0053", WorkType: "tragedy"},
	"Tim":   {Title: "Timon of Athens", PerseusID: "1999.03.0055", WorkType: "tragedy"},
	"Caes":  {Title: "Julius Caesar", PerseusID: "1999.03.0027", WorkType: "tragedy"},
	"Mcb":   {Title: "Macbeth", PerseusID: "1999.03.0028", WorkType: "tragedy"},
	"Hml":   {Title: "Hamlet", PerseusID: "1999.03.0031", WorkType: "tragedy"},
	"Lr":    {Title: "King Lear", PerseusID: "1999.03.0029", WorkType: "tragedy"},
	"Oth":   {Title: "Othello", PerseusID: "1999.03.0034", WorkType: "tragedy"},
	"Ant":   {Title: "Antony and Cleopatra", PerseusID: "1999.03.0025", WorkType: "tragedy"},
	"Cymb":  {Title: "Cymbeline", PerseusID: "1999.03.0040", WorkType: "comedy"},
	"Per":   {Title: "Pericles", PerseusID: "1999.03.0030", WorkType: "comedy"},
	"Ven":   {Title: "Venus and Adonis", PerseusID: "1999.03.0061", WorkType: "poem"},
	"Lucr":  {Title: "Rape of Lucrece", PerseusID: "1999.03.0062", WorkType: "poem"},
	"Sonn":  {Title: "Sonnets", PerseusID: "1999.03.0064", WorkType: "sonnet_sequence"},
	"Pilgr": {Title: "Passionate Pilgrim", PerseusID: "1999.03.0063", WorkType: "poem"},
	"Phoen": {Title: "Phoenix and the Turtle", PerseusID: "1999.03.0066", WorkType: "poem"},
	"Compl": {Title: "Lover's Complaint", PerseusID: "1999.03.0065", WorkType: "poem"},
}

// PerseusToSchmidt maps Perseus short work codes to Schmidt abbreviations.
var PerseusToSchmidt = map[string]string{
	"tmp": "Tp.", "tgv": "Gentl.", "wiv": "Wiv.", "mm": "Meas.",
	"err": "Err.", "ado": "Ado", "lll": "LLL", "mnd": "Mids.",
	"mv": "Merch.", "ayl": "As", "shr": "Shr.", "aww": "All's",
	"tn": "Tw.", "wt": "Wint.", "jn": "John", "r2": "R2",
	"1h4": "H4A", "2h4": "H4B", "h5": "H5", "1h6": "H6A",
	"2h6": "H6B", "3h6": "H6C", "r3": "R3", "h8": "H8",
	"tro": "Troil.", "cor": "Cor.", "tit": "Tit.", "rom": "Rom.",
	"tim": "Tim.", "jc": "Caes.", "mac": "Mcb.", "ham": "Hml.",
	"lr": "Lr.", "oth": "Oth.", "ant": "Ant.", "cym": "Cymb.",
	"per": "Per.", "ven": "Ven.", "luc": "Lucr.", "son": "Sonn.",
	"pp": "Pilgr.", "phoe": "Phoen.", "lc": "Compl.",
}
