import { error } from '@sveltejs/kit';
import { compile } from 'mdsvex';
import { getDocBySlug } from '$lib/content/pipeline.server.js';
import rehypeSlug from '$lib/plugins/rehype-slug.js';
import rehypeRewriteLinks from '$lib/plugins/rehype-rewrite-links.js';
import type { DocPage } from '$lib/types/index.js';

export const prerender = true;

export async function load(): Promise<{ doc: DocPage; html: string }> {
  const doc = getDocBySlug('changelog');

  if (!doc) {
    throw error(404, 'Changelog not found');
  }

  const compiled = await compile(doc.content, {
    rehypePlugins: [rehypeSlug, rehypeRewriteLinks]
  });

  const html = compiled?.code ?? '';
  const htmlContent = html
    .replace(/<script[^>]*>[\s\S]*?<\/script>/gi, '')
    .replace(/<style[^>]*>[\s\S]*?<\/style>/gi, '')
    .trim();

  return { doc, html: htmlContent };
}
