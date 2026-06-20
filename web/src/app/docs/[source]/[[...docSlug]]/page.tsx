import Link from "next/link";
import { notFound } from "next/navigation";
import { ExternalLink } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { getDocument, getSource, type NavNode } from "@/lib/api";

type DocPageProps = {
  params: Promise<{
    source: string;
    docSlug?: string[];
  }>;
};

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
    <main className="min-h-svh bg-background">
      <div className="mx-auto grid w-full max-w-7xl gap-8 px-5 py-6 sm:px-6 lg:grid-cols-[280px_1fr] lg:px-8">
        <aside className="lg:sticky lg:top-6 lg:h-[calc(100svh-3rem)] lg:overflow-auto">
          <Link
            className="text-sm font-medium text-muted-foreground underline-offset-4 hover:underline"
            href="/"
          >
            Voltar ao Atlas
          </Link>

          <div className="mt-5 rounded-lg border p-4">
            <div className="mb-3 flex flex-wrap gap-2">
              <Badge variant="secondary">{source.kind}</Badge>
              <Badge variant="outline">{source.category}</Badge>
            </div>
            <h1 className="text-xl font-semibold">{source.name}</h1>
            <p className="mt-2 text-sm text-muted-foreground">
              {source.description}
            </p>
            <a
              className="mt-4 inline-flex items-center gap-1 text-sm font-medium underline-offset-4 hover:underline"
              href={source.official_url}
              target="_blank"
              rel="noreferrer"
            >
              Ver original <ExternalLink className="size-3.5" aria-hidden />
            </a>
          </div>

          <nav className="mt-5 rounded-lg border p-3 text-sm">
            <p className="mb-2 px-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
              Navegação
            </p>
            {source.nav && source.nav.length > 0 ? (
              <NavList
                nodes={source.nav}
                sourceSlug={source.slug}
                activeSlug={doc?.slug}
              />
            ) : (
              <p className="px-2 py-3 text-muted-foreground">
                Nenhum documento sincronizado ainda.
              </p>
            )}
          </nav>
        </aside>

        {doc ? (
          <article className="min-w-0">
            <header className="border-b pb-5">
              <div className="mb-3 flex flex-wrap gap-2">
                {doc.source.license ? (
                  <Badge variant="outline">Licença: {doc.source.license}</Badge>
                ) : null}
                <Badge variant="outline">{doc.word_count} palavras</Badge>
              </div>
              <h2 className="text-3xl font-bold tracking-tight md:text-4xl">
                {doc.title}
              </h2>
              <p className="mt-3 text-sm text-muted-foreground">
                Fonte: {doc.source.name}. Atribuição preservada com link para a
                documentação oficial.
              </p>
            </header>

            <div
              className="doc-content mt-8"
              dangerouslySetInnerHTML={{ __html: doc.content_html }}
            />

            <footer className="mt-10 border-t pt-5 text-sm text-muted-foreground">
              <a
                className="inline-flex items-center gap-1 font-medium text-foreground underline-offset-4 hover:underline"
                href={doc.source.official_url}
                target="_blank"
                rel="noreferrer"
              >
                Ver original <ExternalLink className="size-3.5" aria-hidden />
              </a>
              {doc.source.license ? (
                <span className="ml-3">Licença: {doc.source.license}</span>
              ) : null}
            </footer>
          </article>
        ) : (
          <section className="rounded-lg border border-dashed p-8">
            <h2 className="text-2xl font-semibold">{source.name}</h2>
            <p className="mt-3 max-w-2xl text-muted-foreground">
              A fonte está cadastrada, mas ainda não há página inicial
              sincronizada para abrir. Use o endpoint admin de sync para popular
              os documentos e a navegação.
            </p>
          </section>
        )}
      </div>
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
    <ul className="space-y-1">
      {nodes.map((node) => (
        <li key={`${node.slug ?? node.title}`}>
          {node.slug ? (
            <Link
              className={`block rounded-md px-2 py-1.5 underline-offset-4 hover:bg-muted ${
                node.slug === activeSlug ? "bg-muted font-medium" : ""
              }`}
              href={`/docs/${sourceSlug}/${node.slug}`}
            >
              {node.title}
            </Link>
          ) : (
            <span className="block px-2 py-1.5 font-medium text-muted-foreground">
              {node.title}
            </span>
          )}
          {node.children && node.children.length > 0 ? (
            <div className="ml-3 border-l pl-2">
              <NavList
                nodes={node.children}
                sourceSlug={sourceSlug}
                activeSlug={activeSlug}
              />
            </div>
          ) : null}
        </li>
      ))}
    </ul>
  );
}
