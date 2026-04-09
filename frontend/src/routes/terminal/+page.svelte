<script lang="ts">
	import { onMount } from 'svelte';
	import { getBaseUrl } from '$lib/api';
	import ErrorBanner from '$lib/components/ErrorBanner.svelte';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';

	let termContainer: HTMLDivElement | undefined = $state();
	let terminal: any = $state(null);
	let ws: WebSocket | null = $state(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let connected = $state(false);

	onMount(async () => {
		try {
			const { Terminal } = await import('@xterm/xterm');
			const { FitAddon } = await import('@xterm/addon-fit');

			const term = new Terminal({
				fontSize: 13,
				fontFamily: 'Menlo, Monaco, "Courier New", monospace',
				theme: {
					background: '#0d1117',
					foreground: '#e6edf3',
					cursor: '#58a6ff',
					selectionBackground: 'rgba(88, 166, 255, 0.3)',
				},
				cursorBlink: true,
				scrollback: 5000,
			});

			const fitAddon = new FitAddon();
			term.loadAddon(fitAddon);

			if (termContainer) {
				term.open(termContainer);
				fitAddon.fit();

				const resizeObserver = new ResizeObserver(() => fitAddon.fit());
				resizeObserver.observe(termContainer);
			}

			terminal = term;

			// Connect WebSocket
			const base = getBaseUrl();
			const wsUrl = base.replace('http', 'ws') + '/api/terminal/default';
			const socket = new WebSocket(wsUrl);
			socket.binaryType = 'arraybuffer';

			socket.onopen = () => {
				connected = true;
				loading = false;
				// Send initial size
				const msg = JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows });
				socket.send(msg);
			};

			socket.onmessage = (event) => {
				if (event.data instanceof ArrayBuffer) {
					const text = new TextDecoder().decode(event.data);
					term.write(text);
				} else {
					try {
						const data = JSON.parse(event.data);
						if (data.type === 'output') {
							term.write(atob(data.data));
						} else if (data.type === 'exit') {
							term.writeln('\r\n[Session ended]');
							connected = false;
						}
					} catch {
						term.write(event.data);
					}
				}
			};

			socket.onclose = () => {
				connected = false;
				term.writeln('\r\n[Disconnected]');
			};

			socket.onerror = () => {
				error = 'Terminal WebSocket connection failed. Backend may not support terminal yet.';
				loading = false;
			};

			ws = socket;

			// Forward input to WebSocket
			term.onData((data: string) => {
				if (socket.readyState === WebSocket.OPEN) {
					socket.send(data);
				}
			});

			term.onResize(({ cols, rows }: { cols: number; rows: number }) => {
				if (socket.readyState === WebSocket.OPEN) {
					socket.send(JSON.stringify({ type: 'resize', cols, rows }));
				}
			});

		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load terminal';
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>Terminal - cure</title>
	<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@xterm/xterm@5/css/xterm.min.css" />
</svelte:head>

<div class="-m-6 flex h-[calc(100vh-3.5rem)] flex-col md:h-screen">
	<!-- Header -->
	<div class="flex h-11 items-center gap-2 border-b border-[var(--border)] bg-[var(--bg-secondary)] px-4">
		<span class="text-xs text-[var(--text-secondary)]">Terminal</span>
		{#if connected}
			<span class="h-2 w-2 rounded-full bg-[var(--success)]"></span>
		{:else}
			<span class="h-2 w-2 rounded-full bg-[var(--text-tertiary)]"></span>
		{/if}
	</div>

	{#if error}
		<div class="p-4">
			<ErrorBanner message={error} onDismiss={() => (error = null)} />
		</div>
	{/if}

	{#if loading}
		<div class="flex flex-1 items-center justify-center">
			<LoadingSpinner />
		</div>
	{/if}

	<div bind:this={termContainer} class="flex-1 p-3" style="display: {loading ? 'none' : 'block'}"></div>
</div>
