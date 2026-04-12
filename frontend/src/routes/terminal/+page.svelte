<script lang="ts">
	import { onMount } from 'svelte';
	import { getBaseUrl } from '$lib/api';
	import ErrorBanner from '$lib/components/ErrorBanner.svelte';

	let termContainer: HTMLDivElement | undefined = $state();
	let error = $state<string | null>(null);
	let connected = $state(false);

	onMount(async () => {
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

		if (!termContainer) return;

		term.open(termContainer);

		// Connect WebSocket
		const base = getBaseUrl();
		const wsUrl = base.replace('http', 'ws') + '/api/terminal/default';
		const socket = new WebSocket(wsUrl);
		socket.binaryType = 'arraybuffer';

		socket.onopen = () => {
			connected = true;
			// Fit and focus AFTER visible and connected
			requestAnimationFrame(() => {
				fitAddon.fit();
				term.focus();
				socket.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }));
			});
		};

		socket.onmessage = (event) => {
			if (event.data instanceof ArrayBuffer) {
				term.write(new Uint8Array(event.data));
			} else {
				try {
					const data = JSON.parse(event.data);
					if (data.type === 'exit') {
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
			error = 'WebSocket connection failed';
		};

		// Keyboard input -> WebSocket
		term.onData((data: string) => {
			if (socket.readyState === WebSocket.OPEN) {
				socket.send(data);
			}
		});

		// Resize events
		term.onResize(({ cols, rows }: { cols: number; rows: number }) => {
			if (socket.readyState === WebSocket.OPEN) {
				socket.send(JSON.stringify({ type: 'resize', cols, rows }));
			}
		});

		// Refit on window resize
		const resizeObserver = new ResizeObserver(() => {
			fitAddon.fit();
		});
		resizeObserver.observe(termContainer);

		// Focus on click
		termContainer.addEventListener('click', () => term.focus());
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

	<div bind:this={termContainer} class="flex-1 overflow-hidden p-3" style="min-height: 0;"></div>
</div>
