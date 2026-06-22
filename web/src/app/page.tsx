import type { ReactNode } from "react";
import Link from "next/link";

import { getSources, searchDocuments, type Source } from "@/lib/api";

type HomeProps = {
  searchParams: Promise<{
    q?: string | string[];
    tipo?: string | string[];
    area?: string | string[];
  }>;
};

const CATEGORY_ORDER = ["frontend", "backend", "mobile", "systems"];

export default async function Home({ searchParams }: HomeProps) {
  const params = await searchParams;
  const q = firstParam(params.q).trim();
  const selectedKind = firstParam(params.tipo);
  const selectedCategory = firstParam(params.area);

  const [sources, results] = await Promise.all([
    getSources(),
    searchDocuments(q),
  ]);

  const ready = sources.filter(isReadySource);
  const pending = sources.filter((s) => !isReadySource(s));
  const filtered = ready.filter(
    (s) =>
      (!selectedKind || s.kind === selectedKind) &&
      (!selectedCategory || s.category === selectedCategory),
  );

  return (
    <main className="mx-auto w-full max-w-7xl px-5 sm:px-6 lg:px-8">
      {/* hero */}
      <section className="border-b border-border py-12 sm:py-16">
        <p className="label mb-3">// acervo</p>
        <h1 className="max-w-3xl text-4xl font-semibold tracking-tight sm:text-5xl">
          Documentação de linguagens e frameworks
        </h1>
        <p className="mt-4 max-w-2xl leading-7 text-muted-foreground">
          Um índice só, leitura limpa e busca unificada. O conteúdo fica no
          idioma original; a curadoria é em PT-BR.
        </p>

        <form
          action="/"
          className="mt-8 flex max-w-2xl border border-border bg-card focus-within:border-line-strong"
        >
          <span className="flex items-center pl-3 font-mono text-gold" aria-hidden>
            {">"}
          </span>
          <input
            name="q"
            defaultValue={q}
            placeholder="buscar em todas as fontes…"
            aria-label="Buscar na documentação"
            className="h-11 w-full bg-transparent px-3 font-mono text-sm outline-none placeholder:text-faint"
          />
          <button
            type="submit"
            className="border-l border-border bg-surface-2 px-4 font-mono text-xs uppercase tracking-wider transition-colors hover:bg-primary hover:text-primary-foreground"
          >
            buscar
          </button>
        </form>
      </section>

      {q ? (
        <SearchResults q={q} results={results} />
      ) : (
        <div className="py-10">
          <FilterBar
            sources={ready}
            selectedKind={selectedKind}
            selectedCategory={selectedCategory}
          />
          <Catalog sources={filtered} />
          <PendingNote sources={pending} />
        </div>
      )}
    </main>
  );
}

function Catalog({ sources }: { sources: Source[] }) {
  if (sources.length === 0) {
    return (
      <p className="border border-dashed border-border p-8 font-mono text-sm text-faint">
        nenhuma fonte pronta para esses filtros.
      </p>
    );
  }

  return (
    <div className="space-y-12">
      {groupByCategory(sources).map(([category, items]) => (
        <section key={category}>
          <div className="mb-4 flex items-baseline gap-4">
            <h2 className="font-mono text-xs uppercase tracking-[0.14em] text-muted-foreground">
              {formatCategory(category)}
            </h2>
            <span className="h-px flex-1 bg-border" aria-hidden />
            <span className="font-mono text-xs text-faint">
              {items.length} {items.length === 1 ? "fonte" : "fontes"}
            </span>
          </div>

          <div className="grid grid-cols-1 gap-px border border-border bg-border sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {items.map((source) => (
              <SourceTile key={source.slug} source={source} />
            ))}
          </div>
        </section>
      ))}
    </div>
  );
}

function SourceTile({ source }: { source: Source }) {
  return (
    <Link
      href={`/docs/${source.slug}`}
      className="group flex min-h-[8.5rem] flex-col gap-2 border-l-2 border-transparent bg-background p-4 transition-colors hover:border-l-gold hover:bg-surface-2"
    >
      <span className="font-mono text-xs text-faint group-hover:text-gold">
        {source.slug}
      </span>
      <span className="text-base font-semibold tracking-tight">
        {source.name}
      </span>
      <span className="line-clamp-2 text-sm text-muted-foreground">
        {source.description}
      </span>
      <span className="mt-auto font-mono text-xs text-faint">
        {formatKind(source.kind)} · {source.doc_count}{" "}
        {source.doc_count === 1 ? "doc" : "docs"}
      </span>
    </Link>
  );
}

