export interface DocMeta {
  title: string;
  description: string;
  order: number;
  section: string;
  slug: string;
}

export interface DocPage extends DocMeta {
  content: string;
}

export interface NavSection {
  id: string;
  title: string;
  defaultOpen: boolean;
  docs: DocMeta[];
}

export interface SiteConfig {
  title: string;
  description: string;
  repoUrl: string;
  basePath: string;
  url: string;
  ogImage: string;
  ogImageWidth: number;
  ogImageHeight: number;
  locale: string;
  license: string;
}
