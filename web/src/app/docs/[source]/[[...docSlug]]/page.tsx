import type { Metadata } from "next";
import type { ReactNode } from "react";
import Link from "next/link";
import { notFound } from "next/navigation";
import {
  ArrowLeft,
  ArrowRight,
  BookOpen,
  ExternalLink,
  LibraryBig,
  Search,
} from "lucide-react";

import { TableOfContents } from "@/components/table-of-contents";
import { getDocument, getSource, type NavNode, type Source } from "@/lib/api";

type DocPageProps = {
  params: Promise<{
    source: string;
    docSlug?: string[];
  }>;
};

type FlatNavItem = {
  slug: string;
  title: string;
  section: string;
};

export async function generateMetadata({
  params,
}: DocPageProps): Promise<Metadata> {
  const { source: sourceSlug, docSlug } = await params;
  const source = await getSource(sourceSlug);

  if (!source) {
    return { title: "Não encontrado — Lore" };
  }

  if (docSlug?.length) {
    const doc = await getDocument(sourceSlug, docSlug.join("/"));
    if (doc) {
      return {
        title: `${doc.title} · ${source.name} — Lore`,
        description: `${doc.title} — documentação de ${source.name} no acervo Lore.`,
      };
    }
  }

  return {
    title: `${source.name} — Lore`,
    description: source.description,
  };
}

export default async function DocPage({ params }: DocPageProps) {
  const { source: sourceSlug, docSlug } = await params;
  const source = await getSource(sourceSlug);

  if (!source) {
    notFound();
  }

  const currentSlug = docSlug?.join("/");
  const doc = currentSlug ? await getDocument(sourceSlug, currentSlug) : null;

  if (currentSlug && !doc) {
    notFound();
  }

  const flatNav = flattenNav(source.nav ?? []);
  const activeSlug = doc?.slug;
  const activeIndex = activeSlug
    ? flatNav.findIndex((item) => item.slug === activeSlug)
    : -1;
  const previousDoc = activeIndex > 0 ? flatNav[activeIndex - 1] : null;
  const nextDoc =
    activeIndex >= 0 && activeIndex < flatNav.length - 1
      ? flatNav[activeIndex + 1]
      : null;

  return (
    <main className="mx-auto grid w-full max-w-7xl grid-cols-1 px-5 sm:px-6 lg:grid-cols-[280px_minmax(0,1fr)] lg:px-0 xl:grid-cols-[280px_minmax(0,1fr)_260px]">
      <aside className="max-h-[68vh] overflow-y-auto border-b border-border py-6 lg:sticky lg:top-14 lg:h-[calc(100svh-3.5rem)] lg:max-h-none lg:overflow-auto lg:border-r lg:border-b-0 lg:px-5 lg:py-8">
        <Link
          href="/"
          className="inline-flex items-center gap-2 font-mono text-xs text-faint underline-offset-4 hover:text-gold"
        >
          <ArrowLeft className="size-3.5" aria-hidden />
          acervo
        </Link>

        <SourceCard source={source} flatNav={flatNav} />

        <nav className="mt-6 border-t border-border pt-4">
          <div className="mb-3 flex items-center justify-between gap-3">
            <p className="label">índice</p>
            <span className="font-mono text-xs text-faint">
              {flatNav.length} docs
            </span>
          </div>
          {source.nav && source.nav.length > 0 ? (
            <NavList
              nodes={source.nav}
              sourceSlug={source.slug}
              activeSlug={activeSlug}
            />
          ) : (
            <p className="font-mono text-xs text-faint">sem documentos ainda.</p>
          )}
        </nav>
      </aside>

      {doc ? (
        <article className="min-w-0 py-8 lg:px-8">
          <header className="border-b border-border pb-6">
            <div className="mb-4 flex flex-wrap items-center gap-2 font-mono text-xs text-faint">
              <Link href={`/docs/${source.slug}`} className="hover:text-gold">
                {source.slug}
              </Link>
              <span aria-hidden>/</span>
              <span className="text-muted-foreground">{doc.slug}</span>
            </div>

            <h1 className="max-w-3xl text-3xl font-semibold md:text-4xl">
              {doc.title}
            </h1>

            <div className="mt-5 grid gap-px border border-border bg-border sm:grid-cols-3">
              <MetaTile
                label="leitura"
                value={`${readingMinutes(doc.word_count)} min`}
              />
              <MetaTile
                label="tamanho"
                value={`${formatNumber(doc.word_count)} palavras`}
              />
              <MetaTile label="licença" value={doc.source.license ?? "n/d"} />
            </div>

            <div className="mt-4 flex flex-wrap gap-2">
              <ActionLink href={source.official_url} external>
                fonte oficial
              </ActionLink>
              <ActionLink href={`/?q=${encodeURIComponent(doc.title)}`}>
                buscar relacionados
              </ActionLink>
            </div>
          </header>

          <div
            className="doc-content mt-8"
            dangerouslySetInnerHTML={{ __html: doc.content_html }}
          />

          <DocPager
            sourceSlug={source.slug}
            previousDoc={previousDoc}
            nextDoc={nextDoc}
          />

          <footer className="mt-10 border-t border-border pt-5 font-mono text-xs leading-6 text-muted-foreground">
            Conteúdo de{" "}
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
        <SourceOverview source={source} flatNav={flatNav} />
      )}

      {doc ? (
        <TableOfContents toc={doc.toc} />
      ) : (
        <SourceSections nav={source.nav ?? []} />
      )}
    </main>
  );
}

