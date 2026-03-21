import { browser } from '$app/environment';

export type CorrectionType = 'line' | 'entry' | 'citation';

export interface Correction {
	id: string;
	timestamp: string;
	type: CorrectionType;
	entryKey: string;
	// Line-level fields (for scene text corrections)
	workTitle?: string;
	act?: number;
	scene?: number;
	lineNumber?: number;
	currentText: string;
	characterName?: string | null;
	editionName?: string;
	// Citation-level fields
	citationRef?: string;
	// Entry-level fields
	senseNumber?: number;
	subSense?: string;
	// Common
	category?: string; // e.g., "wrong-play", "wrong-line", "wrong-character", "parse-error", "other"
	correctionText: string;
	notes: string;
	status: 'pending' | 'submitted';
}

const STORAGE_KEY = 'bardbase-corrections';

function generateId(): string {
	return Date.now().toString(36) + Math.random().toString(36).slice(2, 8);
}

function loadFromStorage(): Correction[] {
	if (!browser) return [];
	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		return raw ? JSON.parse(raw) : [];
	} catch {
		return [];
	}
}

function saveToStorage(items: Correction[]) {
	if (!browser) return;
	localStorage.setItem(STORAGE_KEY, JSON.stringify(items));
}

export const CORRECTION_CATEGORIES = [
	{ value: 'wrong-play', label: 'Wrong play' },
	{ value: 'wrong-act', label: 'Wrong act' },
	{ value: 'wrong-scene', label: 'Wrong scene' },
	{ value: 'wrong-line', label: 'Wrong line number' },
	{ value: 'wrong-character', label: 'Wrong character/speaker' },
	{ value: 'wrong-text', label: 'Wrong text content' },
	{ value: 'parse-error', label: 'Parsing error (sense split, missing data)' },
	{ value: 'missing-citation', label: 'Missing citation' },
	{ value: 'duplicate', label: 'Duplicate entry/citation' },
	{ value: 'other', label: 'Other' }
];

class CorrectionStore {
	items = $state<Correction[]>([]);

	constructor() {
		this.items = loadFromStorage();
	}

	add(data: Omit<Correction, 'id' | 'timestamp' | 'status'>): Correction {
		const correction: Correction = {
			...data,
			id: generateId(),
			timestamp: new Date().toISOString(),
			status: 'pending'
		};
		this.items = [...this.items, correction];
		saveToStorage(this.items);
		return correction;
	}

	remove(id: string) {
		this.items = this.items.filter((c) => c.id !== id);
		saveToStorage(this.items);
	}

	markSubmitted(id: string) {
		this.items = this.items.map((c) => (c.id === id ? { ...c, status: 'submitted' as const } : c));
		saveToStorage(this.items);
	}

	get pending(): Correction[] {
		return this.items.filter((c) => c.status === 'pending');
	}

	get count(): number {
		return this.items.length;
	}

	get pendingCount(): number {
		return this.items.filter((c) => c.status === 'pending').length;
	}

	/** Check if a specific scene line is flagged */
	isFlagged(workTitle: string, act: number, scene: number, lineNumber: number): boolean {
		return this.items.some(
			(c) => c.type === 'line' && c.workTitle === workTitle && c.act === act && c.scene === scene && c.lineNumber === lineNumber
		);
	}

	/** Build a GitHub issue body for a correction */
	toGitHubIssueUrl(c: Correction): string {
		let title: string;
		let body: string;

		if (c.type === 'line') {
			title = `[Correction] ${c.entryKey}: ${c.workTitle} ${c.act}.${c.scene}.${c.lineNumber}`;
			body =
				`## Line Correction\n\n` +
				`**Entry:** ${c.entryKey}\n` +
				`**Work:** ${c.workTitle}\n` +
				`**Location:** Act ${c.act}, Scene ${c.scene}, Line ${c.lineNumber}\n` +
				`**Edition:** ${c.editionName || 'N/A'}\n` +
				`**Character:** ${c.characterName || 'N/A'}\n` +
				`**Category:** ${c.category || 'N/A'}\n\n` +
				`### Current Text\n\`\`\`\n${c.currentText}\n\`\`\`\n\n` +
				`### Proposed Correction\n${c.correctionText}\n\n` +
				`### Notes\n${c.notes || 'None'}\n`;
		} else if (c.type === 'citation') {
			title = `[Citation] ${c.entryKey}: ${c.citationRef || 'unknown ref'}`;
			body =
				`## Citation Correction\n\n` +
				`**Entry:** ${c.entryKey}\n` +
				`**Citation:** ${c.citationRef}\n` +
				`**Category:** ${c.category || 'N/A'}\n\n` +
				`### Current\n\`\`\`\n${c.currentText}\n\`\`\`\n\n` +
				`### Proposed Correction\n${c.correctionText}\n\n` +
				`### Notes\n${c.notes || 'None'}\n`;
		} else {
			title = `[Entry] ${c.entryKey}${c.senseNumber ? ` sense ${c.senseNumber}${c.subSense || ''}` : ''}`;
			body =
				`## Entry Correction\n\n` +
				`**Entry:** ${c.entryKey}\n` +
				(c.senseNumber ? `**Sense:** ${c.senseNumber}${c.subSense ? c.subSense : ''}\n` : '') +
				`**Category:** ${c.category || 'N/A'}\n\n` +
				`### Current\n\`\`\`\n${c.currentText}\n\`\`\`\n\n` +
				`### Proposed Correction\n${c.correctionText}\n\n` +
				`### Notes\n${c.notes || 'None'}\n`;
		}

		return `https://github.com/scottdkey/bardbase/issues/new?title=${encodeURIComponent(title)}&body=${encodeURIComponent(body)}&labels=correction`;
	}
}

export const corrections = new CorrectionStore();
