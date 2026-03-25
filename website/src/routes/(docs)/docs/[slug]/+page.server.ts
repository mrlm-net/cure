import { error } from '@sveltejs/kit';
import { compile } from 'mdsvex';
import { getAllDocSlugs, getDocBySlug } from '$lib/content/pipeline.server.js';
import rehypeSlug from '$lib/plugins/rehype-slug.js';
import rehypeRewriteLinks from '$lib/plugins/rehype-rewrite-links.js';
import type { DocPage } from '$lib/types/index.js';

export const prerender = true;

export async function entries() {
  const slugs = getAllDocSlugs();
  return slugs.map((slug) => ({ slug }));
}

export async function load({ params }): Promise<{ doc: DocPage; html: string }> {
  const doc = getDocBySlug(params.slug);

  if (!doc) {
    throw error(404, `Documentation page "${params.slug}" not found`);
  }

  // Compile markdown to HTML via mdsvex
  const compiled = await compile(doc.content, {
    rehypePlugins: [rehypeSlug, rehypeRewriteLinks]
  });

  const html = compiled?.code ?? '';

  // Strip mdsvex module script wrappers — extract the inner HTML
  // mdsvex wraps output in <script context="module"> and component code;
  // for SSG we just need the rendered HTML portion
  const htmlContent = extractHtml(html);

  return { doc, html: htmlContent };
}

/**
 * Extract the rendered HTML from mdsvex compiled output.
 * mdsvex produces Svelte component code; we strip script and style blocks
 * to get the pure HTML template content.
 */
function extractHtml(code: string): string {
  return code
    .replace(/<script[^>]*>[\s\S]*?<\/script>/gi, '')
    .replace(/<style[^>]*>[\s\S]*?<\/style>/gi, '')
    .trim();
}
