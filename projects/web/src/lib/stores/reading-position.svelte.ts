import { browser } from '$app/environment';

export interface ReadingPosition {
	act: number;
	scene: number;
	scrollY: number;
	timestamp: number;
}

const STORAGE_KEY = 'bardbase-reading-positions';

function loadFromStorage(): Record<string, ReadingPosition> {
	if (!browser) return {};
	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		return raw ? JSON.parse(raw) : {};
	} catch {
		return {};
	}
}

function saveToStorage(positions: Record<string, ReadingPosition>) {
	if (!browser) return;
	localStorage.setItem(STORAGE_KEY, JSON.stringify(positions));
}

class ReadingPositionStore {
	positions = $state<Record<string, ReadingPosition>>({});

	constructor() {
		this.positions = loadFromStorage();
	}

	get(workId: number): ReadingPosition | null {
		return this.positions[String(workId)] ?? null;
	}

	save(workId: number, act: number, scene: number, scrollY: number) {
		this.positions = {
			...this.positions,
			[String(workId)]: { act, scene, scrollY, timestamp: Date.now() }
		};
		saveToStorage(this.positions);
	}

	getAll(): Record<string, ReadingPosition> {
		return this.positions;
	}

	clear(workId: number) {
		const { [String(workId)]: _, ...rest } = this.positions;
		this.positions = rest;
		saveToStorage(this.positions);
	}
}

export const readingPosition = new ReadingPositionStore();
