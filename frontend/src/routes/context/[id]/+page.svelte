<script lang="ts">
	import { page } from '$app/stores';
	import { apiFetch, getBaseUrl } from '$lib/api';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import ErrorBanner from '$lib/components/ErrorBanner.svelte';
	import ChatBubble from '$lib/components/ChatBubble.svelte';

	interface Message {
		role: 'user' | 'assistant';
		content: string;
	}

	interface Session {
		id: string;
		provider: string;
		model: string;
		history: Message[];
	}

	interface SSEEvent {
		kind: string;
		text?: string;
		error?: string;
	}

	const sessionId = $derived($page.params.id ?? '');

	let session = $state<Session | null>(null);
	let messages = $state<Message[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let inputText = $state('');
	let streaming = $state(false);
	let streamLoading = $state(false);
	let streamBuffer = $state('');
	let messagesContainer: HTMLDivElement | undefined = $state();
	let textarea: HTMLTextAreaElement | undefined = $state();
	let userScrolledUp = $state(false);

	async function fetchSession(): Promise<void> {
		try {
			const data = await apiFetch<Session>(`/api/context/sessions/${sessionId}`);
			session = data;
			messages = data.history ?? [];
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load session';
		} finally {
			loading = false;
		}
	}

	async function refreshSession(): Promise<void> {
		try {
			const data = await apiFetch<Session>(`/api/context/sessions/${sessionId}`);
			session = data;
			messages = data.history ?? [];
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to refresh session';
		}
	}

	function scrollToBottom(): void {
		if (userScrolledUp || !messagesContainer) return;
		requestAnimationFrame(() => {
			if (messagesContainer) {
				messagesContainer.scrollTop = messagesContainer.scrollHeight;
			}
		});
	}

	function handleScroll(): void {
		if (!messagesContainer) return;
		const { scrollTop, scrollHeight, clientHeight } = messagesContainer;
		const distanceFromBottom = scrollHeight - scrollTop - clientHeight;
		userScrolledUp = distanceFromBottom > 50;
	}

	function autoResize(): void {
		if (!textarea) return;
		textarea.style.height = 'auto';
		const lineHeight = 24;
		const maxHeight = lineHeight * 6;
		textarea.style.height = `${Math.min(textarea.scrollHeight, maxHeight)}px`;
	}

	function handleKeydown(event: KeyboardEvent): void {
		if (event.key === 'Enter' && !event.shiftKey) {
			event.preventDefault();
			handleSubmit();
		}
	}

	async function handleSubmit(): Promise<void> {
		if (streaming || !inputText.trim()) return;

		const userMessage = inputText.trim();
		inputText = '';
		messages = [...messages, { role: 'user', content: userMessage }];
		streaming = true;
		streamLoading = true;
		streamBuffer = '';
		userScrolledUp = false;
		error = null;

		// Reset textarea height
		if (textarea) {
			textarea.style.height = 'auto';
		}

		scrollToBottom();

		try {
			const base = getBaseUrl();
			const res = await fetch(`${base}/api/context/sessions/${sessionId}/messages`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ message: userMessage }),
			});

			if (!res.ok) throw new Error(`Request failed: ${res.status}`);
			if (!res.body) throw new Error('No response body');

			const reader = res.body.pipeThrough(new TextDecoderStream()).getReader();
			let buffer = '';

			while (true) {
				const { done, value } = await reader.read();
				if (done) break;

				buffer += value;
				const lines = buffer.split('\n');
				buffer = lines.pop() ?? '';

				for (const line of lines) {
					if (!line.startsWith('data: ')) continue;
					try {
						const event: SSEEvent = JSON.parse(line.slice(6));
						if (event.kind === 'start') {
							// Keep streamLoading = true until the first token arrives.
							// The gap between 'start' and the first token is the model
							// generation time — the "Thinking..." indicator covers it.
						} else if (event.kind === 'token') {
							streamLoading = false; // first token clears thinking indicator
							streamBuffer += event.text ?? '';
							scrollToBottom();
						} else if (event.kind === 'done') {
							streaming = false;
							await refreshSession();
						} else if (event.kind === 'error') {
							error = event.error ?? 'Stream error';
							streaming = false;
						}
					} catch {
						/* skip non-JSON lines */
					}
				}
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Unknown error';
		} finally {
			streaming = false;
			streamLoading = false;
		}
	}

	/** Combined messages including the live stream buffer as an in-progress assistant message. */
	const displayMessages = $derived.by(() => {
		if (streaming && streamBuffer) {
			return [...messages, { role: 'assistant' as const, content: streamBuffer }];
		}
		return messages;
	});

	$effect(() => {
		fetchSession();
	});

	// Auto-scroll when messages change
	$effect(() => {
		// Subscribe to displayMessages length change
		displayMessages.length;
		scrollToBottom();
	});
