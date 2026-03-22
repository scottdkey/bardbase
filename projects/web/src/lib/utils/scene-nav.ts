import type { WorkDivision } from '$lib/server/api';

export function findAdjacentScenes(
	toc: WorkDivision[],
	currentAct: number,
	currentScene: number
): { prev: WorkDivision | null; next: WorkDivision | null } {
	const idx = toc.findIndex((d) => d.act === currentAct && d.scene === currentScene);
	return {
		prev: idx > 0 ? toc[idx - 1] : null,
		next: idx >= 0 && idx < toc.length - 1 ? toc[idx + 1] : null
	};
}
