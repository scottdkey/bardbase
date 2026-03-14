interface EntryFree {
  _: string;
  $: EntryFreeMeta;
  orth: Orth;
  bibl?: Bibl[];
  emph?: string[];
  cit?: Cit[];
  lb?: string;
  hi?: {
    _: string;
    $: {
      rend: string;
    };
  };
  foreign?: string;
  pb?: {
    $: {
      n: string;
    };
  };
}

interface Cit {
  quote: string;
  bibl: Bibl;
}

interface EntryOriginal {
  $: EntryMeta;
  entryFree: EntryFree;
}

interface MappedDescription {
  value: string;
  bibl: {
    title: string;
    act?: number;
    scene?: number;
    line?: number;
  }[];
}

interface MappedCitation {
  quote: string;
  line?: number;
  act?: number;
  scene?: number;
  title?: string;
  isSonnet?: boolean;
  isPoem?: boolean;
}

interface MappedObject {
  key: string;
  value: string;
  letterGrouping: string;
  description: MappedDescription[];
  citations: MappedCitation[];
  emph?: string;
  bibl?: Bibl[];
}

interface EntryData {
  original: EntryOriginal;
  mappedObject: MappedObject;
  bibl?: Bibl[];
  emph?: string;
}

interface EntryItem {
  entry: EntryData;
  previousWord: string | null;
  currentWord: string;
  nextWord: string | null;
}

interface EntryMeta {
  type: string;
  n: string;
  org: string;
  sample: string;
}

interface EntryFreeMeta {
  key: string;
  type: string;
  opt: string;
}

interface Orth {
  _: string;
  $: {
    extend?: string;
    extent?: string;
    opt: string;
  };
}

interface Bibl {
  _: string;
  $: {
    n: string;
    default: string;
    valid: string;
  };
}

interface Body {
  div1: Div1;
}

interface Div1 {
  $: EntryMeta;
  entryFree: EntryFree;
}

interface Text {
  body: Body;
}

interface TEI2 {
  "TEI.2": {
    text: Text;
  };
}
