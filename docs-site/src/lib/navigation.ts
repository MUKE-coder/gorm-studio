export interface NavItem {
  title: string;
  slug: string;
  href: string;
}

export interface NavGroup {
  title: string;
  items: NavItem[];
}

export const navigation: NavGroup[] = [
  {
    title: "Getting Started",
    items: [
      { title: "Introduction", slug: "getting-started/introduction", href: "/docs/getting-started/introduction" },
      { title: "Installation", slug: "getting-started/installation", href: "/docs/getting-started/installation" },
      { title: "Quick Start", slug: "getting-started/quick-start", href: "/docs/getting-started/quick-start" },
    ],
  },
  {
    title: "Configuration",
    items: [
      { title: "Basic Config", slug: "configuration/basic", href: "/docs/configuration/basic" },
      { title: "Authentication", slug: "configuration/authentication", href: "/docs/configuration/authentication" },
      { title: "CORS", slug: "configuration/cors", href: "/docs/configuration/cors" },
      { title: "Security", slug: "configuration/security", href: "/docs/configuration/security" },
    ],
  },
  {
    title: "Database Support",
    items: [
      { title: "SQLite", slug: "databases/sqlite", href: "/docs/databases/sqlite" },
      { title: "PostgreSQL", slug: "databases/postgresql", href: "/docs/databases/postgresql" },
      { title: "MySQL", slug: "databases/mysql", href: "/docs/databases/mysql" },
    ],
  },
  {
    title: "Features",
    items: [
      { title: "Schema Browser", slug: "features/schema-browser", href: "/docs/features/schema-browser" },
      { title: "Data Management", slug: "features/data-management", href: "/docs/features/data-management" },
      { title: "SQL Editor", slug: "features/sql-editor", href: "/docs/features/sql-editor" },
      { title: "Relationships", slug: "features/relationships", href: "/docs/features/relationships" },
      { title: "Soft Deletes", slug: "features/soft-deletes", href: "/docs/features/soft-deletes" },
    ],
  },
  {
    title: "Export",
    items: [
      { title: "Schema Export", slug: "export/schema", href: "/docs/export/schema" },
      { title: "Data Export", slug: "export/data", href: "/docs/export/data" },
      { title: "Go Models", slug: "export/models", href: "/docs/export/models" },
    ],
  },
  {
    title: "Import",
    items: [
      { title: "Schema Import", slug: "import/schema", href: "/docs/import/schema" },
      { title: "Data Import", slug: "import/data", href: "/docs/import/data" },
      { title: "Go Models", slug: "import/models", href: "/docs/import/models" },
    ],
  },
  {
    title: "API Reference",
    items: [
      { title: "Schema API", slug: "api/schema", href: "/docs/api/schema" },
      { title: "Rows API", slug: "api/rows", href: "/docs/api/rows" },
      { title: "Export API", slug: "api/export", href: "/docs/api/export" },
      { title: "Import API", slug: "api/import", href: "/docs/api/import" },
      { title: "SQL API", slug: "api/sql", href: "/docs/api/sql" },
      { title: "Stats API", slug: "api/stats", href: "/docs/api/stats" },
    ],
  },
];

export function getAllItems(): NavItem[] {
  return navigation.flatMap((group) => group.items);
}

export function getPrevNext(currentSlug: string) {
  const items = getAllItems();
  const index = items.findIndex((item) => item.slug === currentSlug);
  return {
    prev: index > 0 ? items[index - 1] : null,
    next: index < items.length - 1 ? items[index + 1] : null,
  };
}

export function getBreadcrumbs(currentSlug: string) {
  for (const group of navigation) {
    const item = group.items.find((i) => i.slug === currentSlug);
    if (item) {
      return [
        { title: "Docs", href: "/docs/getting-started/introduction" },
        { title: group.title, href: group.items[0].href },
        { title: item.title, href: item.href },
      ];
    }
  }
  return [{ title: "Docs", href: "/docs/getting-started/introduction" }];
}