function SourceCard({
  source,
  flatNav,
}: {
  source: Source;
  flatNav: FlatNavItem[];
}) {
  return (
    <section className="mt-5">
      <p className="label mb-1">{formatKind(source.kind)}</p>
      <h2 className="text-lg font-semibold">{source.name}</h2>
      <p className="mt-2 text-sm leading-6 text-muted-foreground">
        {source.description}
      </p>
      <div className="mt-4 grid grid-cols-2 gap-px border border-border bg-border">
        <SmallStat label="docs" value={source.doc_count || flatNav.length} />
        <SmallStat label="área" value={formatCategory(source.category)} />
      </div>
    </section>
  );
}

function SourceOverview({
  source,
  flatNav,
}: {
  source: Source;
  flatNav: FlatNavItem[];
}) {
  const featured = pickFeatured(flatNav);

  return (
    <section className="min-w-0 py-8 lg:px-8">
      <p className="label mb-3">{source.slug}</p>
      <h1 className="max-w-3xl text-4xl font-semibold">
        {source.name}
      </h1>
      <p className="mt-4 max-w-2xl leading-7 text-muted-foreground">
        {source.description}
      </p>

      <div className="mt-8 grid gap-px border border-border bg-border md:grid-cols-4">
        <MetaTile
          label="documentos"
          value={`${formatNumber(source.doc_count || flatNav.length)}`}
        />
        <MetaTile label="tipo" value={formatKind(source.kind)} />
        <MetaTile label="área" value={formatCategory(source.category)} />
        <MetaTile label="licença" value={source.license ?? "n/d"} />
      </div>

      <div className="mt-6 flex flex-wrap gap-2">
        {featured[0] ? (
          <ActionLink href={`/docs/${source.slug}/${featured[0].slug}`}>
            começar leitura
          </ActionLink>
        ) : null}
        <ActionLink href={`/?q=${encodeURIComponent(source.name)}`}>
          buscar no acervo
        </ActionLink>
        <ActionLink href={source.official_url} external>
          ver original
        </ActionLink>
      </div>

      <section className="mt-10">
        <div className="mb-4 flex items-baseline gap-4">
          <h2 className="font-mono text-xs uppercase tracking-[0.14em] text-muted-foreground">
            comece por aqui
          </h2>
          <span className="h-px flex-1 bg-border" aria-hidden />
        </div>
        {featured.length > 0 ? (
          <div className="grid gap-px border border-border bg-border md:grid-cols-2">
            {featured.map((item) => (
              <Link
                key={item.slug}
                href={`/docs/${source.slug}/${item.slug}`}
                className="group bg-background p-4 transition-colors hover:bg-surface-2"
              >
                <p className="font-mono text-xs text-faint group-hover:text-gold">
                  {item.section || source.name}
                </p>
                <h3 className="mt-2 font-semibold">{item.title}</h3>
                <p className="mt-2 font-mono text-xs text-muted-foreground">
                  abrir documento →
                </p>
              </Link>
            ))}
          </div>
        ) : (
          <p className="border border-dashed border-border p-8 font-mono text-sm text-faint">
            Esta fonte ainda não tem documentos sincronizados.
          </p>
        )}
      </section>

      <section className="mt-10">
        <div className="mb-4 flex items-baseline gap-4">
          <h2 className="font-mono text-xs uppercase tracking-[0.14em] text-muted-foreground">
            seções
          </h2>
          <span className="h-px flex-1 bg-border" aria-hidden />
        </div>
        <SectionGrid sourceSlug={source.slug} nav={source.nav ?? []} />
      </section>
    </section>
  );
}

