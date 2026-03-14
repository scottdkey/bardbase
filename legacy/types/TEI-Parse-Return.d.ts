interface TEI2ParseReturn {
  value: string;
  valueKey: string;
  schmidt: {
    bibl: Sonnet | Poem | Play;
    parts: string[];
    abbreviations: Abbreviation;
  }[];
}[]

interface Sonnet {
  sonnet: string;
  line: string;
}

interface Poem {
  poem: string;
  line: string;
  poemName: string;
}

interface Play {
  play: string;
  act: string;
  scene: string;
  line: string;
  infoField: string;
  playName: string;
}