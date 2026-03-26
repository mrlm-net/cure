import { error } from '@sveltejs/kit';
import { unified } from 'unified';
import remarkParse from 'remark-parse';
import remarkRehype from 'remark-rehype';
import rehypeRaw from 'rehype-raw';
import rehypeShiki from '@shikijs/rehype';
import rehypeStringify from 'rehype-stringify';
import { getAllDocSlugs, getDocBySlug } from '$lib/content/pipeline.server.js';
import rehypeSlug from '$lib/plugins/rehype-slug.js';
import rehypeRewriteLinks from '$lib/plugins/rehype-rewrite-links.js';
import type { DocPage } from '$lib/types/index.js';

export const prerender = true;

export async function entries() {
  const slugs = getAllDocSlugs();
  return slugs.map((slug) => ({ slug }));
}

const processor = unified()
  .use(remarkParse)
  .use(remarkRehype, { allowDangerousHtml: true })
  .use(rehypeRaw)
  .use(rehypeSlug)
  .use(rehypeRewriteLinks)
  .use(rehypeShiki, { theme: 'github-dark' })
  .use(rehypeStringify);

export async function load({ params }): Promise<{ doc: DocPage; html: string }> {
  const doc = getDocBySlug(params.slug);

  if (!doc) {
    throw error(404, `Documentation page "${params.slug}" not found`);
  }

  const html = String(await processor.process(doc.content));

  return { doc, html };
}
