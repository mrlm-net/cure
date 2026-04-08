<script lang="ts">
  import '../app.css'
  import type { Snippet } from 'svelte'

  interface Props {
    children: Snippet
  }

  let { children }: Props = $props()
</script>

<div class="min-h-screen flex flex-col">
  <nav class="border-b border-white/10 px-6 py-3">
    <span class="text-sm font-semibold tracking-wide text-white/80">cure</span>
  </nav>
  <main class="flex-1">
    {@render children()}
  </main>
</div>
