<script lang="ts">
	import type { MultiEditionScene, LexiconCitationDetail } from '$lib/server/queries';
	import IconClose from '$lib/components/icons/IconClose.svelte';
	import IconBack from '$lib/components/icons/IconBack.svelte';
	import IconButton from '$lib/components/ui/IconButton.svelte';

	let {
		data,
		citation,
		headword,
		onback,
		onclose,
		formatRef,
		citationText: getCitationText,
		citationSpeaker: getCitationSpeaker
	}: {
		data: MultiEditionScene;
		citation: LexiconCitationDetail | null;
		headword: string;
		onback: () => void;
		onclose: () => void;
		formatRef: (c: LexiconCitationDetail) => string;
		citationText: (c: LexiconCitationDetail) => string;
		citationSpeaker: (c: LexiconCitationDetail) => string | null;
	} = $props();

	const EDITION_LABELS: Record<number, string> = { 1: 'OSS', 2: 'SE', 3: 'Per', 4: 'F1', 5: 'Flg' };

	let visibleEditions = $state<number[]>([]);
	let highlightRow = $state<number | null>(null);
	let matchQuality = $state<'exact' | 'nearby' | 'scene' | 'unmatched'>('exact');

	// Initialize on mount / data change
	$effect(() => {
		if (data) {
			const matchedEd = citation?.matched_edition_id ?? 3;
			const available = data.available_editions.map(e => e.id);
			if (available.length <= 2) {
				visibleEditions = available;
			} else {
				const first = available.includes(matchedEd) ? matchedEd : available[0];
				const second = available.find(id => id !== first) ?? available[0];
				visibleEditions = [first, second];
			}
			const candidateLine = citation?.matched_line_number ?? citation?.line ?? null;
			const result = findHeadwordRow(data, candidateLine, citation?.matched_edition_id ?? null, headword);
			highlightRow = result.rowIndex;
			matchQuality = result.quality;
			scrollToHighlight();
		}
	});

	function findHeadwordRow(
		scene: MultiEditionScene,
		targetLine: number | null,
		targetEditionId: number | null,
		hw: string
	): { rowIndex: number | null; quality: 'exact' | 'nearby' | 'scene' | 'unmatched' } {
		if (!hw || scene.rows.length === 0) return { rowIndex: null, quality: 'unmatched' };

		const cleaned = hw.replace(/\d+$/, '').toLowerCase();
		const escaped = cleaned.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
		const pattern = new RegExp(`\\b${escaped}`, 'i');

		let targetIdx = -1;
		if (targetLine != null) {
			const edId = targetEditionId ?? 3;
			targetIdx = scene.rows.findIndex(r => {
				const ed = r.editions[edId];
				return ed && ed.line_number === targetLine;
			});
			if (targetIdx < 0) {
				targetIdx = scene.rows.findIndex(r =>
					Object.values(r.editions).some(ed => ed && ed.line_number === targetLine)
				);
			}
		}

		if (targetIdx >= 0) {
			const row = scene.rows[targetIdx];
			if (Object.values(row.editions).some(ed => ed && pattern.test(ed.content))) {
				return { rowIndex: targetIdx, quality: 'exact' };
			}
			for (let offset = 1; offset <= 5; offset++) {
				for (const idx of [targetIdx - offset, targetIdx + offset]) {
					if (idx >= 0 && idx < scene.rows.length) {
						if (Object.values(scene.rows[idx].editions).some(ed => ed && pattern.test(ed.content))) {
							return { rowIndex: idx, quality: 'nearby' };
						}
					}
				}
			}
		}

		const matchIdx = scene.rows.findIndex(r =>
			Object.values(r.editions).some(ed => ed && pattern.test(ed.content))
		);
		if (matchIdx >= 0) return { rowIndex: matchIdx, quality: 'scene' };

		return { rowIndex: targetIdx >= 0 ? targetIdx : null, quality: 'unmatched' };
	}

	function scrollToHighlight() {
		if (highlightRow == null) return;
		requestAnimationFrame(() => {
			const el = document.getElementById(`scene-row-${highlightRow}`);
			el?.scrollIntoView({ behavior: 'instant', block: 'center' });
		});
	}

	function toggleEdition(edId: number) {
		if (visibleEditions.includes(edId)) {
			if (visibleEditions.length > 1) {
				visibleEditions = visibleEditions.filter(id => id !== edId);
			}
		} else {
			visibleEditions = [...visibleEditions, edId];
		}
	}
