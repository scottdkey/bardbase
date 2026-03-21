<script lang="ts">
	import { dev } from '$app/environment';
	import { corrections, CORRECTION_CATEGORIES, type CorrectionType } from '$lib/stores/corrections.svelte';

	let {
		type = 'line' as CorrectionType,
		entryKey,
		// Line fields
		workTitle,
		act,
		scene,
		lineNumber,
		currentText,
		characterName,
		editionName,
		// Citation fields
		citationRef,
		// Entry fields
		senseNumber,
		subSense,
		// Close handler
		onclose
	}: {
		type?: CorrectionType;
		entryKey: string;
		workTitle?: string;
		act?: number;
		scene?: number;
		lineNumber?: number;
		currentText: string;
		characterName?: string | null;
		editionName?: string;
		citationRef?: string;
		senseNumber?: number;
		subSense?: string;
		onclose: () => void;
	} = $props();

	let correctionText = $state('');
	let notes = $state('');
	let category = $state('other');
	let submitting = $state(false);

	async function submit() {
		if (!correctionText.trim()) return;
		submitting = true;

		const correction = corrections.add({
			type,
			entryKey,
			workTitle,
			act,
			scene,
			lineNumber,
			currentText,
			characterName,
			editionName,
			citationRef,
			senseNumber,
			subSense,
			category,
			correctionText: correctionText.trim(),
			notes: notes.trim()
		});

		if (dev) {
			try {
				await fetch('/api/corrections', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify(correction)
				});
				corrections.markSubmitted(correction.id);
			} catch { /* localStorage save already happened */ }
		} else {
			const url = corrections.toGitHubIssueUrl(correction);
			window.open(url, '_blank');
			corrections.markSubmitted(correction.id);
		}

		submitting = false;
		onclose();
	}

	const typeLabel = $derived(
		type === 'line' ? 'Flag Line' :
		type === 'citation' ? 'Flag Citation' :
		'Flag Entry'
	);
</script>

<div class="correction-backdrop" onclick={onclose} onkeydown={(e) => e.key === 'Escape' && onclose()} role="presentation"></div>
<div class="correction-form" role="dialog" aria-label="Submit correction">
	<div class="form-header">
		<h3>{typeLabel}</h3>
		<button class="form-close" onclick={onclose} aria-label="Close">
			<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<line x1="18" y1="6" x2="6" y2="18" />
				<line x1="6" y1="6" x2="18" y2="18" />
			</svg>
		</button>
	</div>

	<div class="form-context">
		<div class="context-row">
			<span class="context-label">Entry</span>
			<span class="context-value">{entryKey}</span>
		</div>
		{#if type === 'line' && workTitle}
			<div class="context-row">
				<span class="context-label">Location</span>
				<span class="context-value">{workTitle} {act}.{scene}.{lineNumber}</span>
			</div>
			{#if editionName}
				<div class="context-row">
					<span class="context-label">Edition</span>
					<span class="context-value">{editionName}</span>
				</div>
			{/if}
			{#if characterName}
				<div class="context-row">
					<span class="context-label">Speaker</span>
					<span class="context-value">{characterName}</span>
				</div>
			{/if}
		{:else if type === 'citation' && citationRef}
			<div class="context-row">
				<span class="context-label">Citation</span>
				<span class="context-value">{citationRef}</span>
			</div>
		{:else if type === 'entry'}
			{#if senseNumber}
				<div class="context-row">
					<span class="context-label">Sense</span>
					<span class="context-value">{senseNumber}{subSense || ''}</span>
				</div>
			{/if}
		{/if}
		<div class="context-row">
			<span class="context-label">Current</span>
			<span class="context-value current-text">{currentText}</span>
		</div>
	</div>

	<div class="form-fields">
		<label class="field">
			<span class="field-label">Issue type</span>
			<select bind:value={category}>
				{#each CORRECTION_CATEGORIES as cat}
					<option value={cat.value}>{cat.label}</option>
				{/each}
			</select>
		</label>
		<label class="field">
			<span class="field-label">Correction</span>
			<textarea
				bind:value={correctionText}
				rows="3"
				placeholder="Describe what's wrong and what it should be..."
			></textarea>
		</label>
		<label class="field">
			<span class="field-label">Notes (optional)</span>
			<textarea
				bind:value={notes}
				rows="2"
				placeholder="Any additional context..."
			></textarea>
		</label>
	</div>

	<div class="form-actions">
		<button class="btn-cancel" onclick={onclose}>Cancel</button>
		<button class="btn-submit" onclick={submit} disabled={!correctionText.trim() || submitting}>
			{submitting ? 'Saving...' : 'Flag'}
		</button>
	</div>
</div>

<style>
	.correction-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.5);
		z-index: 600;
	}

	.correction-form {
		position: fixed;
		bottom: 48px;
		left: 50%;
		transform: translateX(-50%);
		width: 92%;
		max-width: 500px;
		max-height: 70dvh;
		overflow-y: auto;
		background: var(--color-elevated);
		border: 1px solid var(--color-border);
		border-radius: 12px;
		z-index: 700;
		padding: 16px;
	}

	.form-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 12px;
	}

	.form-header h3 {
		margin: 0;
		font-size: 1rem;
		font-weight: 700;
		color: var(--color-text);
	}

	.form-close {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border: none;
		background: none;
		color: var(--color-text-muted);
		cursor: pointer;
		border-radius: 6px;
	}

	.form-close:hover {
		background: var(--color-hover);
		color: var(--color-text);
	}

	.form-context {
		background: var(--color-surface);
		border-radius: 8px;
		padding: 10px 12px;
		margin-bottom: 12px;
	}

	.context-row {
		display: flex;
		gap: 8px;
		margin-bottom: 4px;
		font-size: 0.8rem;
	}

	.context-row:last-child {
		margin-bottom: 0;
	}

	.context-label {
		color: var(--color-text-muted);
		min-width: 70px;
		flex-shrink: 0;
	}

	.context-value {
		color: var(--color-text);
	}

	.current-text {
		font-style: italic;
		color: var(--color-text-secondary);
	}

	.form-fields {
		display: flex;
		flex-direction: column;
		gap: 10px;
		margin-bottom: 12px;
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.field-label {
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	textarea, select {
		width: 100%;
		padding: 8px 10px;
		border: 1px solid var(--color-border);
		background: var(--color-surface);
		color: var(--color-text);
		font-family: inherit;
		font-size: 0.85rem;
		border-radius: 6px;
		resize: vertical;
		box-sizing: border-box;
	}

	select {
		resize: none;
		cursor: pointer;
	}

	textarea:focus, select:focus {
		outline: none;
		border-color: var(--color-accent);
	}

	.form-actions {
		display: flex;
		justify-content: flex-end;
		gap: 8px;
	}

	.btn-cancel, .btn-submit {
		padding: 6px 14px;
		border-radius: 6px;
		font-family: inherit;
		font-size: 0.8rem;
		font-weight: 600;
		cursor: pointer;
	}

	.btn-cancel {
		border: 1px solid var(--color-border);
		background: none;
		color: var(--color-text-muted);
	}

	.btn-cancel:hover {
		background: var(--color-hover);
	}

	.btn-submit {
		border: none;
		background: var(--color-accent);
		color: var(--color-bg);
	}

	.btn-submit:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn-submit:not(:disabled):hover {
		background: var(--color-accent-hover);
	}
</style>
