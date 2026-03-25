<script lang="ts">
  import { base } from '$app/paths';
  import { siteConfig } from '$lib/config/site.js';
</script>

<svelte:head>
  <title>{siteConfig.title} &mdash; {siteConfig.description}</title>
  <meta name="description" content={siteConfig.description} />
  <meta property="og:title" content={siteConfig.title} />
  <meta property="og:description" content={siteConfig.description} />
  <meta property="og:url" content={siteConfig.url} />
  <meta property="og:locale" content={siteConfig.locale} />
</svelte:head>

<!-- Hero -->
<section class="relative overflow-hidden border-b border-[#30363d] px-4 py-20 md:py-28">
  <!-- Background grid -->
  <div
    class="pointer-events-none absolute inset-0 opacity-5"
    style="background-image: linear-gradient(#30363d 1px, transparent 1px), linear-gradient(90deg, #30363d 1px, transparent 1px); background-size: 48px 48px;"
  ></div>

  <div class="relative mx-auto max-w-4xl text-center">
    <!-- Badge -->
    <div
      class="mb-6 inline-flex items-center gap-2 rounded-full border border-[#30363d] bg-[#161b22] px-3 py-1 text-sm"
    >
      <span class="h-2 w-2 rounded-full bg-[#3fb950]"></span>
      <span class="text-[#9198a1]">Go 1.25+ &middot; Apache 2.0 &middot; Zero dependencies</span>
    </div>

    <h1 class="mb-6 text-4xl font-bold tracking-tight text-[#e6edf3] sm:text-5xl md:text-6xl">
      Automate your
      <span class="text-[#58a6ff]">development workflow</span>
    </h1>

    <p class="mx-auto mb-8 max-w-2xl text-lg text-[#9198a1] sm:text-xl">
      Cure is a Go CLI tool for AI context management, template generation, and network tracing.
      Built on Go's standard library — no dependency bloat, no supply chain risk.
    </p>

    <div class="flex flex-col items-center justify-center gap-3 sm:flex-row">
      <a
        href="{base}/docs/installation"
        class="rounded-lg bg-[#58a6ff] px-6 py-2.5 font-medium text-[#0d1117] hover:bg-[#79b8ff]"
      >
        Get started
      </a>
      <a
        href="{base}/docs"
        class="rounded-lg border border-[#30363d] px-6 py-2.5 font-medium text-[#9198a1] hover:border-[#58a6ff] hover:text-[#e6edf3]"
      >
        Browse docs
      </a>
      <a
        href={siteConfig.repoUrl}
        target="_blank"
        rel="noopener noreferrer"
        class="flex items-center gap-2 rounded-lg border border-[#30363d] px-6 py-2.5 font-medium text-[#9198a1] hover:border-[#58a6ff] hover:text-[#e6edf3]"
      >
        <svg class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor">
          <path
            d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z"
          />
        </svg>
        GitHub
      </a>
    </div>

    <!-- Quick install -->
    <div class="mt-8 inline-flex items-center gap-3 rounded-lg border border-[#30363d] bg-[#161b22] px-4 py-2.5">
      <code class="font-mono text-sm text-[#9198a1]">
        <span class="text-[#848d97]">$</span>
        <span class="text-[#e6edf3]"> go install github.com/mrlm-net/cure/cmd/cure@latest</span>
      </code>
    </div>
  </div>
</section>