function SourceSections({ nav }: { nav: NavNode[] }) {
  if (nav.length === 0) {
    return null;
  }

  return (
    <aside className="hidden py-8 xl:block xl:border-l xl:border-border xl:pl-6">
      <div className="sticky top-[4.5rem]">
        <p className="label mb-3">seções</p>
        <nav className="border-l border-border">
          {nav.map((node) => (
            <a
              key={node.title}
              href={`#${sectionAnchor(node.title)}`}
              className="-ml-px block border-l-2 border-transparent py-1 pl-3 font-mono text-xs text-muted-foreground transition-colors hover:border-l-line-strong hover:text-foreground"
            >
              {node.title}
            </a>
          ))}
        </nav>
      </div>
    </aside>
  );
}

function SectionGrid({
  sourceSlug,
  nav,
}: {
  sourceSlug: string;
  nav: NavNode[];
}) {
  if (nav.length === 0) {
    return null;
  }

  return (
    <div className="grid gap-px border border-border bg-border md:grid-cols-2">
      {nav.map((section) => {
        const docs = flattenNav([section]);
        const firstDoc = docs[0];
        return (
          <section
            id={sectionAnchor(section.title)}
            key={section.title}
            className="bg-background p-4"
          >
            <div className="flex items-start justify-between gap-3">
              <h3 className="font-semibold">{section.title}</h3>
              <span className="font-mono text-xs text-faint">
                {docs.length}
              </span>
            </div>
            {firstDoc ? (
              <Link
                href={`/docs/${sourceSlug}/${firstDoc.slug}`}
                className="mt-3 inline-flex items-center gap-2 font-mono text-xs text-gold underline-offset-4 hover:underline"
              >
                abrir primeira página
                <ArrowRight className="size-3.5" aria-hidden />
              </Link>
            ) : (
              <p className="mt-3 font-mono text-xs text-faint">
                sem páginas diretas.
              </p>
            )}
          </section>
        );
      })}
    </div>
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
                <NavList
                  nodes={node.children}
                  sourceSlug={sourceSlug}
                  activeSlug={activeSlug}
                />
              </div>
            ) : null}
          </li>
        );
      })}
    </ul>
  );
}

function DocPager({
  sourceSlug,
  previousDoc,
  nextDoc,
}: {
  sourceSlug: string;
  previousDoc: FlatNavItem | null;
  nextDoc: FlatNavItem | null;
}) {
  if (!previousDoc && !nextDoc) {
    return null;
  }

  return (
    <nav className="mt-12 grid gap-px border border-border bg-border md:grid-cols-2">
      {previousDoc ? (
        <PagerLink
          href={`/docs/${sourceSlug}/${previousDoc.slug}`}
          label="anterior"
          title={previousDoc.title}
          direction="previous"
        />
      ) : (
        <div className="bg-background p-4" />
      )}
      {nextDoc ? (
        <PagerLink
          href={`/docs/${sourceSlug}/${nextDoc.slug}`}
          label="próximo"
          title={nextDoc.title}
          direction="next"
        />
      ) : (
        <div className="bg-background p-4" />
      )}
    </nav>
  );
}

