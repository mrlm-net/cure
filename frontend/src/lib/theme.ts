/** Theme management — persists to localStorage, respects OS preference. */

export type Theme = 'light' | 'dark';

const STORAGE_KEY = 'cure-theme';

export function getTheme(): Theme {
	if (typeof window === 'undefined') return 'dark';
	const stored = localStorage.getItem(STORAGE_KEY);
	if (stored === 'light' || stored === 'dark') return stored;
	return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
}

export function setTheme(theme: Theme): void {
	if (typeof window === 'undefined') return;
	localStorage.setItem(STORAGE_KEY, theme);
	document.documentElement.setAttribute('data-theme', theme);
}

export function toggleTheme(): Theme {
	const current = getTheme();
	const next = current === 'dark' ? 'light' : 'dark';
	setTheme(next);
	return next;
}

export function initTheme(): void {
	setTheme(getTheme());
}