<!-- Feature cards -->
<section class="px-4 py-16 md:py-20">
  <div class="mx-auto max-w-screen-xl">
    <h2 class="mb-12 text-center text-2xl font-bold text-[#e6edf3] sm:text-3xl">
      Everything you need in one tool
    </h2>

    <div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
      <!-- AI Context -->
      <div class="rounded-xl border border-[#30363d] bg-[#161b22] p-6 hover:border-[#58a6ff]/50">
        <div
          class="mb-4 flex h-10 w-10 items-center justify-center rounded-lg bg-[#58a6ff]/10 text-[#58a6ff]"
        >
          <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z"
            />
          </svg>
        </div>
        <h3 class="mb-2 font-semibold text-[#e6edf3]">AI Context</h3>
        <p class="text-sm text-[#9198a1]">
          Manage multi-turn AI conversations from the terminal. Sessions persist across runs with
          fork, resume, and REPL support.
        </p>
      </div>

      <!-- Tracing -->
      <div class="rounded-xl border border-[#30363d] bg-[#161b22] p-6 hover:border-[#58a6ff]/50">
        <div
          class="mb-4 flex h-10 w-10 items-center justify-center rounded-lg bg-[#d29922]/10 text-[#d29922]"
        >
          <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M9 20l-5.447-2.724A1 1 0 013 16.382V5.618a1 1 0 011.447-.894L9 7m0 13l6-3m-6 3V7m6 10l4.553 2.276A1 1 0 0021 18.382V7.618a1 1 0 00-.553-.894L15 4m0 13V4m0 0L9 7"
            />
          </svg>
        </div>
        <h3 class="mb-2 font-semibold text-[#e6edf3]">Network Tracing</h3>
        <p class="text-sm text-[#9198a1]">
          Trace HTTP, DNS, TCP, and UDP connections with detailed timing, TLS handshake data, and
          NDJSON or HTML output.
        </p>
      </div>

      <!-- Templates -->
      <div class="rounded-xl border border-[#30363d] bg-[#161b22] p-6 hover:border-[#58a6ff]/50">
        <div
          class="mb-4 flex h-10 w-10 items-center justify-center rounded-lg bg-[#3fb950]/10 text-[#3fb950]"
        >
          <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
            />
          </svg>
        </div>
        <h3 class="mb-2 font-semibold text-[#e6edf3]">Templates</h3>
        <p class="text-sm text-[#9198a1]">
          Generate <code class="text-xs">CLAUDE.md</code> project context files and other structured
          templates for AI assistants.
        </p>
      </div>

      <!-- MCP -->
      <div class="rounded-xl border border-[#30363d] bg-[#161b22] p-6 hover:border-[#58a6ff]/50">
        <div
          class="mb-4 flex h-10 w-10 items-center justify-center rounded-lg bg-[#d2a8ff]/10 text-[#d2a8ff]"
        >
          <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"
            />
          </svg>
        </div>
        <h3 class="mb-2 font-semibold text-[#e6edf3]">MCP Server</h3>
        <p class="text-sm text-[#9198a1]">
          Build stdlib-only MCP servers with stdio and HTTP Streamable transports. Zero external
          dependencies.
        </p>
      </div>
    </div>
  </div>
</section>

<!-- Code example -->
<section class="border-t border-[#30363d] bg-[#161b22] px-4 py-16 md:py-20">
  <div class="mx-auto max-w-screen-xl">
    <div class="grid items-center gap-12 lg:grid-cols-2">
      <div>
        <h2 class="mb-4 text-2xl font-bold text-[#e6edf3] sm:text-3xl">
          Start a conversation in seconds
        </h2>
        <p class="mb-6 text-[#9198a1]">
          The <code class="rounded bg-[#1c2128] px-1.5 py-0.5 font-mono text-sm text-[#e6edf3]"
            >cure context</code
          > command group manages multi-turn AI sessions. Sessions are persisted locally so you can
          resume, fork, or list them at any time.
        </p>
        <div class="flex flex-col gap-3">
          <a
            href="{base}/docs/context"
            class="inline-flex items-center gap-2 text-[#58a6ff] hover:text-[#79b8ff]"
          >
            <span>View cure context docs</span>
            <svg
              class="h-4 w-4"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
            </svg>
          </a>
          <a
            href="{base}/docs/pkg-agent"
            class="inline-flex items-center gap-2 text-[#9198a1] hover:text-[#e6edf3]"
          >
            <span>Embed pkg/agent in your own tools</span>
            <svg
              class="h-4 w-4"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
            </svg>
          </a>
        </div>
      </div>

      <!-- Code block -->
      <div class="rounded-xl border border-[#30363d] bg-[#0d1117] p-1">
        <div class="flex items-center gap-2 border-b border-[#30363d] px-4 py-2.5">
          <div class="flex gap-1.5">
            <span class="h-3 w-3 rounded-full bg-[#f85149]/60"></span>
            <span class="h-3 w-3 rounded-full bg-[#d29922]/60"></span>
            <span class="h-3 w-3 rounded-full bg-[#3fb950]/60"></span>
          </div>
          <span class="ml-2 text-xs text-[#848d97]">terminal</span>
        </div>
        <div class="overflow-x-auto p-4">
          <pre class="font-mono text-sm leading-relaxed"><code
