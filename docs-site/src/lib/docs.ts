import fs from "fs";
import path from "path";
import matter from "gray-matter";

const CONTENT_DIR = path.join(process.cwd(), "src/content/docs");

export interface DocFrontmatter {
  title: string;
  description: string;
}

export function getDocBySlug(slug: string) {
  const filePath = path.join(CONTENT_DIR, `${slug}.mdx`);

  if (!fs.existsSync(filePath)) {
    return null;
  }

  const fileContent = fs.readFileSync(filePath, "utf-8");
  const { data, content } = matter(fileContent);

  return {
    frontmatter: data as DocFrontmatter,
    content,
    slug,
  };
}

export function getAllDocSlugs(): string[] {
  const slugs: string[] = [];

  function walk(dir: string, prefix: string = "") {
    const entries = fs.readdirSync(dir, { withFileTypes: true });
    for (const entry of entries) {
      if (entry.isDirectory()) {
        walk(path.join(dir, entry.name), `${prefix}${entry.name}/`);
      } else if (entry.name.endsWith(".mdx")) {
        slugs.push(`${prefix}${entry.name.replace(".mdx", "")}`);
      }
    }
  }

  walk(CONTENT_DIR);
  return slugs;
}
