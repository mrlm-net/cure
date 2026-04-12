<script lang="ts">
	import { goto } from '$app/navigation';
	import { apiFetch } from '$lib/api';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import ErrorBanner from '$lib/components/ErrorBanner.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import { formatRelativeTime } from '$lib/utils';

	interface Session {
		id: string;
		provider: string;
		model: string;
		updated_at: string;
		turns: number;
		name?: string;
		project_name?: string;
		branch_name?: string;
		work_items?: string[];
		agent_role?: string;
		skill_name?: string;
	}

	interface ProjectInfo {
		name: string;
		description?: string;
	}

	let sessions = $state<Session[]>([]);
	let projects = $state<ProjectInfo[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let mutating = $state(false);
	let showCreateForm = $state(false);
	let selectedProject = $state('');
	let selectedTarget = $state(''); // empty = local, "agent-N" = container
	let deleteTarget = $state<string | null>(null);

	async function fetchSessions(): Promise<void> {
		try {
			sessions = await apiFetch<Session[]>('/api/context/sessions');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load sessions';
		} finally {
			loading = false;
		}
	}

	async function fetchProjects(): Promise<void> {
		try {
			projects = await apiFetch<ProjectInfo[]>('/api/project');
			if (projects.length > 0 && !selectedProject) {
				selectedProject = projects[0].name;
			}
		} catch {
			// Projects API may not be available — not critical
		}
	}

	async function createSession(): Promise<void> {
		if (mutating || !selectedProject) return;
		mutating = true;
		error = null;
		try {
			const body: any = { project_name: selectedProject };
			if (selectedTarget) body.container_id = selectedTarget;
			const data = await apiFetch<{ id: string }>('/api/context/sessions', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(body)
			});
			showCreateForm = false;
			if (data?.id) {
				goto(`/context/${data.id}`);
			} else {
				await fetchSessions();
			}
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

	function confirmDelete(id: string): void {
		deleteTarget = id;
	}

	async function doDelete(): Promise<void> {
		if (!deleteTarget || mutating) return;
		mutating = true;
		error = null;
		const id = deleteTarget;
		deleteTarget = null;
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
		fetchProjects();
	});
</script>

<svelte:head>
	<title>Context - cure</title>
</svelte:head>

<div class="space-y-6">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<h1 class="text-xl font-semibold tracking-tight text-[var(--text-primary)]">Sessions</h1>
		<button
			onclick={() => (showCreateForm = !showCreateForm)}
			class="rounded-md bg-[var(--accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--accent-hover)]"
		>
			New Session
		</button>
	</div>

	<!-- Create session form -->
	{#if showCreateForm}
		<div class="rounded-lg border border-[#58a6ff]/30 bg-[var(--bg-secondary)] p-4">
			<div class="flex items-end gap-3">
				<div class="flex-1">
					<label for="project-select" class="mb-1 block text-xs font-medium text-[var(--text-secondary)]">Project scope</label>
					<select
						id="project-select"
						bind:value={selectedProject}
						class="w-full rounded-md border border-[var(--border)] bg-[var(--bg-primary)] px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent)]/50 focus:outline-none"
					>
						{#each projects as p}
							<option value={p.name}>{p.name}{p.description ? ` — ${p.description}` : ''}</option>
						{/each}
					</select>
				</div>
				<div class="flex-1">
					<label for="target-select" class="mb-1 block text-xs font-medium text-[var(--text-secondary)]">Run target</label>
					<select
						id="target-select"
						bind:value={selectedTarget}
						class="w-full rounded-md border border-[var(--border)] bg-[var(--bg-primary)] px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent)]/50 focus:outline-none"
					>
						<option value="">Local (this machine)</option>
						<option value="agent-1">Container: agent-1</option>
						<option value="agent-2">Container: agent-2</option>
						<option value="agent-3">Container: agent-3</option>
						<option value="agent-4">Container: agent-4</option>
					</select>
				</div>
				<button
					onclick={createSession}
					disabled={mutating || !selectedProject}
					class="rounded-md bg-[var(--accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--accent-hover)] disabled:opacity-50 disabled:cursor-not-allowed"
				>
					{mutating ? 'Creating...' : 'Create'}
				</button>
				<button
					onclick={() => (showCreateForm = false)}
					class="rounded-md bg-[var(--bg-tertiary)]/50 px-3 py-2 text-sm text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)]"
				>
					Cancel
				</button>
			</div>
			{#if projects.length === 0}
				<p class="mt-2 text-xs text-[var(--text-tertiary)]">No projects found. Create one with <code class="rounded bg-[var(--bg-tertiary)]/50 px-1 py-0.5">cure project init</code></p>
			{/if}
		</div>
	{/if}

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
				stroke="var(--text-tertiary)"
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
			<p class="text-sm text-[var(--text-secondary)]">No sessions yet</p>
			<p class="mt-1 text-xs text-[var(--text-tertiary)]">Create one to get started</p>
		</div>

	<!-- Session list -->
	{:else}
		<div class="space-y-3">
			{#each sessions as session (session.id)}
				<div
					class="flex items-center justify-between gap-4 rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] px-4 py-3"
				>
					<!-- Session info -->
					<div class="min-w-0 flex-1">
						<div class="flex items-center gap-3">
							<span class="font-mono text-sm text-[var(--accent)]">
								{session.name || session.id.slice(0, 8)}
							</span>
							{#if session.provider}
								<span
									class="rounded bg-[var(--bg-tertiary)]/50 px-2 py-0.5 text-xs text-[var(--text-secondary)]"
								>
									{session.provider}
								</span>
							{/if}
							{#if session.agent_role}
								<span class="rounded bg-[var(--accent)]/10 px-2 py-0.5 text-xs text-[var(--accent)]/70">
									{session.agent_role}
								</span>
							{/if}
						</div>
						<div class="mt-1 flex flex-wrap items-center gap-3 text-xs text-[var(--text-tertiary)]">
							<span>{formatRelativeTime(session.updated_at)}</span>
							<span>{session.turns} turn{session.turns !== 1 ? 's' : ''}</span>
							{#if session.project_name}
								<span class="text-[var(--text-secondary)]">{session.project_name}</span>
							{/if}
							{#if session.branch_name}
								<span class="font-mono text-[var(--text-tertiary)]">{session.branch_name}</span>
							{/if}
							{#if session.work_items?.length}
								<span class="text-[var(--accent)]/50">
									{session.work_items.map(w => `#${w}`).join(', ')}
								</span>
							{/if}
						</div>
					</div>

					<!-- Actions -->
					<div class="flex items-center gap-2">
						<a
							href="/context/{session.id}"
							class="rounded-md bg-[var(--bg-tertiary)]/50 px-3 py-1.5 text-xs text-[var(--accent)] transition-colors hover:bg-[var(--bg-tertiary)]"
							aria-label="Open session {session.id.slice(0, 8)}"
						>
							Open
						</a>
						<button
							onclick={() => forkSession(session.id)}
							disabled={mutating}
							class="rounded-md bg-[var(--bg-tertiary)]/50 px-3 py-1.5 text-xs text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)] disabled:opacity-50 disabled:cursor-not-allowed"
							aria-label="Fork session {session.id.slice(0, 8)}"
						>
							Fork
						</button>
						<button
							onclick={() => confirmDelete(session.id)}
							disabled={mutating}
							class="rounded-md bg-[var(--bg-tertiary)]/50 px-3 py-1.5 text-xs text-[var(--danger)]/70 transition-colors hover:bg-[var(--danger)]/10 hover:text-[var(--danger)] disabled:opacity-50 disabled:cursor-not-allowed"
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

{#if deleteTarget}
	<ConfirmDialog
		title="Delete Session"
		message="This session will be permanently deleted. This cannot be undone."
		confirmLabel="Delete"
		danger={true}
		onConfirm={doDelete}
		onCancel={() => (deleteTarget = null)}
	/>
{/if}