><span class="text-[#848d97]"># Start a new session</span>
<span class="text-[#3fb950]">$</span> <span class="text-[#e6edf3]">cure context new \</span>
    <span class="text-[#e6edf3]">--provider claude \</span>
    <span class="text-[#e6edf3]">--message "Summarise Go 1.25 release notes"</span>

<span class="text-[#9198a1]">Go 1.25 introduces several improvements...</span>
<span class="text-[#9198a1]">session saved: a3f2c1d0e4b5...</span>

<span class="text-[#848d97]"># Resume and continue</span>
<span class="text-[#3fb950]">$</span> <span class="text-[#e6edf3]">cure context resume a3f2c1d0e4b5 \</span>
    <span class="text-[#e6edf3]">--message "Which change is most impactful?"</span>

<span class="text-[#848d97]"># List all sessions</span>
<span class="text-[#3fb950]">$</span> <span class="text-[#e6edf3]">cure context list</span>

<span class="text-[#848d97]"># Fork and explore a branch</span>
<span class="text-[#3fb950]">$</span> <span class="text-[#e6edf3]">cure context fork a3f2c1d0e4b5</span>
</code></pre>
        </div>
      </div>
    </div>
  </div>
</section>

<!-- Packages section -->
<section class="border-t border-[#30363d] px-4 py-16 md:py-20">
  <div class="mx-auto max-w-screen-xl">
    <h2 class="mb-4 text-center text-2xl font-bold text-[#e6edf3] sm:text-3xl">
      Reusable Go packages
    </h2>
    <p class="mx-auto mb-12 max-w-2xl text-center text-[#9198a1]">
      Every package in <code class="rounded bg-[#161b22] px-1.5 py-0.5 font-mono text-sm">pkg/</code>
      is independently importable. Build your own tools on top of cure's foundations.
    </p>

    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {#each [
        { name: 'pkg/terminal', desc: 'CLI routing framework with serial, concurrent, and pipeline execution strategies', slug: 'pkg-terminal' },
        { name: 'pkg/agent', desc: 'Provider-agnostic AI agent abstraction with session management and global registry', slug: 'pkg-agent' },
        { name: 'pkg/agent/store', desc: 'File-backed JSON session store with atomic writes and ID validation', slug: 'pkg-agent-store' },
        { name: 'pkg/mcp', desc: 'Stdlib-only MCP server with stdio and HTTP Streamable transports', slug: 'pkg-mcp' },
        { name: 'pkg/config', desc: 'Hierarchical configuration merging with dot-notation access and multiple loaders', slug: 'pkg-config' },
        { name: 'pkg/tracer', desc: 'Network event tracing for HTTP, TCP, and UDP with NDJSON and HTML output', slug: 'trace' },
      ] as pkg}
        <a
          href="{base}/docs/{pkg.slug}"
          class="group rounded-lg border border-[#30363d] bg-[#161b22] p-5 hover:border-[#58a6ff]/50"
        >
          <code
            class="mb-2 block font-mono text-sm font-medium text-[#58a6ff] group-hover:text-[#79b8ff]"
            >{pkg.name}</code
          >
          <p class="text-sm text-[#9198a1]">{pkg.desc}</p>
        </a>
      {/each}
    </div>
  </div>
</section>

<!-- CTA -->
<section class="border-t border-[#30363d] bg-[#161b22] px-4 py-16 md:py-20">
  <div class="mx-auto max-w-2xl text-center">
    <h2 class="mb-4 text-2xl font-bold text-[#e6edf3] sm:text-3xl">Ready to get started?</h2>
    <p class="mb-8 text-[#9198a1]">
      Install cure in seconds with <code class="font-mono text-[#e6edf3]">go install</code> or build
      from source.
    </p>
    <div class="flex flex-col items-center justify-center gap-3 sm:flex-row">
      <a
        href="{base}/docs/installation"
        class="rounded-lg bg-[#58a6ff] px-6 py-2.5 font-medium text-[#0d1117] hover:bg-[#79b8ff]"
      >
        Installation guide
      </a>
      <a
        href="{base}/docs/quick-start"
        class="rounded-lg border border-[#30363d] px-6 py-2.5 font-medium text-[#9198a1] hover:border-[#58a6ff] hover:text-[#e6edf3]"
      >
        Quick start
      </a>
    </div>
  </div>
</section>
