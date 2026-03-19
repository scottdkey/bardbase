# Missing Schmidt Lexicon Entries

27 entries from the Schmidt Shakespeare Lexicon (1902) could not be fetched from the
Perseus Digital Library API. These entries exist in the `entry_list.json` index but
the Perseus `xmlchunk` endpoint returns HTTP 500 (keys with apostrophes/special chars)
or no document (keys that may be sub-entries in the original text).

## Source for reconstruction

Entries were reconstructed from:
- **Onions, C.T.** *A Shakespeare Glossary* (1911), public domain — `projects/sources/onions/`
- **Dyce, Alexander.** *A General Glossary to Shakespeare's Works* — Perseus Digital Library
- **Open Source Shakespeare** concordance — `projects/sources/oss/`
- **Shakespeare's Words** (shakespeareswords.com) — cross-reference only

## Entries (27)

### Apostrophe-truncated keys (HTTP 500 from Perseus)

| Key | Full headword | Definition | Citations |
|-----|--------------|------------|-----------|
| Cat-o | Cat-o'-mountain | A wild cat; leopard or panther | Tp. IV,1,264; Wiv. II,2,27 |
| Ill-ta | Ill-ta'en | Ill-taken, taken amiss | Wint. II,1,17 |
| Light+o | Light-o'-love | Name of an old dance tune; a loose woman | Gent. I,2,83; Ado III,4,43 |
| New-ta | New-ta'en | Newly taken, freshly captured | Mac. I,3,133 |
| Quart+d | Quart d'ecu | A quarter ecu, a small French coin | AWW IV,3,290 |
| Sicklied+o | Sicklied o'er | Covered over with a sickly hue | Ham. III,1,85 |
| Tu-whit | Tu-whit, tu-who | The cry of an owl | LLL V,2,928 |
| Venomd-mouth | Venom'd-mouth'd | Having a venomous mouth | H8 I,1,120 |

### Standalone entries alongside numbered variants (HTTP 500)

| Key | Definition | Citations |
|-----|------------|-----------|
| Damask | The colour of the damask rose; mingled red and white, or blush-red | LLL V,2,296; Cor. II,1,235; AYL III,5,123; Tw.N. II,4,114; Wint. IV,4,222 |
| Light | (cross-reference to Light1-Light6) | — |
| Lombard | A banker, money-lender (from Lombard Street) | 2H4 I,1,-- |
| Mr | Abbreviation of Master/Mister | Wiv. I,1,-- |
| Sod | Past participle of "seethe" = boiled; scalded (with tears) | Lucr. 1592; LLL IV,2,23 |
| Soul | The spiritual or immortal part of man | (numerous citations) |
| Sprite | Variant spelling of Spright = spirit | Tp. I,2,380; Sonn. 53,1 |
| Strow | Variant of Strew | (see Strew) |

### -scription/-script entries (NOT FOUND in Perseus)

These may be sub-entries within the Perseus XML rather than standalone entries.

| Key | Definition | Citations | Source |
|-----|------------|-----------|--------|
| Circumscription | Restriction, confinement, constraint | Oth. I,2,27 | Onions |
| Description | Kind, sort (in "of this description") | Mer.V. III,2,302 | Onions |
| Inscription | An engraved text | Tim. V,4,67 | Onions |
| Postscript | A note appended after the signature of a letter | Lr. I,2,-- | concordance |
| Prescript1 | (sb.) A prescribed command, direction | Ant. III,8,4 | Onions |
| Prescript2 | (adj.) Prescribed, laid down | H5 III,7,51 | Onions |
| Prescription | A claim founded upon long use | 3H6 III,3,94 | Onions |
| Proscription | Condemnation, sentence of outlawry | JC IV,1,17 | concordance |
| Subscription | Submission, obedience | Lr. III,2,18 | Onions |
| Superscript | The address or direction on a letter | LLL IV,2,137 | Onions |
| Superscription | The address written on a letter | 1H6 IV,1,53; Tim. II,2,82 | Onions |
