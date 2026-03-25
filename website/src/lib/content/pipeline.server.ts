import fs from 'fs';
import path from 'path';
import matter from 'gray-matter';
import type { DocMeta, DocPage } from '$lib/types/index.js';

// Docs directory is one level up from website/
const DOCS_DIR = path.resolve(process.cwd(), '..', 'docs');

/**
 * Read all markdown files from the docs directory and return their frontmatter metadata.
 */
export function getAllDocMeta(): DocMeta[] {
  if (!fs.existsSync(DOCS_DIR)) {
    return [];
  }

  const files = fs.readdirSync(DOCS_DIR).filter((f) => f.endsWith('.md'));
  const docs: DocMeta[] = [];

  for (const file of files) {
    const slug = file.replace(/\.md$/, '');
    const raw = fs.readFileSync(path.join(DOCS_DIR, file), 'utf-8');
    const { data } = matter(raw);

    docs.push({
      slug,
      title: data.title ?? slug,
      description: data.description ?? '',
      order: typeof data.order === 'number' ? data.order : 99,
      section: data.section ?? 'other'
    });
  }

  return docs;
}

/**
 * Read a single doc file by slug and return its metadata and raw content.
 */
export function getDocBySlug(slug: string): DocPage | null {
  const filePath = path.join(DOCS_DIR, `${slug}.md`);

  if (!fs.existsSync(filePath)) {
    return null;
  }

  const raw = fs.readFileSync(filePath, 'utf-8');
  const { data, content } = matter(raw);

  return {
    slug,
    title: data.title ?? slug,
    description: data.description ?? '',
    order: typeof data.order === 'number' ? data.order : 99,
    section: data.section ?? 'other',
    content
  };
}

/**
 * Return all doc slugs for static prerendering.
 */
export function getAllDocSlugs(): string[] {
  if (!fs.existsSync(DOCS_DIR)) {
    return [];
  }
  return fs
    .readdirSync(DOCS_DIR)
    .filter((f) => f.endsWith('.md'))
    .map((f) => f.replace(/\.md$/, ''));
}
