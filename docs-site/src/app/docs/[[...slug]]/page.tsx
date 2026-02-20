import { notFound, redirect } from "next/navigation";
import { MDXRemote } from "next-mdx-remote/rsc";
import { getDocBySlug, getAllDocSlugs } from "@/lib/docs";
import { getBreadcrumbs, getPrevNext } from "@/lib/navigation";
import { extractToc } from "@/lib/toc";
import { Breadcrumbs } from "@/components/Breadcrumbs";
import { PrevNext } from "@/components/PrevNext";
import { TableOfContents } from "@/components/TableOfContents";
import { mdxComponents } from "@/components/mdx";
import remarkGfm from "remark-gfm";
import rehypePrettyCode from "rehype-pretty-code";
import rehypeSlug from "rehype-slug";

interface PageProps {
  params: Promise<{ slug?: string[] }>;
}

export async function generateStaticParams() {
  const slugs = getAllDocSlugs();
  return slugs.map((slug) => ({
    slug: slug.split("/"),
  }));
}

export async function generateMetadata({ params }: PageProps) {
  const { slug } = await params;
  if (!slug) return { title: "Documentation" };

  const slugPath = slug.join("/");
  const doc = getDocBySlug(slugPath);

  if (!doc) return { title: "Documentation" };

  return {
    title: doc.frontmatter.title,
    description: doc.frontmatter.description,
  };
}

export default async function DocPage({ params }: PageProps) {
  const { slug } = await params;

  if (!slug || slug.length === 0) {
    redirect("/docs/getting-started/introduction");
  }

  const slugPath = slug.join("/");
  const doc = getDocBySlug(slugPath);

  if (!doc) {
    notFound();
  }

  const breadcrumbs = getBreadcrumbs(slugPath);
  const { prev, next } = getPrevNext(slugPath);
  const toc = extractToc(doc.content);

  return (
    <div className="flex gap-10">
      <div className="flex-1 min-w-0">
        <Breadcrumbs items={breadcrumbs} />
        <h1 className="text-3xl font-bold text-text-primary mb-2">
          {doc.frontmatter.title}
        </h1>
        {doc.frontmatter.description && (
          <p className="text-lg text-text-secondary mb-8 leading-relaxed">
            {doc.frontmatter.description}
          </p>
        )}
        <article className="prose-custom">
          <MDXRemote
            source={doc.content}
            components={mdxComponents}
            options={{
              mdxOptions: {
                remarkPlugins: [remarkGfm],
                rehypePlugins: [
                  rehypeSlug,
                  [
                    rehypePrettyCode,
                    {
                      theme: {
                        dark: "github-dark-dimmed",
                        light: "github-light",
                      },
                      keepBackground: false,
                      defaultLang: "go",
                    },
                  ],
                ],
              },
            }}
          />
        </article>
        <PrevNext prev={prev} next={next} />
      </div>
      <TableOfContents items={toc} />
    </div>
  );
}