function SearchResults({
  q,
  results,
}: {
  q: string;
  results: Awaited<ReturnType<typeof searchDocuments>>;
}) {
  return (
    <div className="py-10">
      <div className="mb-4 flex items-baseline gap-4">
        <h2 className="font-mono text-xs uppercase tracking-[0.14em] text-muted-foreground">
          resultados
        </h2>
        <span className="h-px flex-1 bg-border" aria-hidden />
        <span className="font-mono text-xs text-faint">
          {results.length} para “{q}”
        </span>
      </div>

      {results.length === 0 ? (
        <p className="border border-dashed border-border p-8 font-mono text-sm text-faint">
          nenhum resultado nas fontes ativas.
        </p>
      ) : (
        <ul className="border border-border">
          {results.map((r) => (
            <li
              key={`${r.source.slug}/${r.slug}`}
              className="border-b border-border last:border-b-0"
            >
              <Link
                href={r.document_url}
                className="block border-l-2 border-transparent p-4 transition-colors hover:border-l-gold hover:bg-surface-2"
              >
                <div className="mb-1 flex items-center gap-2 font-mono text-xs text-faint">
                  <span className="text-muted-foreground">{r.source.name}</span>
                  <span aria-hidden>/</span>
                  <span>{r.slug}</span>
                </div>
                <p className="font-semibold">{r.title}</p>
                <p className="mt-1 text-sm leading-6 text-muted-foreground">
                  {highlightExcerpt(r.excerpt)}
                </p>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function FilterBar({
  sources,
  selectedKind,
  selectedCategory,
}: {
  sources: Source[];
  selectedKind: string;
  selectedCategory: string;
}) {
  const kinds = unique(sources.map((s) => s.kind));
  const categories = unique(sources.map((s) => s.category));

  return (
    <div className="mb-8 space-y-3 border border-border bg-surface-2/20 p-3">
      <FilterGroup
        label="tipo"
        param="tipo"
        options={kinds}
        selected={selectedKind}
        preserve={{ area: selectedCategory }}
        formatter={formatKind}
      />
      <FilterGroup
        label="área"
        param="area"
        options={categories}
        selected={selectedCategory}
        preserve={{ tipo: selectedKind }}
        formatter={formatCategory}
      />
    </div>
  );
}

function FilterGroup({
  label,
  param,
  options,
  selected,
  preserve,
  formatter,
}: {
  label: string;
  param: "tipo" | "area";
  options: string[];
  selected: string;
  preserve: Partial<Record<"tipo" | "area", string>>;
  formatter: (value: string) => string;
}) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <span className="label mr-1 w-10">{label}</span>
      <FilterLink href={filterHref(param, "", preserve)} active={!selected}>
        todos
      </FilterLink>
      {options.map((option) => (
        <FilterLink
          key={option}
          href={filterHref(param, option, preserve)}
          active={selected === option}
        >
          {formatter(option)}
        </FilterLink>
      ))}
    </div>
  );
}

function FilterLink({
  href,
  active,
  children,
}: {
  href: string;
  active: boolean;
  children: ReactNode;
}) {
  return (
    <Link
      className={`border px-2.5 py-1 font-mono text-xs transition-colors ${
        active
          ? "border-gold bg-gold text-background"
          : "border-border text-muted-foreground hover:border-line-strong hover:text-foreground"
      }`}
      href={href}
    >
      {children}
    </Link>
  );
}

function PendingNote({ sources }: { sources: Source[] }) {
  if (sources.length === 0) {
    return null;
  }
  return (
    <p className="mt-12 border-t border-border pt-4 font-mono text-xs text-faint">
      pendentes ({sources.length}): {sources.map((s) => s.slug).join(" · ")}
    </p>
  );
}

function highlightExcerpt(excerpt: string) {
  return excerpt.split(/(<mark>|<\/mark>)/).reduce<ReactNode[]>(
    (parts, part, index, all) => {
      if (part === "<mark>" || part === "</mark>") {
        return parts;
      }
      const highlighted = all[index - 1] === "<mark>";
      parts.push(
        highlighted ? (
          <mark key={index} className="bg-transparent font-semibold text-gold">
            {part}
          </mark>
        ) : (
          <span key={index}>{part}</span>
        ),
      );
      return parts;
    },
    [],
  );
}

function isReadySource(source: Source) {
  return source.status === "active" && source.doc_count > 0;
}

function groupByCategory(sources: Source[]): [string, Source[]][] {
  const map = new Map<string, Source[]>();
  for (const s of sources) {
    const arr = map.get(s.category) ?? [];
    arr.push(s);
    map.set(s.category, arr);
  }
  return Array.from(map.entries()).sort((a, b) => {
    const ra = CATEGORY_ORDER.indexOf(a[0]);
    const rb = CATEGORY_ORDER.indexOf(b[0]);
    return (
      (ra === -1 ? 99 : ra) - (rb === -1 ? 99 : rb) ||
      a[0].localeCompare(b[0])
    );
  });
}

function filterHref(
  param: "tipo" | "area",
  value: string,
  preserve: Partial<Record<"tipo" | "area", string>>,
) {
  const params = new URLSearchParams();
  for (const [key, preservedValue] of Object.entries(preserve)) {
    if (preservedValue) {
      params.set(key, preservedValue);
    }
  }
  if (value) {
    params.set(param, value);
  }
  const qs = params.toString();
  return qs ? `/?${qs}` : "/";
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

function unique(values: string[]) {
  return Array.from(new Set(values)).sort((a, b) => a.localeCompare(b));
}

function firstParam(value: string | string[] | undefined): string {
  if (Array.isArray(value)) {
    return value[0] ?? "";
  }
  return value ?? "";
}
