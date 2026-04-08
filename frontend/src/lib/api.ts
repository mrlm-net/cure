/**
 * Central API fetch wrapper for communicating with the cure backend.
 * Base URL is derived from window.__CURE_PORT__ injected by the Go server.
 */

export function getBaseUrl(): string {
	if (typeof window !== 'undefined') {
		const port = window.__CURE_PORT__;
		if (port && port > 0) {
			return `http://127.0.0.1:${port}`;
		}
	}
	return '';
}

/** Error thrown when an API request returns a non-OK status. */
export class ApiError extends Error {
	constructor(
		public status: number,
		message: string
	) {
		super(message);
		this.name = 'ApiError';
	}
}

/**
 * Typed fetch wrapper that prepends the cure backend base URL.
 * Throws ApiError for non-OK responses.
 */
export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
	const res = await fetch(`${getBaseUrl()}${path}`, init);
	if (!res.ok) {
		const body = await res.json().catch(() => ({ error: res.statusText }));
		throw new ApiError(res.status, body.error ?? res.statusText);
	}
	return res.json();
}
