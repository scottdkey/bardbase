const STORAGE_KEY = 'bardbase-theme';

export type ThemeMode = 'light' | 'dark';

function createTheme() {
	let mode = $state<ThemeMode>('dark');

	return {
		get mode() {
			return mode;
		},
		get isDark() {
			return mode === 'dark';
		},
		toggle() {
			mode = mode === 'light' ? 'dark' : 'light';
			if (typeof window !== 'undefined') {
				localStorage.setItem(STORAGE_KEY, mode);
				document.documentElement.dataset.theme = mode;
			}
		},
		init() {
			if (typeof window !== 'undefined') {
				const stored = localStorage.getItem(STORAGE_KEY);
				if (stored === 'light' || stored === 'dark') {
					mode = stored;
				}
				document.documentElement.dataset.theme = mode;
			}
		}
	};
}

export const theme = createTheme();