</script>

<div class="modal-header">
	<IconButton onclick={onback} label="Back to entry" size={36}>
		<IconBack size={20} />
	</IconButton>
	<div class="scene-title">
		<h2>{data.work_title}</h2>
		{#if data.work_title === 'Sonnets'}
			<span class="scene-location">Sonnet {data.scene}</span>
		{:else if data.scene === 0}
			<span class="scene-location">{data.work_title}</span>
		{:else}
			<span class="scene-location">Act {data.act}, Scene {data.scene}</span>
		{/if}
	</div>
	{#if highlightRow != null}
		<button
			class="jump-btn"
			class:review={matchQuality === 'scene' || matchQuality === 'unmatched'}
			onclick={scrollToHighlight}
			aria-label="Jump to referenced line"
		>
			Row {highlightRow}
			{#if matchQuality === 'scene'}
				<span class="match-flag" title="Headword found elsewhere in scene — needs review">?</span>
			{:else if matchQuality === 'unmatched'}
				<span class="match-flag" title="Headword not found in scene — needs review">!</span>
			{/if}
		</button>
	{/if}
	<IconButton onclick={onclose} label="Close" size={36}>
		<IconClose size={20} />
	</IconButton>
</div>

{#if citation}
	<div class="scene-citation-context">
		<span class="context-ref">{formatRef(citation)}</span>
		{#if getCitationSpeaker(citation)}
			<span class="context-speaker">{getCitationSpeaker(citation)}</span>
		{/if}
		<p class="context-quote">{getCitationText(citation)}</p>
	</div>
{/if}

<div class="edition-selector">
	{#each data.available_editions as ed (ed.id)}
		<label class="edition-toggle">
			<input
				type="checkbox"
				checked={visibleEditions.includes(ed.id)}
				onchange={() => toggleEdition(ed.id)}
			/>
			<span class="edition-label">{EDITION_LABELS[ed.id] ?? ed.code}</span>
		</label>
	{/each}
</div>

<div class="modal-body scene-body">
	<div class="scene-columns" style="--col-count: {visibleEditions.length}">
		<div class="column-headers">
			{#each visibleEditions as edId (edId)}
				{@const ed = data.available_editions.find(e => e.id === edId)}
				<div class="col-header">{ed?.name ?? EDITION_LABELS[edId]}</div>
			{/each}
		</div>
		{#each data.rows as row, rowIdx (rowIdx)}
			<div
				id="scene-row-{rowIdx}"
				class="aligned-row"
				class:highlighted={rowIdx === highlightRow}
				class:needs-review={rowIdx === highlightRow && (matchQuality === 'scene' || matchQuality === 'unmatched')}
			>
				{#each visibleEditions as edId (edId)}
					{@const cell = row.editions[edId]}
					<div class="edition-cell" class:empty={!cell} class:stage-direction={cell?.content_type === 'stage_direction'}>
						{#if cell}
							<span class="line-number">{cell.line_number ?? ''}</span>
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

<style>
	.modal-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px 20px 0;
		flex-shrink: 0;
		gap: 12px;
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
		border-color: var(--color-warning);
		color: var(--color-warning);
	}

	.jump-btn:hover {
		background: var(--color-hover);
	}

	.match-flag {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 16px;
		height: 16px;
		margin-left: 4px;
		border-radius: 50%;
		background: var(--color-warning);
		color: var(--color-bg);
		font-size: 0.6rem;
		font-weight: 800;
	}

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

	.modal-body {
		flex: 1;
		overflow-y: auto;
		padding: 16px 20px 20px;
	}

	.scene-body {
		display: flex;
		flex-direction: column;
		align-items: center;
		font-size: 16px;
		line-height: 1.7;
	}

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
		border-left-color: var(--color-warning);
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

	.line-empty {
		color: var(--color-text-muted);
		font-size: 0.7rem;
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
