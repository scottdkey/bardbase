<script lang="ts">
	import { onMount } from 'svelte';
	import type { MultiEditionScene } from '$lib/types';
	import IconBack from '$lib/components/icons/IconBack.svelte';
	import IconButton from '$lib/components/ui/IconButton.svelte';
	import { goto } from '$app/navigation';
	import { readingPosition } from '$lib/stores/reading-position.svelte';
	import { findAdjacentScenes } from '$lib/utils/scene-nav';

	let { data } = $props();
	let scene: MultiEditionScene = $derived(data.scene);
	let adjacent = $derived(findAdjacentScenes(data.toc, data.act, data.sceneNum));

	const EDITION_LABELS: Record<number, string> = {
		1: 'OSS',
		2: 'SE',
		3: 'Per',
		4: 'F1',
		5: 'Flg'
	};

	let visibleEditions = $state<number[]>([]);
	let highlightRow = $state<number | null>(null);
	let matchQuality = $state<'exact' | 'nearby' | 'scene' | 'unmatched'>('exact');
	let tocOpen = $state(false);
	let headerEl = $state<HTMLElement | null>(null);
	let headerHeight = $state(100);

	$effect(() => {
		if (headerEl) {
			headerHeight = headerEl.offsetHeight;
		}
	});

	// Touch swipe tracking
	let touchStartX = 0;
	let touchStartY = 0;

	$effect(() => {
		if (scene) {
			const matchedEd = data.editionId ?? 3;
			const available = scene.available_editions.map((e) => e.id);
			if (available.length <= 2) {
				visibleEditions = available;
			} else {
				const first = available.includes(matchedEd) ? matchedEd : available[0];
				const second = available.find((id) => id !== first) ?? available[0];
				visibleEditions = [first, second];
			}

			if (data.isReference) {
				const candidateLine = data.line;
				const result = findHeadwordRow(scene, candidateLine, data.editionId, data.headword);
				highlightRow = result.rowIndex;
				matchQuality = result.quality;
				scrollToHighlight();
			} else {
				highlightRow = null;
			}
		}
	});

	// Reading position: save on scene load and restore scroll
	onMount(() => {
		if (!data.isReference) {
			const saved = readingPosition.get(data.workId);
			if (saved && saved.act === data.act && saved.scene === data.sceneNum && saved.scrollY > 0) {
				requestAnimationFrame(() => window.scrollTo(0, saved.scrollY));
			}
			readingPosition.save(data.workId, data.act, data.sceneNum, 0);

			let scrollTimer: ReturnType<typeof setTimeout>;
			function onScroll() {
				clearTimeout(scrollTimer);
				scrollTimer = setTimeout(() => {
					readingPosition.save(data.workId, data.act, data.sceneNum, window.scrollY);
				}, 500);
			}
			window.addEventListener('scroll', onScroll, { passive: true });
			return () => {
				clearTimeout(scrollTimer);
				window.removeEventListener('scroll', onScroll);
			};
		}
	});

	function findHeadwordRow(
		s: MultiEditionScene,
		targetLine: number | null,
		targetEditionId: number | null,
		hw: string
	): { rowIndex: number | null; quality: 'exact' | 'nearby' | 'scene' | 'unmatched' } {
		if (!hw || s.rows.length === 0) return { rowIndex: null, quality: 'unmatched' };
		const cleaned = hw.replace(/\d+$/, '').toLowerCase();
		const escaped = cleaned.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
		const pattern = new RegExp(`\\b${escaped}`, 'i');

		let targetIdx = -1;
		if (targetLine != null) {
			const edId = targetEditionId ?? 3;
			targetIdx = s.rows.findIndex((r) => {
				const ed = r.editions[edId];
				return ed && ed.line_number === targetLine;
			});
			if (targetIdx < 0) {
				targetIdx = s.rows.findIndex((r) =>
					Object.values(r.editions).some((ed) => ed && ed.line_number === targetLine)
				);
			}
		}
		if (targetIdx >= 0) {
			const row = s.rows[targetIdx];
			if (Object.values(row.editions).some((ed) => ed && pattern.test(ed.content))) {
				return { rowIndex: targetIdx, quality: 'exact' };
			}
			for (let offset = 1; offset <= 5; offset++) {
				for (const idx of [targetIdx - offset, targetIdx + offset]) {
					if (idx >= 0 && idx < s.rows.length) {
						if (Object.values(s.rows[idx].editions).some((ed) => ed && pattern.test(ed.content))) {
							return { rowIndex: idx, quality: 'nearby' };
						}
					}
				}
			}
		}
		const matchIdx = s.rows.findIndex((r) =>
			Object.values(r.editions).some((ed) => ed && pattern.test(ed.content))
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
				visibleEditions = visibleEditions.filter((id) => id !== edId);
			}
		} else {
			visibleEditions = [...visibleEditions, edId];
		}
	}

	function getRowCharacter(row: (typeof scene.rows)[number]): string | null {
		for (const edId of visibleEditions) {
			const cell = row.editions[edId];
			if (cell?.character_name && cell.character_name !== '(stage directions)') {
				return cell.character_name;
			}
		}
		return null;
	}

	function sceneTitle(): string {
		if (scene.work_title === 'Sonnets') return `Sonnet ${scene.scene}`;
		if (scene.scene === 0) return scene.work_title;
		return `Act ${scene.act}, Scene ${scene.scene}`;
	}

	function goBack() {
		if (data.isReference) {
			history.back();
		} else {
			goto('/editions');
		}
	}

	function gotoScene(act: number, sc: number) {
		tocOpen = false;
		goto(`/text/${data.workId}/${act}/${sc}`);
	}

	function gotoPrev() {
		if (adjacent.prev) gotoScene(adjacent.prev.act, adjacent.prev.scene);
	}
	function gotoNext() {
		if (adjacent.next) gotoScene(adjacent.next.act, adjacent.next.scene);
	}

	function handleTouchStart(e: TouchEvent) {
		if (data.isReference) return;
		touchStartX = e.touches[0].clientX;
		touchStartY = e.touches[0].clientY;
	}

	function handleTouchEnd(e: TouchEvent) {
		if (data.isReference) return;
		const dx = e.changedTouches[0].clientX - touchStartX;
		const dy = e.changedTouches[0].clientY - touchStartY;
		if (Math.abs(dx) > 80 && Math.abs(dx) > Math.abs(dy) * 1.5) {
			if (dx > 0) gotoPrev();
			else gotoNext();
		}
	}

	// Group TOC by act for the panel
	let tocByAct = $derived(() => {
		const acts = new Map<number, typeof data.toc>();
		for (const d of data.toc) {
			const list = acts.get(d.act) ?? [];
			list.push(d);
			acts.set(d.act, list);
		}
		return acts;
	});
