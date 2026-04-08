<script lang="ts">
	import { apiFetch } from '$lib/api';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import ErrorBanner from '$lib/components/ErrorBanner.svelte';

	let serverPort = $state(0);
	let sessionCount = $state<number | null>(null);
	let sessionError = $state<string | null>(null);
	let doctorStatus = $state<string | null>(null);
	let doctorError = $state<string | null>(null);

	$effect(() => {
		if (typeof window !== 'undefined') {
			serverPort = window.__CURE_PORT__ || 0;
		}
	});

	$effect(() => {
		apiFetch<{ count: number }>('/api/context/sessions')
			.then((data) => {
				sessionCount = data.count;
			})
			.catch((err) => {
				sessionError = err instanceof Error ? err.message : 'Failed to load sessions';
			});
	});

	$effect(() => {
		apiFetch<{ status: string }>('/api/doctor')
			.then((data) => {
				doctorStatus = data.status;
			})
			.catch((err) => {
				doctorError = err instanceof Error ? err.message : 'Failed to load doctor status';
			});
	});

	const serverAddress = $derived(
		serverPort > 0 ? `http://127.0.0.1:${serverPort}` : 'Not configured'
	);

	const doctorColor = $derived(
		doctorStatus === 'pass'
			? 'text-[#3fb950]'
			: doctorStatus === 'warn'
				? 'text-[#d29922]'
				: doctorStatus === 'fail'
					? 'text-[#f85149]'
					: 'text-[rgba(230,237,243,0.5)]'
	);
</script>

<svelte:head>
	<title>Dashboard - cure</title>
</svelte:head>

<div class="space-y-6">
	<h1 class="text-xl font-semibold tracking-tight text-[#e6edf3]">Dashboard</h1>

	<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
		<!-- Server Info Card -->
		<div class="rounded-lg border border-white/10 bg-[#161b22] p-4">
			<h2 class="mb-2 text-xs font-medium uppercase tracking-wider text-[rgba(230,237,243,0.5)]">
				Server Info
			</h2>
			<p class="text-sm font-mono text-[#e6edf3]">{serverAddress}</p>
		</div>

		<!-- Sessions Card -->
		<div class="rounded-lg border border-white/10 bg-[#161b22] p-4">
			<h2 class="mb-2 text-xs font-medium uppercase tracking-wider text-[rgba(230,237,243,0.5)]">
				Sessions
			</h2>
			{#if sessionError}
				<ErrorBanner message={sessionError} onDismiss={() => (sessionError = null)} />
			{:else if sessionCount !== null}
				<p class="text-2xl font-semibold text-[#e6edf3]">{sessionCount}</p>
			{:else}
				<div class="flex items-center gap-2">
					<LoadingSpinner size="sm" />
					<span class="text-sm text-[rgba(230,237,243,0.5)]">---</span>
				</div>
			{/if}
		</div>

		<!-- Doctor Status Card -->
		<div class="rounded-lg border border-white/10 bg-[#161b22] p-4">
			<h2 class="mb-2 text-xs font-medium uppercase tracking-wider text-[rgba(230,237,243,0.5)]">
				Doctor
			</h2>
			{#if doctorError}
				<ErrorBanner message={doctorError} onDismiss={() => (doctorError = null)} />
			{:else if doctorStatus !== null}
				<p class="text-lg font-semibold capitalize {doctorColor}">{doctorStatus}</p>
			{:else}
				<div class="flex items-center gap-2">
					<LoadingSpinner size="sm" />
					<span class="text-sm text-[rgba(230,237,243,0.5)]">---</span>
				</div>
			{/if}
		</div>

		<!-- Quick Actions Card -->
		<div class="rounded-lg border border-white/10 bg-[#161b22] p-4">
			<h2 class="mb-2 text-xs font-medium uppercase tracking-wider text-[rgba(230,237,243,0.5)]">
				Quick Actions
			</h2>
			<div class="flex flex-col gap-2">
				<a
					href="/context"
					class="inline-flex items-center gap-2 rounded-md bg-white/5 px-3 py-2 text-sm text-[#58a6ff] transition-colors hover:bg-white/10"
				>
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
						<path d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />
					</svg>
					Context
				</a>
				<a
					href="/doctor"
					class="inline-flex items-center gap-2 rounded-md bg-white/5 px-3 py-2 text-sm text-[#58a6ff] transition-colors hover:bg-white/10"
				>
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
						<path d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
					</svg>
					Doctor
				</a>
			</div>
		</div>
	</div>
</div>
