<script lang="ts">
  import type {
    LexiconEntryDetail,
    LexiconCitationDetail,
    LexiconSubEntryDetail,
    MultiEditionScene,
    EditionLineRef,
  } from "$lib/types";
  import CorrectionForm from "./CorrectionForm.svelte";
  import SceneViewer from "./SceneViewer.svelte";
  import IconClose from "$lib/components/icons/IconClose.svelte";
  import IconFlag from "$lib/components/icons/IconFlag.svelte";
  import IconButton from "$lib/components/ui/IconButton.svelte";
  import CollapsibleSection from "$lib/components/ui/CollapsibleSection.svelte";
  import { SvelteSet } from "svelte/reactivity";
  import { groupBy } from "$lib/utils";

  let {
    entry,
    onclose,
  }: {
    entry: LexiconEntryDetail | null;
    onclose: () => void;
  } = $props();

  let expandedCitations = new SvelteSet<number>();
  let sceneData = $state<MultiEditionScene | null>(null);
  let sceneCitation = $state<LexiconCitationDetail | null>(null);
  let savedScrollTop = $state(0);
  let correctionLine = $state<{
    lineNumber: number;
    content: string;
    characterName: string | null;
  } | null>(null);
  let correctionEntry = $state<{
    type: "entry" | "citation";
    currentText: string;
    senseNumber?: number;
    subSense?: string;
    citationRef?: string;
  } | null>(null);

  let hasMultipleSubEntries = $derived(
    entry ? entry.subEntries.length > 1 : false,
  );

  const EDITION_LABELS: Record<number, string> = {
    1: "OSS",
    2: "SE",
    3: "Per",
    4: "F1",
    5: "Flg",
  };

  function formatEditionLines(refs: EditionLineRef[]): string {
    return refs
      .map(
        (r) =>
          `${EDITION_LABELS[r.edition_id] ?? r.edition_code} ${r.line_number ?? "—"}`,
      )
      .join(" · ");
  }

  // Group citations by sense_id for a given sub-entry
  function getCitationsBySense(sub: LexiconSubEntryDetail) {
    // eslint-disable-next-line svelte/prefer-svelte-reactivity -- pure data transform, not reactive state
    const bySense = new Map<number, LexiconCitationDetail[]>();
    const unassigned: LexiconCitationDetail[] = [];
    for (const c of sub.citations) {
      if (c.sense_id != null) {
        const list = bySense.get(c.sense_id) ?? [];
        list.push(c);
        bySense.set(c.sense_id, list);
      } else {
        unassigned.push(c);
      }
    }
    return { bySense, unassigned };
  }

  function groupByWork(citations: LexiconCitationDetail[]) {
    return groupBy(citations, (c) => c.work_title || c.work_abbrev || "Other");
  }

  function toggleCitation(id: number) {
    if (expandedCitations.has(id)) {
      expandedCitations.delete(id);
    } else {
      expandedCitations.add(id);
    }
  }

  function formatRef(c: LexiconCitationDetail): string {
    const parts: string[] = [];
    if (c.work_title) {
      parts.push(c.work_title);
    } else if (c.work_abbrev) {
      parts.push(c.work_abbrev);
    }
    if (c.act != null) {
      let loc = `${c.act}`;
      if (c.scene != null) loc += `.${c.scene}`;
      if (c.line != null) loc += `.${c.line}`;
      parts.push(loc);
    }
    return parts.join(" ") || c.raw_bibl || "";
  }

  function formatCitationLoc(c: LexiconCitationDetail): string {
    if (c.act != null) {
      let loc = `${c.act}`;
      if (c.scene != null) loc += `.${c.scene}`;
      if (c.line != null) loc += `.${c.line}`;
      return loc;
    }
    return c.raw_bibl || "";
  }

  function citationText(c: LexiconCitationDetail): string {
    // Prefer quote_text (Schmidt's original fragment) when available —
    // matched_line is the full line from the text edition and may not
    // contain the headword if the match confidence was low.
    if (c.quote_text) return c.quote_text;
    if (c.matched_line) return c.matched_line;
    return c.display_text || "";
  }

  function citationSpeaker(c: LexiconCitationDetail): string | null {
    return c.matched_character || null;
  }

  async function openScene(c: LexiconCitationDetail) {
    if (!c.work_id || c.act == null) return;
    const body = document.querySelector(".modal-body");
    if (body) savedScrollTop = body.scrollTop;
    sceneCitation = c;
    try {
      const scene = c.scene ?? 1;
      const res = await fetch(`/api/text/scene/${c.work_id}/${c.act}/${scene}`);
      if (res.ok) {
        sceneData = await res.json();
      }
    } finally {
      // loading done
    }
  }

  function closeScene() {
    sceneData = null;
    sceneCitation = null;
    // Restore scroll position after the entry view re-renders
    requestAnimationFrame(() => {
      const body = document.querySelector(".modal-body");
      if (body) body.scrollTop = savedScrollTop;
    });
  }

  // Reset state when entry changes
  $effect(() => {
    if (entry) {
      expandedCitations.clear();
      sceneData = null;
    }
  });

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Escape") {
      if (sceneData) {
        closeScene();
      } else {
        onclose();
      }
    }
  }
