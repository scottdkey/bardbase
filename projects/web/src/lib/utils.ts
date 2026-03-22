/**
 * Group an array of items by a key function.
 * Returns a Map preserving insertion order.
 */
export function groupBy<T>(items: T[], keyFn: (item: T) => string): Map<string, T[]> {
	const groups = new Map<string, T[]>();
	for (const item of items) {
		const key = keyFn(item);
		const list = groups.get(key) ?? [];
		list.push(item);
		groups.set(key, list);
	}
	return groups;
}