function PagerLink({
  href,
  label,
  title,
  direction,
}: {
  href: string;
  label: string;
  title: string;
  direction: "previous" | "next";
}) {
  return (
    <Link
      href={href}
      className={`group flex min-h-28 flex-col justify-between bg-background p-4 transition-colors hover:bg-surface-2 ${
        direction === "next" ? "text-right" : ""
      }`}
    >
      <span className="font-mono text-xs text-faint group-hover:text-gold">
        {label}
      </span>
      <span className="inline-flex items-center gap-2 font-semibold">
        {direction === "previous" ? (
          <ArrowLeft className="size-4" aria-hidden />
        ) : null}
        <span>{title}</span>
        {direction === "next" ? (
          <ArrowRight className="size-4" aria-hidden />
        ) : null}
      </span>
    </Link>
  );
}

function MetaTile({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-background p-3">
      <p className="font-mono text-[0.68rem] uppercase tracking-[0.14em] text-faint">
        {label}
      </p>
      <p className="mt-1 text-sm font-medium text-foreground">{value}</p>
    </div>
  );
}

function SmallStat({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="bg-background p-3">
      <p className="font-mono text-[0.65rem] uppercase tracking-[0.14em] text-faint">
        {label}
      </p>
      <p className="mt-1 text-sm font-semibold">{value}</p>
    </div>
  );
}

function ActionLink({
  href,
  external,
  children,
}: {
  href: string;
  external?: boolean;
  children: ReactNode;
}) {
  return (
    <Link
      href={href}
      target={external ? "_blank" : undefined}
      rel={external ? "noreferrer" : undefined}
      className="inline-flex items-center gap-2 border border-border bg-surface-2/40 px-3 py-2 font-mono text-xs text-muted-foreground transition-colors hover:border-line-strong hover:text-gold"
    >
      {external ? (
        <ExternalLink className="size-3.5" aria-hidden />
      ) : href.includes("?q=") ? (
        <Search className="size-3.5" aria-hidden />
      ) : (
        <BookOpen className="size-3.5" aria-hidden />
      )}
      {children}
    </Link>
  );
}

function flattenNav(nodes: NavNode[], section = "", out: FlatNavItem[] = []) {
  for (const node of nodes) {
    const currentSection = node.slug ? section : node.title;
    if (node.slug) {
      out.push({
        slug: node.slug,
        title: node.title,
        section: section || "documentação",
      });
    }
    if (node.children?.length) {
      flattenNav(node.children, currentSection || section, out);
    }
  }
  return out;
}

function pickFeatured(items: FlatNavItem[]) {
  const priorities = ["get-started", "handbook", "tutorial", "guide", "intro"];
  const picked = new Map<string, FlatNavItem>();

  for (const priority of priorities) {
    const match = items.find(
      (item) =>
        item.slug.includes(priority) ||
        item.title.toLowerCase().includes(priority),
    );
    if (match) {
      picked.set(match.slug, match);
    }
  }

  for (const item of items) {
    if (picked.size >= 6) {
      break;
    }
    picked.set(item.slug, item);
  }

  return Array.from(picked.values()).slice(0, 6);
}

function sectionAnchor(title: string) {
  return `secao-${title
    .toLowerCase()
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "")}`;
}

function readingMinutes(words: number) {
  return Math.max(1, Math.ceil(words / 220));
}

function formatNumber(value: number) {
  return new Intl.NumberFormat("pt-BR").format(value);
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

function formatCategory(value: string) {
  const labels: Record<string, string> = {
    backend: "backend",
    frontend: "frontend",
    mobile: "mobile",
    systems: "sistemas",
  };
  return labels[value] ?? value;
}
