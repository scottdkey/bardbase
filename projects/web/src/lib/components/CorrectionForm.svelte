<script lang="ts">
	import { dev } from '$app/environment';
	import { corrections, CORRECTION_CATEGORIES, type CorrectionType } from '$lib/stores/corrections.svelte';
	import IconClose from '$lib/components/icons/IconClose.svelte';
	import IconButton from '$lib/components/ui/IconButton.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import FormField from '$lib/components/ui/FormField.svelte';

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
		<IconButton onclick={onclose} label="Close" size={28}>
			<IconClose size={16} />
		</IconButton>
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
		<FormField label="Issue type">
			<select bind:value={category}>
				{#each CORRECTION_CATEGORIES as cat (cat.value)}
					<option value={cat.value}>{cat.label}</option>
				{/each}
			</select>
		</FormField>
		<FormField label="Correction">
			<textarea
				bind:value={correctionText}
				rows="3"
				placeholder="Describe what's wrong and what it should be..."
			></textarea>
		</FormField>
		<FormField label="Notes (optional)">
			<textarea
				bind:value={notes}
				rows="2"
				placeholder="Any additional context..."
			></textarea>
		</FormField>
	</div>

	<div class="form-actions">
		<Button variant="secondary" onclick={onclose}>Cancel</Button>
		<Button variant="primary" onclick={submit} disabled={!correctionText.trim() || submitting}>
			{submitting ? 'Saving...' : 'Flag'}
		</Button>
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

</style>
