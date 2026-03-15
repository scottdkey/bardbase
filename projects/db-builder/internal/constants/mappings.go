package constants

// OSSToSchmidt maps OSS/Moby work IDs to Schmidt abbreviations.
// Loaded from projects/data/oss_to_schmidt.json.
var OSSToSchmidt map[string]string

// SEPlayRepos maps Standard Ebooks repo names to OSS work IDs.
// Loaded from projects/data/se_play_repos.json.
var SEPlayRepos map[string]string

// SEPoetryMap maps SE poetry article IDs to OSS work IDs.
// Loaded from projects/data/se_poetry_map.json.
var SEPoetryMap map[string]string

// FolgerSlugs maps OSS work IDs to Folger Shakespeare Library URL slugs.
// Loaded from projects/data/folger_slugs.json.
var FolgerSlugs map[string]string

// GenreMap maps OSS single-letter genre codes to full work type names.
// Loaded from projects/data/genre_map.json.
var GenreMap map[string]string
