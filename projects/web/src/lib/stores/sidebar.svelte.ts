function createSidebar() {
	let isOpen = $state(false);

	return {
		get open() { return isOpen; },
		toggle() { isOpen = !isOpen; },
		close() { isOpen = false; },
		show() { isOpen = true; }
	};
}

export const sidebar = createSidebar();