</script>

{#snippet citationList(citations: LexiconCitationDetail[])}
  {@const byWork = groupByWork(citations)}
  <div class="citation-groups">
    {#each [...byWork.entries()] as [workName, workCitations] (workName)}
      <div class="citation-work-group">
        <h4 class="work-group-title">{workName}</h4>
        <ul class="citation-list">
          {#each workCitations as citation (citation.id)}
            <li class="citation-row">
              <button
                class="citation-item"
                class:clickable={citation.work_id != null &&
                  citation.act != null}
                onclick={() => {
                  if (citation.work_id != null && citation.act != null) {
                    openScene(citation);
                  } else {
                    toggleCitation(citation.id);
                  }
                }}
              >
                <span class="citation-ref">{formatCitationLoc(citation)}</span>
                {#if citation.edition_lines && citation.edition_lines.length > 0}
                  <span class="edition-refs"
                    >{formatEditionLines(citation.edition_lines)}</span
                  >
                {/if}
                {#if citationSpeaker(citation)}
                  <span class="citation-speaker"
                    >{citationSpeaker(citation)}</span
                  >
                {/if}
                <p class="citation-quote">{citationText(citation)}</p>
              </button>
              <IconButton
                label="Flag this citation for correction"
                title="Flag this citation for correction"
                size={28}
                variant="danger"
                opacity={0}
                onclick={(e) => {
                  e.stopPropagation();
                  correctionEntry = {
                    type: "citation",
                    currentText: citationText(citation),
                    citationRef: formatRef(citation),
                  };
                }}
              >
                <IconFlag size={12} />
              </IconButton>
            </li>
          {/each}
        </ul>
      </div>
    {/each}
  </div>
{/snippet}

{#if entry}
  <div
    class="modal-backdrop"
    onclick={() => (sceneData ? closeScene() : onclose())}
    onkeydown={handleKeydown}
    role="presentation"
  ></div>

  <div
    class="modal"
    role="dialog"
    aria-label="Entry: {entry.key}"
    onkeydown={handleKeydown}
    tabindex="-1"
  >
    {#if sceneData}
      <SceneViewer
        data={sceneData}
        citation={sceneCitation}
        headword={entry?.key ?? ''}
        onback={closeScene}
        {onclose}
        {formatRef}
        citationText={citationText}
        citationSpeaker={citationSpeaker}
      />
    {:else}
      <!-- Entry detail view -->
      <div class="modal-header">
        <h2 class="entry-word">{entry.key}</h2>
        <IconButton
          label="Flag this entry for correction"
          title="Flag this entry for correction"
          size={36}
          variant="danger"
          opacity={0.4}
          onclick={() =>
            (correctionEntry = {
              type: "entry",
              currentText:
                entry.senses
                  .map(
                    (s) =>
                      `${s.sense_number}${s.sub_sense || ""}) ${s.definition_text}`,
                  )
                  .join("\n") ||
                entry.full_text ||
                entry.key,
            })}
        >
          <IconFlag size={16} />
        </IconButton>
        <IconButton onclick={onclose} label="Close" size={36}>
          <IconClose size={20} />
        </IconButton>
      </div>

      {#if entry.orthography && entry.orthography.replace(/[,.\s]+$/g, "") !== entry.key}
        <p class="orthography">{entry.orthography}</p>
      {/if}

      <div class="modal-body">
        {#each entry.subEntries as sub, subIdx (sub.id)}
          {@const citGroups = getCitationsBySense(sub)}
          {#if hasMultipleSubEntries}
            <div class="sub-entry-header" class:first={subIdx === 0}>
              <h3 class="sub-entry-key">{sub.key}</h3>
              {#if sub.entry_type}
                <span class="sub-entry-type">{sub.entry_type}</span>
              {/if}
            </div>
          {/if}

          {#if sub.senses.length > 0}
            <section class="senses" aria-label="Definitions">
              {#each sub.senses as sense (sense.id)}
                <div class="sense-block" class:sub-sense={sense.sub_sense}>
                  <div class="sense">
                    {#if sense.sub_sense}
                      <span class="sense-num sub">{sense.sub_sense})</span>
                    {:else}
                      <span class="sense-num">{sense.sense_number})</span>
                    {/if}
                    <p class="sense-def">{sense.definition_text}</p>
                  </div>
                  {#if citGroups.bySense.has(sense.id)}
                    {@const senseCitations = citGroups.bySense.get(sense.id)!}
                    <CollapsibleSection label="References" count={senseCitations.length}>
                      {@render citationList(senseCitations)}
                    </CollapsibleSection>
                  {/if}
                </div>
              {/each}
            </section>
          {:else if sub.full_text}
            <section class="full-text" aria-label="Definition">
              <p>{sub.full_text}</p>
            </section>
          {/if}

          {#if citGroups.unassigned.length > 0}
            <CollapsibleSection label="References" count={citGroups.unassigned.length}>
              {@render citationList(citGroups.unassigned)}
            </CollapsibleSection>
          {/if}
        {/each}

        {#if entry.references && entry.references.length > 0}
          {@const refsBySource = groupBy(entry.references, (r) => r.source_name)}
          <section class="reference-works">
            <h3 class="ref-section-title">Reference Works</h3>
            {#each [...refsBySource.entries()] as [sourceName, refs] (sourceName)}
              <CollapsibleSection label={sourceName} count={refs.length}>
                <ul class="ref-citation-list">
                  {#each refs as ref, refIdx (refIdx)}
                    <li class="ref-citation-item">
                      <span class="ref-location"
                        >{ref.work_title ?? ref.work_abbrev ?? ""}
                        {ref.act != null
                          ? `${ref.act}.${ref.scene ?? ""}.${ref.line ?? ""}`
                          : ""}</span
                      >
                      {#if ref.edition_lines.length > 0}
                        <span class="edition-refs"
                          >{formatEditionLines(ref.edition_lines)}</span
                        >
                      {/if}
                    </li>
                  {/each}
                </ul>
              </CollapsibleSection>
            {/each}
          </section>
        {/if}
      </div>
    {/if}
  </div>
{/if}

{#if correctionLine && sceneData && entry}
  <CorrectionForm
    type="line"
    entryKey={entry.key}
    workTitle={sceneData.work_title}
    act={sceneData.act}
    scene={sceneData.scene}
    lineNumber={correctionLine.lineNumber}
    currentText={correctionLine.content}
    characterName={correctionLine.characterName}
    editionName="Multi-edition"
    onclose={() => (correctionLine = null)}
  />
{/if}

{#if correctionEntry && entry}
  <CorrectionForm
    type={correctionEntry.type}
    entryKey={entry.key}
    currentText={correctionEntry.currentText}
    senseNumber={correctionEntry.senseNumber}
    subSense={correctionEntry.subSense}
    citationRef={correctionEntry.citationRef}
    onclose={() => (correctionEntry = null)}
  />
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: var(--color-overlay);
    z-index: 400;
  }

  .modal {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    width: 92%;
    max-width: 640px;
    max-height: 85dvh;
    background: var(--color-elevated);
    border: 1px solid var(--color-border);
    border-radius: 16px;
    z-index: 500;
    display: flex;
    flex-direction: column;
    animation: modal-in 0.2s ease-out;
    outline: none;
  }

  @keyframes modal-in {
    from {
      opacity: 0;
      transform: translate(-50%, -48%);
    }
    to {
      opacity: 1;
      transform: translate(-50%, -50%);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .modal {
      animation: none;
    }
  }

  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 20px 20px 0;
    flex-shrink: 0;
    gap: 12px;
  }

  .entry-word {
    margin: 0;
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--color-text);
    flex: 1;
  }

  .orthography {
    padding: 0 20px;
    margin: 4px 0 0;
    font-style: italic;
    color: var(--color-text-muted);
    font-size: 0.9rem;
  }

  .modal-body {
    flex: 1;
    overflow-y: auto;
    padding: 16px 20px 20px;
  }

  /* ─── Sub-entries ─── */
  .sub-entry-header {
    display: flex;
    align-items: baseline;
    gap: 8px;
    margin-top: 20px;
    padding-bottom: 6px;
    border-bottom: 1px solid var(--color-border);
  }

  .sub-entry-header.first {
    margin-top: 0;
  }

  .sub-entry-key {
    margin: 0;
    font-size: 1.1rem;
    font-weight: 700;
    color: var(--color-accent);
  }

  .sub-entry-type {
    font-size: 0.8rem;
    font-style: italic;
    color: var(--color-text-muted);
  }

  /* ─── Senses ─── */
  .senses {
    margin-bottom: 24px;
  }

  .sense-block {
    margin-bottom: 20px;
  }

  .sense-block:last-child {
    margin-bottom: 0;
  }

  .sense {
    display: flex;
    gap: 8px;
  }

  .sense-num {
    color: var(--color-accent);
    font-weight: 600;
    flex-shrink: 0;
    min-width: 24px;
  }

  .sense-num.sub {
    font-weight: 500;
    font-style: italic;
  }

  .sense-block.sub-sense {
    margin-left: 20px;
  }

  .sense-def {
    margin: 0;
    color: var(--color-text-secondary);
    font-size: 0.95rem;
    line-height: 1.65;
  }

  .full-text {
    margin-bottom: 24px;
  }

  .full-text p {
    margin: 0;
    color: var(--color-text-secondary);
    font-size: 0.95rem;
    line-height: 1.65;
  }

  /* ─── Citations ─── */
  .citation-groups {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .work-group-title {
    margin: 0 0 2px;
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--color-accent);
    opacity: 0.8;
  }

  .citation-list {
    list-style: none;
    padding: 0;
    margin: 0;
  }

  .citation-row {
    display: flex;
    align-items: flex-start;
    gap: 0;
  }

  .citation-row:hover :global(.icon-btn) {
    opacity: 0.5;
  }

  .citation-item {
    display: block;
    flex: 1;
    min-width: 0;
    padding: 8px 12px;
    border: none;
    background: none;
    text-align: left;
    cursor: pointer;
    border-radius: 8px;
    font-family: inherit;
    transition: background 0.15s;
    color: var(--color-text);
  }

  .citation-item:hover {
    background: var(--color-hover);
  }

  .citation-item:active {
    background: var(--color-active);
  }

  .citation-ref {
    display: block;
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--color-accent);
    margin-bottom: 2px;
  }

  .citation-speaker {
    display: block;
    font-size: 0.7rem;
    font-weight: 600;
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }

  .citation-quote {
    margin: 2px 0 0;
    font-size: 0.85rem;
    color: var(--color-text-secondary);
    font-style: italic;
    line-height: 1.5;
  }

  /* ─── Edition refs on citations ─── */
  .edition-refs {
    display: block;
    font-size: 0.6rem;
    color: var(--color-text-muted);
    margin-bottom: 2px;
  }

  /* ─── Reference works ─── */
  .reference-works {
    margin-top: 20px;
    padding-top: 16px;
    border-top: 1px solid var(--color-border);
  }

  .ref-section-title {
    margin: 0 0 8px;
    font-size: 0.85rem;
    font-weight: 700;
    color: var(--color-text);
  }


  .ref-citation-list {
    list-style: none;
    padding: 0 0 0 12px;
    margin: 4px 0 0;
  }

  .ref-citation-item {
    padding: 2px 0;
    font-size: 0.75rem;
    color: var(--color-text-secondary);
  }

  .ref-location {
    font-weight: 600;
    color: var(--color-text);
    margin-right: 6px;
  }

</style>
