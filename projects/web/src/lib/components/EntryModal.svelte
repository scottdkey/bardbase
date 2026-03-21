<script lang="ts">
	import type { LexiconEntryDetail, LexiconCitationDetail, LexiconSubEntryDetail, SceneTextResult } from '$lib/server/queries';
	import CorrectionForm from './CorrectionForm.svelte';
	import { corrections } from '$lib/stores/corrections.svelte';

	let {
		entry,
		onclose
	}: {
		entry: LexiconEntryDetail | null;
		onclose: () => void;
	} = $props();

	let expandedCitations = $state<Set<number>>(new Set());
	let sceneData = $state<SceneTextResult | null>(null);
	let sceneHighlightLine = $state<number | null>(null);
	let sceneMatchQuality = $state<'exact' | 'nearby' | 'scene' | 'unmatched'>('exact');
	let sceneCitation = $state<LexiconCitationDetail | null>(null);
	let sceneLoading = $state(false);
	let savedScrollTop = $state(0);
	let correctionLine = $state<{ lineNumber: number; content: string; characterName: string | null } | null>(null);
	let correctionEntry = $state<{ type: 'entry' | 'citation'; currentText: string; senseNumber?: number; subSense?: string; citationRef?: string } | null>(null);

	let hasMultipleSubEntries = $derived(entry ? entry.subEntries.length > 1 : false);

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
	function groupByWork(citations: LexiconCitationDetail[]): Map<string, LexiconCitationDetail[]> {
		const groups = new Map<string, LexiconCitationDetail[]>();
		for (const c of citations) {
			const key = c.work_title || c.work_abbrev || 'Other';
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
		return parts.join(' ') || c.raw_bibl || '';
	}

	function formatCitationLoc(c: LexiconCitationDetail): string {
		if (c.act != null) {
			let loc = `${c.act}`;
			if (c.scene != null) loc += `.${c.scene}`;
			if (c.line != null) loc += `.${c.line}`;
			return loc;
		}
		return c.raw_bibl || '';
	}

	function citationText(c: LexiconCitationDetail): string {
		if (c.matched_line) return c.matched_line;
		return c.quote_text || c.display_text || '';
	}

	function citationSpeaker(c: LexiconCitationDetail): string | null {
		return c.matched_character || null;
	}

	function scrollToHighlight() {
		if (sceneHighlightLine == null) return;
		requestAnimationFrame(() => {
			const el = document.getElementById(`scene-line-${sceneHighlightLine}`);
			el?.scrollIntoView({ behavior: 'instant', block: 'center' });
		});
	}

	/**
	 * Find the best line to highlight by validating the headword is present.
	 * Returns the line number and match quality.
	 */
	function findHeadwordLine(
		lines: { line_number: number | null; content: string }[],
		targetLine: number | null,
		headword: string
	): { line: number | null; quality: 'exact' | 'nearby' | 'scene' | 'unmatched' } {
		if (!headword || lines.length === 0) return { line: targetLine, quality: 'unmatched' };

		const hw = headword.replace(/\d+$/, '').toLowerCase();
		const hwEscaped = hw.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
		const hwPattern = new RegExp(`\\b${hwEscaped}`, 'i');

		// Check if target line already contains the headword
		if (targetLine != null) {
			const targetLineData = lines.find(l => l.line_number === targetLine);
			if (targetLineData && hwPattern.test(targetLineData.content)) {
				return { line: targetLine, quality: 'exact' };
			}
		}

		// Search nearby lines (±5)
		const targetIdx = targetLine != null ? lines.findIndex(l => l.line_number === targetLine) : -1;
		if (targetIdx >= 0) {
			for (let offset = 1; offset <= 5; offset++) {
				for (const idx of [targetIdx - offset, targetIdx + offset]) {
					if (idx >= 0 && idx < lines.length && lines[idx].line_number != null && hwPattern.test(lines[idx].content)) {
						return { line: lines[idx].line_number, quality: 'nearby' };
					}
				}
			}
		}

		// Fall back: any line in scene — flag for review
		const match = lines.find(l => l.line_number != null && hwPattern.test(l.content));
		if (match) return { line: match.line_number, quality: 'scene' };

		// Nothing found — flag for review
		return { line: targetLine, quality: 'unmatched' };
	}

	async function openScene(c: LexiconCitationDetail) {
		if (!c.work_id || c.act == null || c.scene == null) return;
		// Save scroll position of the entry body
		const body = document.querySelector('.modal-body');
		if (body) savedScrollTop = body.scrollTop;
		sceneLoading = true;
		sceneCitation = c;
		try {
			let url = `/api/text/scene?workId=${c.work_id}&act=${c.act}&scene=${c.scene}`;
			if (c.matched_edition_id != null) url += `&editionId=${c.matched_edition_id}`;
			const res = await fetch(url);
			if (res.ok) {
				const data: SceneTextResult = await res.json();
				sceneData = data;
				const candidateLine = c.matched_line_number ?? c.line;
				const result = findHeadwordLine(data.lines, candidateLine, entry?.key ?? '');
				sceneHighlightLine = result.line;
				sceneMatchQuality = result.quality;
				scrollToHighlight();
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
			const body = document.querySelector('.modal-body');
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
		if (e.key === 'Escape') {
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
						<li>
							<button
								class="citation-item"
								class:clickable={citation.work_id != null && citation.act != null && citation.scene != null}
								onclick={() => {
									if (citation.work_id != null && citation.act != null && citation.scene != null) {
										openScene(citation);
									} else {
										toggleCitation(citation.id);
									}
								}}
								oncontextmenu={(e) => { e.preventDefault(); correctionEntry = { type: 'citation', currentText: citationText(citation), citationRef: formatRef(citation) }; }}
							>
								<span class="citation-ref">{formatCitationLoc(citation)}</span>
								{#if citationSpeaker(citation)}
									<span class="citation-speaker">{citationSpeaker(citation)}</span>
								{/if}
								<p class="citation-quote">{citationText(citation)}</p>
							</button>
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
		onclick={() => sceneData ? closeScene() : onclose()}
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
			<!-- Scene text viewer -->
			<div class="modal-header">
				<button class="back-btn" onclick={closeScene} aria-label="Back to entry">
					<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<polyline points="15 18 9 12 15 6" />
					</svg>
				</button>
				<div class="scene-title">
					<h2>{sceneData.work_title}</h2>
					<span class="scene-location">Act {sceneData.act}, Scene {sceneData.scene}</span>
				</div>
				{#if sceneHighlightLine != null}
					<button class="jump-btn" class:review={sceneMatchQuality === 'scene' || sceneMatchQuality === 'unmatched'} onclick={scrollToHighlight} aria-label="Jump to referenced line">
						Line {sceneHighlightLine}
						{#if sceneMatchQuality === 'scene'}
							<span class="match-flag" title="Headword found elsewhere in scene — needs review">?</span>
						{:else if sceneMatchQuality === 'unmatched'}
							<span class="match-flag" title="Headword not found in scene — needs review">!</span>
						{/if}
					</button>
				{/if}
				<button class="close-btn" onclick={onclose} aria-label="Close">
					<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<line x1="18" y1="6" x2="6" y2="18" />
						<line x1="6" y1="6" x2="18" y2="18" />
					</svg>
				</button>
			</div>
			{#if sceneCitation}
				<div class="scene-citation-context">
					<span class="context-ref">{formatRef(sceneCitation)}</span>
					{#if citationSpeaker(sceneCitation)}
						<span class="context-speaker">{citationSpeaker(sceneCitation)}</span>
					{/if}
					<p class="context-quote">{citationText(sceneCitation)}</p>
				</div>
			{/if}
			<div class="modal-body scene-body">
				<div class="scene-lines">
					{#each sceneData.lines as line, i (line.id)}
						{@const prevSpeaker = i > 0 ? sceneData.lines[i - 1].character_name : null}
						{@const showSpeaker = line.character_name && line.character_name !== prevSpeaker && line.content_type !== 'stage_direction'}
						{#if showSpeaker}
							<div class="speaker-name">{line.character_name}</div>
						{/if}
						<div
							id="scene-line-{line.line_number}"
							class="text-line"
							class:highlighted={line.line_number === sceneHighlightLine}
							class:needs-review={line.line_number === sceneHighlightLine && (sceneMatchQuality === 'scene' || sceneMatchQuality === 'unmatched')}
							class:flagged={line.line_number != null && sceneData && corrections.isFlagged(sceneData.work_title, sceneData.act, sceneData.scene, line.line_number)}
							class:stage-direction={line.content_type === 'stage_direction'}
						>
							<span class="line-number">{line.line_number ?? ''}</span>
							<span class="line-content">{line.content}</span>
							{#if line.line_number != null}
								<button
									class="flag-btn"
									title="Flag for correction"
									onclick={(e) => { e.stopPropagation(); correctionLine = { lineNumber: line.line_number!, content: line.content, characterName: line.character_name }; }}
								>
									<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
										<path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z" />
										<line x1="4" y1="22" x2="4" y2="15" />
									</svg>
								</button>
							{/if}
						</div>
					{/each}
				</div>
			</div>
		{:else}
			<!-- Entry detail view -->
			<div class="modal-header">
				<h2 class="entry-word">{entry.key}</h2>
				<button
					class="entry-flag-btn"
					title="Flag this entry for correction"
					onclick={() => correctionEntry = { type: 'entry', currentText: entry.senses.map(s => `${s.sense_number}${s.sub_sense || ''}) ${s.definition_text}`).join('\n') || entry.full_text || entry.key }}
				>
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z" />
						<line x1="4" y1="22" x2="4" y2="15" />
					</svg>
				</button>
				<button class="close-btn" onclick={onclose} aria-label="Close">
					<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<line x1="18" y1="6" x2="6" y2="18" />
						<line x1="6" y1="6" x2="18" y2="18" />
					</svg>
				</button>
			</div>

			{#if entry.orthography && entry.orthography.replace(/[,.\s]+$/g, '') !== entry.key}
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
											<summary class="refs-toggle">References ({senseCitations.length})</summary>
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
							<summary class="refs-toggle">References ({citGroups.unassigned.length})</summary>
							{@render citationList(citGroups.unassigned)}
						</details>
					{/if}
				{/each}
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
		editionName={sceneData.edition_name}
		onclose={() => correctionLine = null}
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
		onclose={() => correctionEntry = null}
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

	.close-btn,
	.back-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 36px;
		height: 36px;
		border: none;
		background: none;
		color: var(--color-text-muted);
		cursor: pointer;
		border-radius: 8px;
		flex-shrink: 0;
	}

	.close-btn:hover,
	.back-btn:hover {
		background: var(--color-hover);
		color: var(--color-text);
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

	.sub-entry-text {
		margin: 6px 0 12px;
		font-size: 0.85rem;
		color: var(--color-text-secondary);
		line-height: 1.6;
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
		content: '\25B6';
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

	.citation-work-group {
		/* nothing extra needed */
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

	.citation-item {
		display: block;
		width: 100%;
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

	.scene-lines {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
	}

	.speaker-name {
		font-weight: 700;
		color: var(--color-accent);
		font-size: 0.8rem;
		text-transform: uppercase;
		letter-spacing: 0.03em;
		text-align: center;
		align-self: stretch;
		margin-top: 16px;
		margin-bottom: 4px;
	}

	.speaker-name:first-child {
		margin-top: 0;
	}

	.text-line {
		display: flex;
		gap: 6px;
		align-items: baseline;
		padding: 1px 8px;
		border-radius: 4px;
	}

	.text-line.highlighted {
		background: var(--color-active);
		border-left: 3px solid var(--color-accent);
		padding-left: 5px;
	}

	.text-line.needs-review {
		border-left-color: #e8a735;
		background: rgba(232, 167, 53, 0.1);
	}

	.text-line.flagged {
		border-left: 3px solid #e85535;
		padding-left: 5px;
		background: rgba(232, 85, 53, 0.08);
	}

	/* ─── Line flag button (always in DOM, invisible until hover) ─── */
	.flag-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 36px;
		height: 36px;
		min-width: 36px;
		border: none;
		background: none;
		color: transparent;
		cursor: pointer;
		border-radius: 6px;
		padding: 0;
		margin-left: auto;
		flex-shrink: 0;
		transition: color 0.1s;
	}

	.text-line:hover .flag-btn {
		color: var(--color-text-muted);
	}

	/* On touch devices, always subtly visible */
	@media (pointer: coarse) {
		.flag-btn {
			color: var(--color-text-muted);
			opacity: 0.3;
		}

		.text-line:active .flag-btn {
			opacity: 1;
		}
	}

	.flag-btn:hover {
		color: #e85535 !important;
		background: rgba(232, 85, 53, 0.1);
	}

	/* ─── Entry flag button (always visible, small) ─── */
	.entry-flag-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 36px;
		height: 36px;
		border: none;
		background: none;
		color: var(--color-text-muted);
		cursor: pointer;
		border-radius: 8px;
		flex-shrink: 0;
		padding: 0;
		opacity: 0.4;
		transition: opacity 0.15s;
	}

	.entry-flag-btn:hover {
		opacity: 1;
		color: #e85535;
		background: rgba(232, 85, 53, 0.1);
	}


	.text-line.stage-direction {
		font-style: italic;
		margin-top: 8px;
		margin-bottom: 8px;
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

	.text-line.stage-direction .line-content {
		color: var(--color-text-muted);
	}
</style>