</script>

<svelte:head>
	<title>{scene.work_title} {sceneTitle()} &mdash; Bardbase</title>
</svelte:head>

<div
	class="scene-page"
	ontouchstart={handleTouchStart}
	ontouchend={handleTouchEnd}
>
	<div class="scene-header" bind:this={headerEl}>
		<IconButton onclick={goBack} label="Back" size={36}>
			<IconBack size={20} />
		</IconButton>
		{#if !data.isReference}
			<button class="toc-btn" onclick={() => (tocOpen = !tocOpen)} aria-label="Table of contents">
				<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<line x1="3" y1="6" x2="21" y2="6" /><line x1="3" y1="12" x2="21" y2="12" /><line x1="3" y1="18" x2="21" y2="18" />
				</svg>
			</button>
		{/if}
		<div class="scene-title">
			<h1>{scene.work_title}</h1>
			<span class="scene-location">{sceneTitle()}</span>
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
					<span class="match-flag" title="Headword found elsewhere in scene">?</span>
				{:else if matchQuality === 'unmatched'}
					<span class="match-flag" title="Headword not found in scene">!</span>
				{/if}
			</button>
		{/if}
	</div>

	<div class="edition-selector">
		{#each scene.available_editions as ed (ed.id)}
			<button
				class="edition-pill"
				class:active={visibleEditions.includes(ed.id)}
				onclick={() => toggleEdition(ed.id)}
				aria-pressed={visibleEditions.includes(ed.id)}
			>
				{EDITION_LABELS[ed.id] ?? ed.code}
			</button>
		{/each}
	</div>

	<div class="scene-body">
		<div class="scene-columns" style="--col-count: {visibleEditions.length}">
			<div class="column-headers" style="top: {headerHeight}px">
				{#each visibleEditions as edId (edId)}
					{@const ed = scene.available_editions.find((e) => e.id === edId)}
					<div class="col-header">{ed?.name ?? EDITION_LABELS[edId]}</div>
				{/each}
			</div>
			{#each scene.rows as row, rowIdx (rowIdx)}
				{@const char = getRowCharacter(row)}
				{@const prevChar = rowIdx > 0 ? getRowCharacter(scene.rows[rowIdx - 1]) : null}
				{#if char && char !== prevChar}
					<div class="speaker-row">
						<span class="speaker-name">{char}</span>
					</div>
				{/if}
				<div
					id="scene-row-{rowIdx}"
					class="aligned-row"
					class:highlighted={rowIdx === highlightRow}
					class:needs-review={rowIdx === highlightRow &&
						(matchQuality === 'scene' || matchQuality === 'unmatched')}
				>
					{#each visibleEditions as edId (edId)}
						{@const cell = row.editions[edId]}
						<div
							class="edition-cell"
							class:empty={!cell}
							class:stage-direction={cell?.content_type === 'stage_direction'}
						>
							{#if cell}
								<span class="line-number">{cell.line_number ?? ''}</span>
								<span class="line-content">{cell.content}</span>
							{:else}
								<span class="line-empty">&mdash;</span>
							{/if}
						</div>
					{/each}
				</div>
			{/each}
		</div>
	</div>
</div>

<!-- Floating nav arrows (reading mode only) -->
{#if !data.isReference}
	{#if adjacent.prev}
		<button class="nav-arrow nav-prev" onclick={gotoPrev} aria-label="Previous scene">
			&#8249;
		</button>
	{/if}
	{#if adjacent.next}
		<button class="nav-arrow nav-next" onclick={gotoNext} aria-label="Next scene">
			&#8250;
		</button>
	{/if}
{/if}

<!-- TOC panel -->
{#if tocOpen}
	<div class="toc-backdrop" onclick={() => (tocOpen = false)} role="presentation"></div>
	<nav class="toc-panel" aria-label="Table of contents">
		<h2 class="toc-title">{scene.work_title}</h2>
		{#each [...tocByAct().entries()] as [actNum, scenes] (actNum)}
			<div class="toc-act">
				{#if actNum > 0}
					<h3 class="toc-act-title">Act {actNum}</h3>
				{/if}
				<ul class="toc-scene-list">
					{#each scenes as sc (sc.scene)}
						<li>
							<button
								class="toc-scene-btn"
								class:current={sc.act === data.act && sc.scene === data.sceneNum}
								onclick={() => gotoScene(sc.act, sc.scene)}
							>
								{#if actNum === 0}
									{sc.description ?? `${sc.scene}`}
								{:else}
									Scene {sc.scene}
								{/if}
								<span class="toc-line-count">{sc.line_count} lines</span>
							</button>
						</li>
					{/each}
				</ul>
			</div>
		{/each}
	</nav>
{/if}

<style>
	.scene-page {
		margin: 0 auto;
		padding: 0 56px 48px;
	}

	.scene-header {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 16px 0 8px;
		position: sticky;
		top: 0;
		z-index: 20;
		background: var(--color-bg);
	}

	.toc-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		border: none;
		background: none;
		color: var(--color-text-muted);
		cursor: pointer;
		padding: 6px;
		border-radius: 6px;
		flex-shrink: 0;
	}

	.toc-btn:hover {
		color: var(--color-text);
		background: var(--color-hover);
	}

	.scene-title {
		flex: 1;
		min-width: 0;
	}

	.scene-title h1 {
		margin: 0;
		font-size: 1.2rem;
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

	.edition-selector {
		position: fixed;
		bottom: 52px;
		left: 50%;
		transform: translateX(-50%);
		z-index: 250;
		display: flex;
		gap: 4px;
		padding: 6px 8px;
		background: var(--color-elevated);
		border: 1px solid var(--color-border);
		border-radius: 24px;
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.3);
	}

	.edition-pill {
		padding: 4px 10px;
		border: none;
		background: none;
		color: var(--color-text-muted);
		font-family: inherit;
		font-size: 0.65rem;
		font-weight: 700;
		cursor: pointer;
		border-radius: 14px;
		transition: background 0.15s, color 0.15s;
		user-select: none;
		letter-spacing: 0.02em;
	}

	.edition-pill:hover {
		color: var(--color-text);
		background: var(--color-hover);
	}

	.edition-pill.active {
		background: var(--color-accent);
		color: var(--color-bg);
	}

	.scene-body {
		font-size: 16px;
		line-height: 1.7;
		padding-top: 36px;
	}

	.scene-columns {
		width: 100%;
		max-width: calc(var(--col-count) * 600px);
		margin: 0 auto;
	}

	.column-headers {
		display: grid;
		grid-template-columns: repeat(var(--col-count), 1fr);
		gap: 1px;
		position: fixed;
		left: 56px;
		right: 56px;
		z-index: 15;
		background: var(--color-surface);
		border-bottom: 1px solid var(--color-border);
		max-width: calc(var(--col-count) * 600px);
		margin: 0 auto;
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

	.speaker-row {
		padding: 10px 6px 2px;
		text-align: center;
	}

	.speaker-name {
		font-size: 0.7rem;
		font-weight: 700;
		color: var(--color-accent);
		text-transform: uppercase;
		letter-spacing: 0.04em;
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

	/* ─── Floating nav arrows ─── */
	.nav-arrow {
		position: fixed;
		top: 50%;
		transform: translateY(-50%);
		z-index: 100;
		width: 40px;
		height: 64px;
		border: none;
		background: var(--color-surface);
		color: var(--color-text-muted);
		font-size: 1.8rem;
		cursor: pointer;
		border-radius: 8px;
		opacity: 0.5;
		transition: opacity 0.15s, background 0.15s;
		display: flex;
		align-items: center;
		justify-content: center;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
	}

	.nav-arrow:hover {
		opacity: 1;
		background: var(--color-elevated);
		color: var(--color-text);
	}

	.nav-prev {
		left: 8px;
	}

	.nav-next {
		right: 8px;
	}

	/* ─── TOC panel ─── */
	.toc-backdrop {
		position: fixed;
		inset: 0;
		background: var(--color-overlay);
		z-index: 300;
	}

	.toc-panel {
		position: fixed;
		top: 0;
		left: 0;
		bottom: 0;
		width: 280px;
		max-width: 85vw;
		background: var(--color-elevated);
		border-right: 1px solid var(--color-border);
		z-index: 400;
		overflow-y: auto;
		padding: 20px 16px;
		animation: toc-slide-in 0.2s ease-out;
	}

	@keyframes toc-slide-in {
		from {
			transform: translateX(-100%);
		}
		to {
			transform: translateX(0);
		}
	}

	.toc-title {
		margin: 0 0 16px;
		font-size: 1rem;
		font-weight: 700;
		color: var(--color-text);
	}

	.toc-act {
		margin-bottom: 12px;
	}

	.toc-act-title {
		margin: 0 0 4px;
		font-size: 0.75rem;
		font-weight: 700;
		color: var(--color-accent);
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.toc-scene-list {
		list-style: none;
		padding: 0;
		margin: 0;
	}

	.toc-scene-btn {
		display: flex;
		justify-content: space-between;
		align-items: center;
		width: 100%;
		padding: 8px 12px;
		border: none;
		background: none;
		color: var(--color-text);
		font-family: inherit;
		font-size: 0.85rem;
		cursor: pointer;
		border-radius: 6px;
		text-align: left;
		transition: background 0.15s;
	}

	.toc-scene-btn:hover {
		background: var(--color-hover);
	}

	.toc-scene-btn.current {
		background: var(--color-active);
		font-weight: 700;
		color: var(--color-accent);
	}

	.toc-line-count {
		font-size: 0.65rem;
		color: var(--color-text-muted);
		flex-shrink: 0;
		margin-left: 8px;
	}

	@media (max-width: 600px) {
		.scene-page {
			padding: 0 8px 48px;
		}

		.column-headers {
			left: 8px;
			right: 8px;
		}

		.scene-columns {
			overflow-x: auto;
		}

		.aligned-row {
			min-width: calc(var(--col-count) * 180px);
		}

		.nav-arrow {
			width: 32px;
			height: 48px;
			font-size: 1.4rem;
		}

		.nav-prev {
			left: 4px;
		}

		.nav-next {
			right: 4px;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.toc-panel {
			animation: none;
		}
	}
</style>
