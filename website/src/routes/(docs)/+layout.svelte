<script lang="ts">
  import Header from '$lib/components/layout/Header.svelte';
  import Sidebar from '$lib/components/layout/Sidebar.svelte';
  import Footer from '$lib/components/layout/Footer.svelte';
  import type { NavSection } from '$lib/types/index.js';

  interface Props {
    data: { sections: NavSection[] };
    children: import('svelte').Snippet;
  }

  let { data, children }: Props = $props();

  let mobileMenuOpen = $state(false);
</script>

<Header onMenuToggle={() => (mobileMenuOpen = !mobileMenuOpen)} />

<Sidebar
  sections={data.sections}
  mobileOpen={mobileMenuOpen}
  onClose={() => (mobileMenuOpen = false)}
/>

<!-- Main content offset for fixed sidebar -->
<div class="min-h-screen pt-14">
  <div class="md:pl-64">
    <main class="mx-auto max-w-4xl px-4 py-10 md:px-8 md:py-16">
      {@render children()}
    </main>
    <Footer />
  </div>
</div>
