<script lang="ts">
  import type {
    LexiconEntryDetail,
    LexiconCitationDetail,
    LexiconSubEntryDetail,
    MultiEditionScene,
    EditionLineRef,
    ReferenceCitation,
  } from "$lib/server/queries";
  import CorrectionForm from "./CorrectionForm.svelte";
  import IconClose from "$lib/components/icons/IconClose.svelte";
  import IconBack from "$lib/components/icons/IconBack.svelte";
  import IconFlag from "$lib/components/icons/IconFlag.svelte";
  import IconButton from "$lib/components/ui/IconButton.svelte";

  let {
    entry,
    onclose,
  }: {
    entry: LexiconEntryDetail | null;
    onclose: () => void;
  } = $props();

  let expandedCitations = $state<Set<number>>(new Set());
  let sceneData = $state<MultiEditionScene | null>(null);
  let sceneHighlightLine = $state<number | null>(null);
  let sceneMatchQuality = $state<"exact" | "nearby" | "scene" | "unmatched">(
    "exact",
  );
  let sceneCitation = $state<LexiconCitationDetail | null>(null);
  let sceneLoading = $state(false);
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
  let visibleEditions = $state<number[]>([]);

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

  function groupRefsBySource(
    refs: ReferenceCitation[],
  ): Map<string, ReferenceCitation[]> {
    const groups = new Map<string, ReferenceCitation[]>();
    for (const r of refs) {
      const list = groups.get(r.source_name) ?? [];
      list.push(r);
      groups.set(r.source_name, list);
    }
    return groups;
  }

  // Group citations by sense_id for a given sub-entry
  function getCitationsBySense(sub: LexiconSubEntryDetail) {
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

  // Group citations by work/play
  function groupByWork(
    citations: LexiconCitationDetail[],
  ): Map<string, LexiconCitationDetail[]> {
    const groups = new Map<string, LexiconCitationDetail[]>();
    for (const c of citations) {
      const key = c.work_title || c.work_abbrev || "Other";
      const list = groups.get(key) ?? [];
      list.push(c);
      groups.set(key, list);
    }
    return groups;
  }

  function toggleCitation(id: number) {
    const next = new Set(expandedCitations);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    expandedCitations = next;
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

  function scrollToHighlight() {
    scrollToHighlightRow();
  }

  /**
   * Find the best row index to highlight by validating the headword is present.
   * Searches across all editions in the aligned rows.
   */
  function findHeadwordRow(
    data: MultiEditionScene,
    targetLine: number | null,
    targetEditionId: number | null,
    headword: string,
  ): {
    rowIndex: number | null;
    quality: "exact" | "nearby" | "scene" | "unmatched";
  } {
    if (!headword || data.rows.length === 0)
      return { rowIndex: null, quality: "unmatched" };

    const hw = headword.replace(/\d+$/, "").toLowerCase();
    const hwEscaped = hw.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    const hwPattern = new RegExp(`\\b${hwEscaped}`, "i");

    // Find row matching the target line number in the target edition
    let targetIdx = -1;
    if (targetLine != null) {
      const edId = targetEditionId ?? 3;
      targetIdx = data.rows.findIndex((r) => {
        const ed = r.editions[edId];
        return ed && ed.line_number === targetLine;
      });
      // If not found in preferred edition, try any edition
      if (targetIdx < 0) {
        targetIdx = data.rows.findIndex((r) =>
          Object.values(r.editions).some(
            (ed) => ed && ed.line_number === targetLine,
          ),
        );
      }
    }

    // Check if target row contains the headword
    if (targetIdx >= 0) {
      const row = data.rows[targetIdx];
      if (
        Object.values(row.editions).some(
          (ed) => ed && hwPattern.test(ed.content),
        )
      ) {
        return { rowIndex: targetIdx, quality: "exact" };
      }
      // Search nearby (±5)
      for (let offset = 1; offset <= 5; offset++) {
        for (const idx of [targetIdx - offset, targetIdx + offset]) {
          if (idx >= 0 && idx < data.rows.length) {
            const r = data.rows[idx];
            if (
              Object.values(r.editions).some(
                (ed) => ed && hwPattern.test(ed.content),
              )
            ) {
              return { rowIndex: idx, quality: "nearby" };
            }
          }
        }
      }
    }

    // Fall back: any row in scene
    const matchIdx = data.rows.findIndex((r) =>
      Object.values(r.editions).some((ed) => ed && hwPattern.test(ed.content)),
    );
    if (matchIdx >= 0) return { rowIndex: matchIdx, quality: "scene" };

    return {
      rowIndex: targetIdx >= 0 ? targetIdx : null,
      quality: "unmatched",
    };
  }

  function scrollToHighlightRow() {
    if (sceneHighlightLine == null) return;
    requestAnimationFrame(() => {
      const el = document.getElementById(`scene-row-${sceneHighlightLine}`);
      el?.scrollIntoView({ behavior: "instant", block: "center" });
    });
  }

  async function openScene(c: LexiconCitationDetail) {
    if (!c.work_id || c.act == null) return;
    const body = document.querySelector(".modal-body");
    if (body) savedScrollTop = body.scrollTop;
    sceneLoading = true;
    sceneCitation = c;
    try {
      const scene = c.scene ?? 1;
      const res = await fetch(`/api/text/scene/${c.work_id}/${c.act}/${scene}`);
      if (res.ok) {
        const data: MultiEditionScene = await res.json();
        sceneData = data;
        // Default to showing 2 editions: the matched edition + OSS (or first two available)
        const matchedEd = c.matched_edition_id ?? 3;
        const available = data.available_editions.map((e) => e.id);
        if (available.length <= 2) {
          visibleEditions = available;
        } else {
          const first = available.includes(matchedEd)
            ? matchedEd
            : available[0];
          const second = available.find((id) => id !== first) ?? available[0];
          visibleEditions = [first, second];
        }
        const candidateLine = c.matched_line_number ?? c.line;
        const result = findHeadwordRow(
          data,
          candidateLine,
          c.matched_edition_id,
          entry?.key ?? "",
        );
        sceneHighlightLine = result.rowIndex;
        sceneMatchQuality = result.quality;
        scrollToHighlightRow();
      }
    } finally {
      sceneLoading = false;
    }
  }

  function closeScene() {
    sceneData = null;
    sceneHighlightLine = null;
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
      expandedCitations = new Set();
      sceneData = null;
      sceneHighlightLine = null;
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
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
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
      <!-- Multi-edition scene viewer -->
      <div class="modal-header">
        <IconButton onclick={closeScene} label="Back to entry" size={36}>
          <IconBack size={20} />
        </IconButton>
        <div class="scene-title">
          <h2>{sceneData.work_title}</h2>
          {#if sceneData.work_title === "Sonnets"}
            <span class="scene-location">Sonnet {sceneData.scene}</span>
          {:else if sceneData.scene === 0}
            <span class="scene-location">{sceneData.work_title}</span>
          {:else}
            <span class="scene-location"
              >Act {sceneData.act}, Scene {sceneData.scene}</span
            >
          {/if}
        </div>
        {#if sceneHighlightLine != null}
          <button
            class="jump-btn"
            class:review={sceneMatchQuality === "scene" ||
              sceneMatchQuality === "unmatched"}
            onclick={scrollToHighlight}
            aria-label="Jump to referenced line"
          >
            Row {sceneHighlightLine}
            {#if sceneMatchQuality === "scene"}
              <span
                class="match-flag"
                title="Headword found elsewhere in scene — needs review">?</span
              >
            {:else if sceneMatchQuality === "unmatched"}
              <span
                class="match-flag"
                title="Headword not found in scene — needs review">!</span
              >
            {/if}
          </button>
        {/if}
        <IconButton onclick={onclose} label="Close" size={36}>
          <IconClose size={20} />
        </IconButton>
      </div>
      {#if sceneCitation}
        <div class="scene-citation-context">
          <span class="context-ref">{formatRef(sceneCitation)}</span>
          {#if citationSpeaker(sceneCitation)}
            <span class="context-speaker">{citationSpeaker(sceneCitation)}</span
            >
          {/if}
          <p class="context-quote">{citationText(sceneCitation)}</p>
        </div>
      {/if}
      <!-- Edition selector -->
      <div class="edition-selector">
        {#each sceneData.available_editions as ed (ed.id)}
          <label class="edition-toggle">
            <input
              type="checkbox"
              checked={visibleEditions.includes(ed.id)}
              onchange={() => {
                if (visibleEditions.includes(ed.id)) {
                  if (visibleEditions.length > 1) {
                    visibleEditions = visibleEditions.filter(
                      (id) => id !== ed.id,
                    );
                  }
                } else {
                  visibleEditions = [...visibleEditions, ed.id];
                }
              }}
            />
            <span class="edition-label">{EDITION_LABELS[ed.id] ?? ed.code}</span
            >
          </label>
        {/each}
      </div>
      <div class="modal-body scene-body">
        <div
          class="scene-columns"
          style="--col-count: {visibleEditions.length}"
        >
          <!-- Column headers -->
          <div class="column-headers">
            {#each visibleEditions as edId}
              {@const ed = sceneData.available_editions.find(
                (e) => e.id === edId,
              )}
              <div class="col-header">{ed?.name ?? EDITION_LABELS[edId]}</div>
            {/each}
          </div>
          <!-- Aligned rows -->
          {#each sceneData.rows as row, rowIdx}
            <div
              id="scene-row-{rowIdx}"
              class="aligned-row"
              class:highlighted={rowIdx === sceneHighlightLine}
              class:needs-review={rowIdx === sceneHighlightLine &&
                (sceneMatchQuality === "scene" ||
                  sceneMatchQuality === "unmatched")}
            >
              {#each visibleEditions as edId}
                {@const cell = row.editions[edId]}
                <div
                  class="edition-cell"
                  class:empty={!cell}
                  class:stage-direction={cell?.content_type ===
                    "stage_direction"}
                >
                  {#if cell}
                    <span class="line-number">{cell.line_number ?? ""}</span>
                    <span class="line-content">{cell.content}</span>
                  {:else}
                    <span class="line-empty">—</span>
                  {/if}
                </div>
              {/each}
            </div>
          {/each}
        </div>
      </div>
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
              {#each sub.senses as sense}
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
                    <details class="sense-citations">
                      <summary class="refs-toggle"
                        >References ({senseCitations.length})</summary
                      >
                      {@render citationList(senseCitations)}
                    </details>
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
            <details class="citations-section">
              <summary class="refs-toggle"
                >References ({citGroups.unassigned.length})</summary
              >
              {@render citationList(citGroups.unassigned)}
            </details>
          {/if}
        {/each}

        {#if entry.references && entry.references.length > 0}
          {@const refsBySource = groupRefsBySource(entry.references)}
          <section class="reference-works">
            <h3 class="ref-section-title">Reference Works</h3>
            {#each [...refsBySource.entries()] as [sourceName, refs] (sourceName)}
              <details class="ref-source-group">
                <summary class="refs-toggle"
                  >{sourceName} ({refs.length})</summary
                >
                <ul class="ref-citation-list">
                  {#each refs as ref}
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
              </details>
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
    editionName={"Multi-edition"}
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

  .scene-title {
    flex: 1;
    min-width: 0;
  }

  .scene-title h2 {
    margin: 0;
    font-size: 1.1rem;
    font-weight: 700;
    color: var(--color-text);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .scene-location {
    font-size: 0.8rem;
    color: var(--color-text-muted);
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

  .sense-citations {
    margin-top: 8px;
    padding-left: 32px;
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

  /* ─── Collapsible refs toggle ─── */
  .refs-toggle {
    margin: 0 0 4px;
    font-size: 0.7rem;
    font-weight: 600;
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.06em;
    cursor: pointer;
    list-style: none;
  }

  .refs-toggle::-webkit-details-marker {
    display: none;
  }

  .refs-toggle::before {
    content: "\25B6";
    display: inline-block;
    font-size: 0.55rem;
    margin-right: 6px;
    transition: transform 0.15s;
  }

  :global(details[open]) > .refs-toggle::before {
    transform: rotate(90deg);
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

  .jump-btn {
    display: flex;
    align-items: center;
    padding: 4px 10px;
    border: 1px solid var(--color-accent);
    background: none;
    color: var(--color-accent);
    font-family: inherit;
    font-size: 0.7rem;
    font-weight: 600;
    cursor: pointer;
    border-radius: 6px;
    flex-shrink: 0;
    white-space: nowrap;
  }

  .jump-btn.review {
    border-color: #e8a735;
    color: #e8a735;
  }

  .match-flag {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    height: 16px;
    margin-left: 4px;
    border-radius: 50%;
    background: #e8a735;
    color: #1a1a2e;
    font-size: 0.6rem;
    font-weight: 800;
  }

  .jump-btn:hover {
    background: var(--color-hover);
  }

  /* ─── Citation context in scene viewer ─── */
  .scene-citation-context {
    padding: 8px 20px;
    background: var(--color-hover);
    border-bottom: 1px solid var(--color-border);
    flex-shrink: 0;
  }

  .context-ref {
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--color-accent);
  }

  .context-speaker {
    font-size: 0.65rem;
    font-weight: 600;
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.03em;
    margin-left: 8px;
  }

  .context-quote {
    margin: 2px 0 0;
    font-size: 0.8rem;
    color: var(--color-text-secondary);
    font-style: italic;
    line-height: 1.4;
  }

  /* ─── Scene text viewer ─── */
  .scene-body {
    display: flex;
    flex-direction: column;
    align-items: center;
    font-size: 16px;
    line-height: 1.7;
  }

  .line-number {
    font-size: 0.7rem;
    color: var(--color-text-muted);
    min-width: 28px;
    text-align: right;
    flex-shrink: 0;
    user-select: none;
  }

  .line-content {
    color: var(--color-text);
  }

  /* ─── Edition refs on citations ─── */
  .edition-refs {
    display: block;
    font-size: 0.6rem;
    color: var(--color-text-muted);
    margin-bottom: 2px;
  }

  /* ─── Edition selector ─── */
  .edition-selector {
    display: flex;
    gap: 6px;
    padding: 6px 20px;
    border-bottom: 1px solid var(--color-border);
    flex-shrink: 0;
    flex-wrap: wrap;
  }

  .edition-toggle {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 0.65rem;
    font-weight: 600;
    color: var(--color-text-muted);
    cursor: pointer;
  }

  .edition-toggle input {
    margin: 0;
    width: 14px;
    height: 14px;
    cursor: pointer;
  }

  .edition-label {
    user-select: none;
  }

  /* ─── Multi-edition columns ─── */
  .scene-columns {
    width: 100%;
  }

  .column-headers {
    display: grid;
    grid-template-columns: repeat(var(--col-count), 1fr);
    gap: 1px;
    position: sticky;
    top: 0;
    z-index: 1;
    background: var(--color-surface);
    border-bottom: 1px solid var(--color-border);
  }

  .col-header {
    padding: 6px 8px;
    font-size: 0.6rem;
    font-weight: 700;
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    text-align: center;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .aligned-row {
    display: grid;
    grid-template-columns: repeat(var(--col-count), 1fr);
    gap: 1px;
    border-bottom: 1px solid color-mix(in srgb, var(--color-border) 40%, transparent);
    min-height: 24px;
  }

  .aligned-row.highlighted {
    background: var(--color-active);
    border-left: 3px solid var(--color-accent);
  }

  .aligned-row.needs-review {
    border-left-color: #e8a735;
    background: rgba(232, 167, 53, 0.1);
  }

  .edition-cell {
    display: flex;
    gap: 4px;
    align-items: baseline;
    padding: 2px 6px;
    font-size: 0.8rem;
    line-height: 1.5;
    min-width: 0;
  }

  .edition-cell.empty {
    opacity: 0.3;
  }

  .edition-cell.stage-direction {
    font-style: italic;
  }

  .edition-cell.stage-direction .line-content {
    color: var(--color-text-muted);
  }

  .line-empty {
    color: var(--color-text-muted);
    font-size: 0.7rem;
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

  .ref-source-group {
    margin-bottom: 8px;
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

  @media (max-width: 600px) {
    .scene-columns {
      overflow-x: auto;
    }

    .column-headers,
    .aligned-row {
      min-width: calc(var(--col-count) * 180px);
    }
  }
</style>
