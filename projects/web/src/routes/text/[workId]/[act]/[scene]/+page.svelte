<script lang="ts">
	import { onMount } from 'svelte';
	import type { MultiEditionScene } from '$lib/types';
	import type { LineReference } from '$lib/server/api';
	import IconBack from '$lib/components/icons/IconBack.svelte';
	import IconButton from '$lib/components/ui/IconButton.svelte';
	import WordPopover from '$lib/components/WordPopover.svelte';
	import ReferenceDrawer from '$lib/components/ReferenceDrawer.svelte';
	import { goto } from '$app/navigation';
	import { readingPosition } from '$lib/stores/reading-position.svelte';
	import { findAdjacentScenes } from '$lib/utils/scene-nav';

	let { data } = $props();
	let scene: MultiEditionScene = $derived(data.scene);
	let adjacent = $derived(findAdjacentScenes(data.toc, data.act, data.sceneNum));

	// References: pre-loaded from DB, keyed by line number
	let refs: Record<string, LineReference[]> = $derived(data.references ?? {});

	// Build a row-level reference map: for each aligned row, collect ALL references
	// from ALL edition line numbers in that row. This ensures that a citation
	// stored under one edition's line number highlights the word in every edition.
	let rowRefs = $derived.by(() => {
		const map = new Map<number, LineReference[]>();
		for (let rowIdx = 0; rowIdx < scene.rows.length; rowIdx++) {
			const row = scene.rows[rowIdx];
			const collected: LineReference[] = [];
			const seen = new Set<string>();
			for (const cell of Object.values(row.editions)) {
				if (!cell || cell.line_number == null) continue;
				const lineRefs = refs[String(cell.line_number)];
				if (!lineRefs) continue;
				for (const ref of lineRefs) {
					const key = `${ref.entry_id}-${ref.source_code}`;
					if (!seen.has(key)) {
						seen.add(key);
						collected.push(ref);
					}
				}
			}
			if (collected.length > 0) map.set(rowIdx, collected);
		}
		return map;
	});

	function normalizeWord(raw: string): string {
		return raw.replace(/^[^a-zA-Z'-]+|[^a-zA-Z'-]+$/g, '').toLowerCase();
	}

	function refKeyMatches(refKey: string, word: string): boolean {
		const key = refKey.toLowerCase().replace(/\s+/g, ' ');
		if (key === word) return true;
		// Key starts with word
		if (key.startsWith(word + ' ') || key.startsWith(word + ',')) return true;
		// Word appears as a standalone word anywhere in the key (for phrase headwords like Bartlett)
		if (key.includes(' ' + word + ' ') || key.includes(' ' + word + ',') || key.endsWith(' ' + word)) return true;
		return false;
	}

	// Check if a word in a specific row has a reference (filtered by enabled sources)
	function hasWordRef(rowIdx: number, word: string): boolean {
		const rowRefList = rowRefs.get(rowIdx);
		if (!rowRefList || enabledSources.size === 0) return false;
		const clean = normalizeWord(word);
		return rowRefList.some(
			(r) => enabledSources.has(r.source_code) && refKeyMatches(r.entry_key, clean)
		);
	}

	// Get references for a word in a specific row (filtered by enabled sources)
	function getWordRefs(rowIdx: number, word: string): LineReference[] {
		const rowRefList = rowRefs.get(rowIdx);
		if (!rowRefList) return [];
		const clean = normalizeWord(word);
		return rowRefList.filter(
			(r) => enabledSources.has(r.source_code) && refKeyMatches(r.entry_key, clean)
		);
	}

	// Word popover state
	let popoverWord = $state<string | null>(null);
	let popoverRefs = $state<LineReference[]>([]);
	let popoverX = $state(0);
	let popoverY = $state(0);
	let hoverTimer: ReturnType<typeof setTimeout>;

	function showWordPopover(word: string, rowIdx: number, x: number, y: number) {
		const wordRefs = getWordRefs(rowIdx, word);
		if (wordRefs.length === 0) return;
		popoverWord = word.replace(/^[^a-zA-Z'-]+|[^a-zA-Z'-]+$/g, '');
		popoverRefs = wordRefs;
		popoverX = x;
		popoverY = y;
	}

	function handleWordInteraction(e: MouseEvent) {
		const target = e.target as HTMLElement;
		if (!target.classList.contains('ref')) return;
		clearTimeout(hoverTimer);
		const rowIdx = parseInt(target.dataset.row ?? '', 10);
		if (isNaN(rowIdx)) return;
		const rect = target.getBoundingClientRect();
		if (e.type === 'click') {
			showWordPopover(target.textContent ?? '', rowIdx, rect.left, rect.bottom);
		} else {
			hoverTimer = setTimeout(() => {
				showWordPopover(target.textContent ?? '', rowIdx, rect.left, rect.bottom);
			}, 300);
		}
	}

	function handleWordLeave() {
		clearTimeout(hoverTimer);
	}

	function closePopover() {
		popoverWord = null;
	}

	// Reference drawer state
	let drawerRef = $state<LineReference | null>(null);

	function selectEntry(ref: LineReference) {
		popoverWord = null;
		drawerRef = ref;
	}

	function closeDrawer() {
		drawerRef = null;
	}

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
	let editionsDropdownOpen = $state(false);
	let refsDropdownOpen = $state(false);
	let headerEl = $state<HTMLElement | null>(null);
	let headerHeight = $state(100);
	let headerVisible = $state(true);
	let lastScrollY = 0;

	const EDITION_PREF_KEY = 'bardbase-preferred-editions';
	const REF_PREF_KEY = 'bardbase-preferred-refs';

	const SOURCE_LABELS: Record<string, string> = {
		schmidt: 'Schmidt',
		onions: 'Onions',
		abbott: 'Abbott',
		bartlett: 'Bartlett',
		henley_farmer: 'H&F'
	};

	// Reference source toggles
	let enabledSources = $state<Set<string>>(new Set());

	function loadPreferredEditions(): number[] | null {
		try {
			const raw = localStorage.getItem(EDITION_PREF_KEY);
			return raw ? JSON.parse(raw) : null;
		} catch {
			return null;
		}
	}

	function savePreferredEditions(editions: number[]) {
		try {
			localStorage.setItem(EDITION_PREF_KEY, JSON.stringify(editions));
		} catch {}
	}

	function loadPreferredRefs(): Set<string> | null {
		try {
			const raw = localStorage.getItem(REF_PREF_KEY);
			return raw ? new Set(JSON.parse(raw)) : null;
		} catch {
			return null;
		}
	}

	function savePreferredRefs(sources: Set<string>) {
		try {
			localStorage.setItem(REF_PREF_KEY, JSON.stringify([...sources]));
		} catch {}
	}

	function toggleSource(code: string) {
		if (enabledSources.has(code)) {
			enabledSources.delete(code);
		} else {
			enabledSources.add(code);
		}
		enabledSources = new Set(enabledSources);
		savePreferredRefs(enabledSources);
	}

	// Available reference sources in this scene
	let availableSources = $derived.by(() => {
		const sources = new Set<string>();
		for (const lineRefs of Object.values(refs)) {
			for (const ref of lineRefs) {
				sources.add(ref.source_code);
			}
		}
		return [...sources].sort();
	});

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
			const available = scene.available_editions.map((e) => e.id);
			const preferred = loadPreferredEditions();

			if (preferred) {
				// Use saved preference, filtered to what's available in this scene
				const filtered = preferred.filter((id) => available.includes(id));
				visibleEditions = filtered.length > 0 ? filtered : [available[0]];
			} else if (data.isReference && data.editionId) {
				// Reference mode: show the referenced edition + one other
				const matchedEd = data.editionId;
				const first = available.includes(matchedEd) ? matchedEd : available[0];
				const second = available.find((id) => id !== first) ?? available[0];
				visibleEditions = [first, second];
			} else {
				// First visit, no preference: default to first two
				if (available.length <= 2) {
					visibleEditions = available;
				} else {
					visibleEditions = [available[0], available[1]];
				}
			}

			if (data.isReference) {
				const candidateLine = data.line;
				if (data.headword) {
					const result = findHeadwordRow(scene, candidateLine, data.editionId, data.headword);
					highlightRow = result.rowIndex;
					matchQuality = result.quality;
				} else if (candidateLine != null) {
					// Line-only reference (no headword): find the row by line number
					const idx = scene.rows.findIndex((r) =>
						Object.values(r.editions).some((ed) => ed && ed.line_number === candidateLine)
					);
					highlightRow = idx >= 0 ? idx : null;
					matchQuality = idx >= 0 ? 'exact' : 'unmatched';
				}
				scrollToHighlight();
			} else {
				highlightRow = null;
			}
		}
	});

	// Initialize reference sources from localStorage
	$effect(() => {
		if (availableSources.length > 0) {
			const saved = loadPreferredRefs();
			if (saved) {
				enabledSources = new Set([...saved].filter((s) => availableSources.includes(s)));
			} else {
				enabledSources = new Set(availableSources);
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
		}

		let scrollTimer: ReturnType<typeof setTimeout>;
		function onScroll() {
			const y = window.scrollY;
			// Auto-hide header on scroll down, show on scroll up
			if (y > lastScrollY && y > 100) {
				headerVisible = false;
				editionsDropdownOpen = false;
				refsDropdownOpen = false;
			} else {
				headerVisible = true;
			}
			lastScrollY = y;

			// Save reading position (debounced)
			if (!data.isReference) {
				clearTimeout(scrollTimer);
				scrollTimer = setTimeout(() => {
					readingPosition.save(data.workId, data.act, data.sceneNum, window.scrollY);
				}, 500);
			}
		}
		window.addEventListener('scroll', onScroll, { passive: true });
		return () => {
			clearTimeout(scrollTimer);
			window.removeEventListener('scroll', onScroll);
		};
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
		savePreferredEditions(visibleEditions);
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

	// Build character name → description map for tooltips
	let charDescriptions = $derived.by(() => {
		const map = new Map<string, string>();
		if (!scene.characters) return map;
		for (const c of scene.characters) {
			if (c.description) {
				map.set(c.name, c.description);
			}
		}
		return map;
	});

	function sceneTitle(): string {
		if (scene.work_title === 'Sonnets') return `Sonnet ${scene.scene}`;
		if (scene.scene === 0) return scene.work_title;
		return `Act ${scene.act}, Scene ${scene.scene}`;
	}

	function goBack() {
		history.back();
	}

	function gotoScene(act: number, sc: number) {
		tocOpen = false;
		goto(`/text/${data.slug}/${act}/${sc}`);
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

	function handleWindowClick(e: MouseEvent) {
		const target = e.target as HTMLElement;
		if (!target.closest('.dropdown')) {
			editionsDropdownOpen = false;
			refsDropdownOpen = false;
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

<!-- svelte-ignore a11y_no_static_element_interactions -->
<svelte:window onclick={handleWindowClick} />

<svelte:head>
	<title>{scene.work_title} {sceneTitle()} &mdash; Bardbase</title>
</svelte:head>

<div
	class="scene-page"
	ontouchstart={handleTouchStart}
	ontouchend={handleTouchEnd}
>
	<div class="scene-header" class:hidden={!headerVisible} bind:this={headerEl}>
		<div class="header-row">
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
			<div class="header-dropdowns">
				<div class="dropdown">
					<button class="dropdown-trigger" onclick={() => { editionsDropdownOpen = !editionsDropdownOpen; refsDropdownOpen = false; }}>
						Editions <span class="dropdown-count">{visibleEditions.length}</span>
					</button>
					{#if editionsDropdownOpen}
						<div class="dropdown-panel">
							{#each scene.available_editions as ed (ed.id)}
								<label class="dropdown-item">
									<input
										type="checkbox"
										checked={visibleEditions.includes(ed.id)}
										onchange={() => toggleEdition(ed.id)}
									/>
									<span>{ed.name}</span>
								</label>
							{/each}
						</div>
					{/if}
				</div>
				{#if availableSources.length > 0}
					<div class="dropdown">
						<button class="dropdown-trigger" onclick={() => { refsDropdownOpen = !refsDropdownOpen; editionsDropdownOpen = false; }}>
							References <span class="dropdown-count">{enabledSources.size}</span>
						</button>
						{#if refsDropdownOpen}
							<div class="dropdown-panel">
								{#each availableSources as src (src)}
									{@const FULL_LABELS: Record<string, string> = {
										schmidt: 'Schmidt Lexicon',
										onions: 'Onions Glossary',
										abbott: 'Abbott Grammar',
										bartlett: 'Bartlett Concordance',
										henley_farmer: 'Henley & Farmer'
									}}
									<label class="dropdown-item">
										<input
											type="checkbox"
											checked={enabledSources.has(src)}
											onchange={() => toggleSource(src)}
										/>
										<span>{FULL_LABELS[src] ?? src}</span>
									</label>
								{/each}
							</div>
						{/if}
					</div>
				{/if}
			</div>
		</div>
	</div>

	<div class="scene-body">
		<div class="scene-columns" style="--col-count: {visibleEditions.length}">
			<div class="column-headers" style="top: {headerVisible ? headerHeight : 0}px">
				{#each visibleEditions as edId (edId)}
					{@const ed = scene.available_editions.find((e) => e.id === edId)}
					<div class="col-header">{ed?.name ?? EDITION_LABELS[edId]}</div>
				{/each}
			</div>
			{#each scene.rows as row, rowIdx (rowIdx)}
				{@const char = getRowCharacter(row)}
				{@const prevChar = rowIdx > 0 ? getRowCharacter(scene.rows[rowIdx - 1]) : null}
				{#if char && char !== prevChar}
					{@const desc = charDescriptions.get(char)}
					<div class="speaker-row">
						{#if desc}
							<span class="speaker-name has-desc" tabindex="0">{char}<span class="speaker-desc">{desc}</span></span>
						{:else}
							<span class="speaker-name">{char}</span>
						{/if}
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
								<!-- svelte-ignore a11y_no_static_element_interactions -->
								<span
									class="line-content"
									onmouseenter={handleWordInteraction}
									onmouseleave={handleWordLeave}
									onclick={handleWordInteraction}
								>{#if rowRefs.has(rowIdx)}{#each cell.content.split(/(\s+)/) as part}{#if /\s/.test(part)}{part}{:else if hasWordRef(rowIdx, part)}<span class="word ref" data-row={rowIdx}>{part}</span>{:else}{part}{/if}{/each}{:else}{cell.content}{/if}</span>
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

{#if popoverWord}
	<WordPopover
		word={popoverWord}
		references={popoverRefs}
		x={popoverX}
		y={popoverY}
		onclose={closePopover}
		onselect={selectEntry}
	/>
{/if}

{#if drawerRef}
	<ReferenceDrawer ref={drawerRef} onclose={closeDrawer} />
{/if}

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
								<!-- line count removed -->
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
		position: sticky;
		top: 0;
		z-index: 20;
		background: var(--color-bg);
		padding: 8px 0 4px;
		transition: transform 0.25s ease;
	}

	.scene-header.hidden {
		transform: translateY(-100%);
	}

	.header-row {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.header-dropdowns {
		display: flex;
		flex-direction: column;
		gap: 2px;
		margin-left: auto;
		flex-shrink: 0;
		align-items: flex-end;
	}

	.dropdown {
		position: relative;
	}

	.dropdown-trigger {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 3px 8px;
		border: 1px solid var(--color-border);
		background: none;
		color: var(--color-text-muted);
		font-family: inherit;
		font-size: 0.6rem;
		font-weight: 600;
		cursor: pointer;
		border-radius: 6px;
		white-space: nowrap;
		transition: border-color 0.15s;
	}

	.dropdown-trigger:hover {
		border-color: var(--color-accent);
		color: var(--color-text);
	}

	.dropdown-count {
		background: var(--color-accent);
		color: var(--color-bg);
		font-size: 0.55rem;
		font-weight: 700;
		padding: 0 4px;
		border-radius: 8px;
		min-width: 14px;
		text-align: center;
	}

	.dropdown-panel {
		position: absolute;
		top: 100%;
		right: 0;
		margin-top: 4px;
		min-width: 200px;
		background: var(--color-elevated);
		border: 1px solid var(--color-border);
		border-radius: 8px;
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.25);
		padding: 6px 0;
		z-index: 30;
	}

	.dropdown-item {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 6px 12px;
		font-size: 0.75rem;
		color: var(--color-text);
		cursor: pointer;
		transition: background 0.1s;
	}

	.dropdown-item:hover {
		background: var(--color-hover);
	}

	.dropdown-item input {
		margin: 0;
		width: 14px;
		height: 14px;
		cursor: pointer;
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
		font-size: 1.1rem;
		font-weight: 700;
		color: var(--color-text);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.scene-location {
		font-size: 0.75rem;
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
		transition: top 0.25s ease;
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

	.speaker-name.has-desc {
		cursor: pointer;
		position: relative;
	}

	.speaker-desc {
		display: none;
		position: absolute;
		left: 0;
		top: 100%;
		font-size: 0.65rem;
		color: var(--color-text-secondary);
		font-style: italic;
		text-transform: none;
		letter-spacing: normal;
		font-weight: 400;
		padding: 4px 8px;
		background: var(--color-elevated);
		border: 1px solid var(--color-border);
		border-radius: 4px;
		z-index: 10;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
		width: max-content;
		max-width: min(500px, 90vw);
	}

	.speaker-name.has-desc:hover .speaker-desc,
	.speaker-name.has-desc:focus .speaker-desc {
		display: block;
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

	.line-content .word.ref {
		cursor: pointer;
		border-radius: 3px;
		color: var(--color-accent);
		font-weight: 700;
		background: var(--color-hover);
		padding: 1px 2px;
		transition: background 0.15s;
	}

	.line-content .word.ref:hover {
		background: var(--color-active);
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
