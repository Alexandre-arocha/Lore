import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";

import { TableOfContents } from "@/components/table-of-contents";
import { getDocument, getSource, type NavNode } from "@/lib/api";

type DocPageProps = {
  params: Promise<{
    source: string;
    docSlug?: string[];
  }>;
};

export async function generateMetadata({
  params,
}: DocPageProps): Promise<Metadata> {
  const { source: sourceSlug, docSlug } = await params;
  const [source, doc] = await Promise.all([
    getSource(sourceSlug),
    getDocument(sourceSlug, docSlug?.join("/") ?? "index"),
  ]);

  if (!source) {
    return { title: "Não encontrado — Lore" };
  }
  if (doc) {
    return {
      title: `${doc.title} · ${source.name} — Lore`,
      description: `${doc.title} — documentação de ${source.name} no acervo Lore.`,
    };
  }
  return { title: `${source.name} — Lore`, description: source.description };
}

export default async function DocPage({ params }: DocPageProps) {
  const { source: sourceSlug, docSlug } = await params;
  const source = await getSource(sourceSlug);

  if (!source) {
    notFound();
  }

  const currentSlug = docSlug?.join("/") ?? "index";
  const doc = await getDocument(sourceSlug, currentSlug);

  if (docSlug && !doc) {
    notFound();
  }

  return (
    <main className="mx-auto grid w-full max-w-7xl grid-cols-1 px-5 sm:px-6 lg:grid-cols-[248px_minmax(0,1fr)] lg:gap-0 lg:px-0 xl:grid-cols-[248px_minmax(0,1fr)_240px]">
      {/* left: source + index */}
      <aside className="max-h-[60vh] overflow-y-auto border-b border-border py-6 lg:sticky lg:top-14 lg:max-h-none lg:h-[calc(100svh-3.5rem)] lg:overflow-auto lg:border-r lg:border-b-0 lg:py-8 lg:pr-6 lg:pl-5">
        <Link
          href="/"
          className="font-mono text-xs text-faint underline-offset-4 hover:text-gold"
        >
          ← acervo
        </Link>

        <div className="mt-5">
          <p className="label mb-1">{formatKind(source.kind)}</p>
          <h1 className="text-lg font-semibold tracking-tight">{source.name}</h1>
          <p className="mt-2 text-sm leading-6 text-muted-foreground">
            {source.description}
          </p>
          <a
            className="mt-3 inline-flex items-center gap-1 font-mono text-xs text-gold underline-offset-4 hover:underline"
            href={source.official_url}
            target="_blank"
            rel="noreferrer"
          >
            ver original ↗
          </a>
        </div>

        <nav className="mt-6 border-t border-border pt-4">
          <p className="label mb-3">índice</p>
          {source.nav && source.nav.length > 0 ? (
            <NavList nodes={source.nav} sourceSlug={source.slug} activeSlug={doc?.slug} />
          ) : (
            <p className="font-mono text-xs text-faint">sem documentos ainda.</p>
          )}
        </nav>
      </aside>

      {/* center: content */}
      {doc ? (
        <article className="min-w-0 py-8 lg:px-8">
          <header className="border-b border-border pb-6">
            <p className="label mb-3">
              {source.slug} / {doc.slug}
            </p>
            <h2 className="text-3xl font-semibold tracking-tight md:text-4xl">
              {doc.title}
            </h2>
            <p className="mt-3 font-mono text-xs text-faint">
              {doc.word_count} palavras
              {doc.source.license ? ` · ${doc.source.license}` : ""}
            </p>
          </header>

          <div
            className="doc-content mt-8"
            dangerouslySetInnerHTML={{ __html: doc.content_html }}
          />

          <footer className="mt-12 border-t border-border pt-5 font-mono text-xs text-muted-foreground">
            conteúdo de{" "}
            <a
              className="text-gold underline-offset-4 hover:underline"
              href={doc.source.official_url}
              target="_blank"
              rel="noreferrer"
            >
              {doc.source.name} ↗
            </a>
            {doc.source.license ? ` · licença ${doc.source.license}` : ""}
          </footer>
        </article>
      ) : (
        <section className="min-w-0 py-8 lg:px-8">
          <p className="label mb-3">{source.slug}</p>
          <h2 className="text-3xl font-semibold tracking-tight">{source.name}</h2>
          <p className="mt-4 max-w-2xl text-muted-foreground">
            Fonte cadastrada, mas ainda sem página inicial sincronizada. Escolha
            uma entrada no índice ou rode a ingestão para popular os documentos.
          </p>
        </section>
      )}

      {/* right: on this page (scroll-spy) */}
      {doc ? <TableOfContents toc={doc.toc} /> : null}
    </main>
  );
}

function NavList({
  nodes,
  sourceSlug,
  activeSlug,
}: {
  nodes: NavNode[];
  sourceSlug: string;
  activeSlug?: string;
}) {
  return (
    <ul className="space-y-0.5">
      {nodes.map((node) => {
        const active = node.slug === activeSlug;
        return (
          <li key={`${node.slug ?? node.title}`}>
            {node.slug ? (
              <Link
                className={`-ml-px block border-l-2 py-1 pl-3 font-mono text-xs transition-colors ${
                  active
                    ? "border-l-gold text-gold"
                    : "border-transparent text-muted-foreground hover:border-l-line-strong hover:text-foreground"
                }`}
                href={`/docs/${sourceSlug}/${node.slug}`}
              >
                {node.title}
              </Link>
            ) : (
              <span className="mt-3 block py-1 font-mono text-[0.7rem] uppercase tracking-[0.14em] text-faint">
                {node.title}
              </span>
            )}
            {node.children && node.children.length > 0 ? (
              <div className="ml-3 border-l border-border pl-1">
                <NavList nodes={node.children} sourceSlug={sourceSlug} activeSlug={activeSlug} />
              </div>
            ) : null}
          </li>
        );
      })}
    </ul>
  );
}

function formatKind(value: string) {
  const labels: Record<string, string> = {
    framework: "framework",
    language: "linguagem",
    library: "biblioteca",
    tool: "ferramenta",
  };
  return labels[value] ?? value;
}
