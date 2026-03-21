<script lang="ts">
	import type { LexiconEntryDetail, LexiconCitationDetail, SceneTextResult } from '$lib/server/queries';

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
	let sceneLoading = $state(false);

	// Group citations by sense_id
	let citationsBySense = $derived.by(() => {
		if (!entry) return { bySense: new Map<number, LexiconCitationDetail[]>(), unassigned: [] as LexiconCitationDetail[] };
		const bySense = new Map<number, LexiconCitationDetail[]>();
		const unassigned: LexiconCitationDetail[] = [];
		for (const c of entry.citations) {
			if (c.sense_id != null) {
				const list = bySense.get(c.sense_id) ?? [];
				list.push(c);
				bySense.set(c.sense_id, list);
			} else {
				unassigned.push(c);
			}
		}
		return { bySense, unassigned };
	});

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

	async function openScene(c: LexiconCitationDetail) {
		if (!c.work_id || c.act == null || c.scene == null) return;
		sceneLoading = true;
		try {
			const res = await fetch(`/api/text/scene?workId=${c.work_id}&act=${c.act}&scene=${c.scene}`);
			if (res.ok) {
				sceneData = await res.json();
				sceneHighlightLine = c.line;
				scrollToHighlight();
			}
		} finally {
			sceneLoading = false;
		}
	}

	function closeScene() {
		sceneData = null;
		sceneHighlightLine = null;
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
	<ul class="citation-list">
		{#each citations as citation (citation.id)}
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
				>
					<span class="citation-ref">{formatRef(citation)}</span>
					{#if citationSpeaker(citation)}
						<span class="citation-speaker">{citationSpeaker(citation)}</span>
					{/if}
					<p class="citation-quote">{citationText(citation)}</p>
				</button>
			</li>
		{/each}
	</ul>
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
					<button class="jump-btn" onclick={scrollToHighlight} aria-label="Jump to referenced line">
						Line {sceneHighlightLine}
					</button>
				{/if}
				<button class="close-btn" onclick={onclose} aria-label="Close">
					<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<line x1="18" y1="6" x2="6" y2="18" />
						<line x1="6" y1="6" x2="18" y2="18" />
					</svg>
				</button>
			</div>
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
							class:stage-direction={line.content_type === 'stage_direction'}
						>
							<span class="line-number">{line.line_number ?? ''}</span>
							<span class="line-content">{line.content}</span>
						</div>
					{/each}
				</div>
			</div>
		{:else}
			<!-- Entry detail view -->
			<div class="modal-header">
				<h2 class="entry-word">{entry.key}</h2>
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
				{#if entry.senses.length > 0}
					<section class="senses" aria-label="Definitions">
						{#each entry.senses as sense}
							<div class="sense-block" class:sub-sense={sense.sub_sense}>
								<div class="sense">
									{#if sense.sub_sense}
										<span class="sense-num sub">{sense.sub_sense})</span>
									{:else}
										<span class="sense-num">{sense.sense_number})</span>
									{/if}
									<p class="sense-def">{sense.definition_text}</p>
								</div>
								{#if citationsBySense.bySense.has(sense.id)}
									{@const senseCitations = citationsBySense.bySense.get(sense.id)!}
									<details class="sense-citations">
										<summary class="refs-toggle">References ({senseCitations.length})</summary>
										{@render citationList(senseCitations)}
									</details>
								{/if}
							</div>
						{/each}
					</section>
				{:else if entry.full_text}
					<section class="full-text" aria-label="Definition">
						<p>{entry.full_text}</p>
					</section>
				{/if}

				{#if citationsBySense.unassigned.length > 0}
					<details class="citations-section">
						<summary class="refs-toggle">References ({citationsBySense.unassigned.length})</summary>
						{@render citationList(citationsBySense.unassigned)}
					</details>
				{/if}
			</div>
		{/if}
	</div>
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

	.jump-btn:hover {
		background: var(--color-hover);
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