</script>

<svelte:head>
	<title>Session {sessionId.slice(0, 8)} - cure</title>
</svelte:head>

<!-- Full-bleed layout: negate parent padding -->
<div class="-m-6 flex h-[calc(100vh-3.5rem)] flex-col md:h-screen">
	<!-- Header -->
	<div
		class="flex items-center gap-3 border-b border-white/10 bg-[#0d1117] px-4 py-3 md:px-6"
	>
		<a
			href="/context"
			class="rounded-md p-1 text-[rgba(230,237,243,0.5)] transition-colors hover:bg-white/5 hover:text-[#e6edf3]"
			aria-label="Back to sessions"
		>
			<svg
				width="20"
				height="20"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
				aria-hidden="true"
			>
				<path d="M15 18l-6-6 6-6" />
			</svg>
		</a>
		<div class="min-w-0">
			<span class="font-mono text-sm text-[#58a6ff]">{sessionId.slice(0, 8)}</span>
			{#if session?.model}
				<span class="ml-2 text-xs text-[rgba(230,237,243,0.4)]">{session.model}</span>
			{/if}
		</div>
	</div>

	<!-- Messages area -->
	{#if loading}
		<div class="flex flex-1 items-center justify-center">
			<LoadingSpinner />
		</div>
	{:else if error && messages.length === 0}
		<div class="flex-1 p-4 md:p-6">
			<ErrorBanner message={error} onDismiss={() => (error = null)} />
		</div>
	{:else}
		<div
			bind:this={messagesContainer}
			onscroll={handleScroll}
			class="flex-1 overflow-y-auto px-4 py-4 md:px-6"
		>
			{#if displayMessages.length === 0}
				<div class="flex h-full items-center justify-center">
					<p class="text-sm text-[rgba(230,237,243,0.3)]">
						Send a message to start the conversation
					</p>
				</div>
			{:else}
				<div class="mx-auto max-w-3xl space-y-4">
					{#each displayMessages as msg, i}
						{@const isStreamingMsg = streaming && i === displayMessages.length - 1 && msg.role === 'assistant'}
						<ChatBubble
							role={msg.role}
							content={msg.content}
							streaming={isStreamingMsg}
						/>
					{/each}
				</div>
			{/if}

			<!-- Stream loading indicator -->
			{#if streamLoading}
				<div class="mx-auto mt-4 flex max-w-3xl items-center gap-2">
					<LoadingSpinner size="sm" />
					<span class="text-xs text-[rgba(230,237,243,0.4)]">Thinking...</span>
				</div>
			{/if}

			<!-- Inline error during streaming -->
			{#if error && messages.length > 0}
				<div class="mx-auto mt-4 max-w-3xl">
					<ErrorBanner message={error} onDismiss={() => (error = null)} />
				</div>
			{/if}
		</div>
	{/if}

	<!-- Input area -->
	<div
		class="border-t border-white/10 bg-[#0d1117] px-4 py-3 md:px-6"
		style="padding-bottom: max(0.75rem, env(safe-area-inset-bottom));"
	>
		<form
			onsubmit={(e) => { e.preventDefault(); handleSubmit(); }}
			class="mx-auto flex max-w-3xl items-end gap-3"
		>
			<textarea
				bind:this={textarea}
				bind:value={inputText}
				oninput={autoResize}
				onkeydown={handleKeydown}
				placeholder="Type a message..."
				rows={1}
				disabled={streaming}
				aria-label="Message input"
				class="flex-1 resize-none rounded-lg border border-white/10 bg-[#161b22] px-4 py-2.5 text-sm text-[#e6edf3] placeholder:text-[rgba(230,237,243,0.3)] focus:border-[#58a6ff]/50 focus:outline-none disabled:opacity-50"
			></textarea>
			<button
				type="submit"
				disabled={streaming || !inputText.trim()}
				aria-label="Send message"
				aria-disabled={streaming || !inputText.trim()}
				class="shrink-0 rounded-lg bg-[#58a6ff] p-2.5 text-[#0d1117] transition-colors hover:bg-[#79b8ff] disabled:opacity-50 disabled:cursor-not-allowed"
			>
				<svg
					width="18"
					height="18"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
					aria-hidden="true"
				>
					<path d="M22 2L11 13M22 2l-7 20-4-9-9-4 20-7z" />
				</svg>
			</button>
		</form>
	</div>
</div>
