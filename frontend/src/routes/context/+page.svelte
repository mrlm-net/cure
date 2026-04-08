<script lang="ts">
	import { goto } from '$app/navigation';
	import { apiFetch } from '$lib/api';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import ErrorBanner from '$lib/components/ErrorBanner.svelte';
	import { formatRelativeTime } from '$lib/utils';

	interface Session {
		id: string;
		provider: string;
		model: string;
		updated_at: string;
		turn_count: number;
	}

	let sessions = $state<Session[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let mutating = $state(false);

	async function fetchSessions(): Promise<void> {
		try {
			const data = await apiFetch<{ sessions: Session[] }>('/api/context/sessions');
			sessions = data.sessions ?? [];
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load sessions';
		} finally {
			loading = false;
		}
	}

	async function createSession(): Promise<void> {
		if (mutating) return;
		mutating = true;
		error = null;
		try {
			await apiFetch<{ id: string }>('/api/context/sessions', { method: 'POST' });
			await fetchSessions();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create session';
		} finally {
			mutating = false;
		}
	}

	async function forkSession(id: string): Promise<void> {
		if (mutating) return;
		mutating = true;
		error = null;
		try {
			await apiFetch<{ id: string }>(`/api/context/sessions/${id}/fork`, { method: 'POST' });
			await fetchSessions();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to fork session';
		} finally {
			mutating = false;
		}
	}

	async function deleteSession(id: string): Promise<void> {
		if (mutating) return;
		if (!window.confirm(`Delete session ${id.slice(0, 8)}...? This cannot be undone.`)) return;
		mutating = true;
		error = null;
		try {
			await apiFetch<void>(`/api/context/sessions/${id}`, { method: 'DELETE' });
			await fetchSessions();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete session';
		} finally {
			mutating = false;
		}
	}

	$effect(() => {
		fetchSessions();
	});
</script>

<svelte:head>
	<title>Context - cure</title>
</svelte:head>

<div class="space-y-6">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<h1 class="text-xl font-semibold tracking-tight text-[#e6edf3]">Context Sessions</h1>
		<button
			onclick={createSession}
			disabled={mutating}
			class="rounded-md bg-[#58a6ff] px-4 py-2 text-sm font-medium text-[#0d1117] transition-colors hover:bg-[#79b8ff] disabled:opacity-50 disabled:cursor-not-allowed"
		>
			{mutating ? 'Working...' : 'New Session'}
		</button>
	</div>

	<!-- Error -->
	{#if error}
		<ErrorBanner message={error} onDismiss={() => (error = null)} />
	{/if}

	<!-- Loading -->
	{#if loading}
		<div class="flex items-center justify-center py-12">
			<LoadingSpinner />
		</div>

	<!-- Empty state -->
	{:else if sessions.length === 0}
		<div class="flex flex-col items-center justify-center py-16 text-center">
			<svg
				width="48"
				height="48"
				viewBox="0 0 24 24"
				fill="none"
				stroke="rgba(230,237,243,0.2)"
				stroke-width="1.5"
				stroke-linecap="round"
				stroke-linejoin="round"
				aria-hidden="true"
				class="mb-4"
			>
				<path
					d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z"
				/>
			</svg>
			<p class="text-sm text-[rgba(230,237,243,0.5)]">No sessions yet</p>
			<p class="mt-1 text-xs text-[rgba(230,237,243,0.3)]">Create one to get started</p>
		</div>

	<!-- Session list -->
	{:else}
		<div class="space-y-3">
			{#each sessions as session (session.id)}
				<div
					class="flex items-center justify-between gap-4 rounded-lg border border-white/10 bg-[#161b22] px-4 py-3"
				>
					<!-- Session info -->
					<div class="min-w-0 flex-1">
						<div class="flex items-center gap-3">
							<span class="font-mono text-sm text-[#58a6ff]">
								{session.id.slice(0, 8)}
							</span>
							{#if session.provider}
								<span
									class="rounded bg-white/5 px-2 py-0.5 text-xs text-[rgba(230,237,243,0.5)]"
								>
									{session.provider}
								</span>
							{/if}
							{#if session.model}
								<span class="text-xs text-[rgba(230,237,243,0.4)]">
									{session.model}
								</span>
							{/if}
						</div>
						<div class="mt-1 flex items-center gap-3 text-xs text-[rgba(230,237,243,0.3)]">
							<span>{formatRelativeTime(session.updated_at)}</span>
							<span>{session.turn_count} turn{session.turn_count !== 1 ? 's' : ''}</span>
						</div>
					</div>

					<!-- Actions -->
					<div class="flex items-center gap-2">
						<a
							href="/context/{session.id}"
							class="rounded-md bg-white/5 px-3 py-1.5 text-xs text-[#58a6ff] transition-colors hover:bg-white/10"
							aria-label="Open session {session.id.slice(0, 8)}"
						>
							Open
						</a>
						<button
							onclick={() => forkSession(session.id)}
							disabled={mutating}
							class="rounded-md bg-white/5 px-3 py-1.5 text-xs text-[rgba(230,237,243,0.5)] transition-colors hover:bg-white/10 hover:text-[#e6edf3] disabled:opacity-50 disabled:cursor-not-allowed"
							aria-label="Fork session {session.id.slice(0, 8)}"
						>
							Fork
						</button>
						<button
							onclick={() => deleteSession(session.id)}
							disabled={mutating}
							class="rounded-md bg-white/5 px-3 py-1.5 text-xs text-[#f85149]/70 transition-colors hover:bg-red-500/10 hover:text-[#f85149] disabled:opacity-50 disabled:cursor-not-allowed"
							aria-label="Delete session {session.id.slice(0, 8)}"
						>
							Delete
						</button>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
