import type { ReactNode } from "react";
import Link from "next/link";
import {
  Activity,
  BookOpen,
  Database,
  ExternalLink,
  Search,
} from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  getHealth,
  getSources,
  searchDocuments,
  type Source,
} from "@/lib/api";

type HomeProps = {
  searchParams: Promise<{
    q?: string | string[];
    tipo?: string | string[];
    area?: string | string[];
  }>;
};

export default async function Home({ searchParams }: HomeProps) {
  const params = await searchParams;
  const q = firstParam(params.q).trim();
  const selectedKind = firstParam(params.tipo);
  const selectedCategory = firstParam(params.area);
  const [health, sources, results] = await Promise.all([
    getHealth(),
    getSources(),
    searchDocuments(q),
  ]);

  const readySources = sources.filter(isReadySource);
  const pendingSources = sources.filter((source) => !isReadySource(source));
  const filteredSources = readySources.filter(
    (source) =>
      (!selectedKind || source.kind === selectedKind) &&
      (!selectedCategory || source.category === selectedCategory),
  );
  const totalDocs = readySources.reduce((sum, source) => sum + source.doc_count, 0);
  const online = health?.status === "ok";

  return (
    <main className="min-h-svh bg-background">
      <section className="border-b border-border">
        <div className="mx-auto grid w-full max-w-7xl gap-8 px-5 py-10 sm:px-6 lg:grid-cols-[1fr_320px] lg:px-8">
          <div className="space-y-8">
            <div className="max-w-3xl space-y-4">
              <div className="flex items-center gap-2">
                <BookOpen className="size-5 text-gold" aria-hidden />
                <span className="label">acervo publico de documentacao</span>
              </div>
              <div className="space-y-3">
                <h1 className="text-4xl font-semibold tracking-tight sm:text-5xl">
                  Lore
                </h1>
                <p className="max-w-2xl text-base leading-7 text-muted-foreground sm:text-lg">
                  Documentacao tecnica em um indice so: linguagens, frameworks e
                  bibliotecas com busca unificada, navegacao rapida e leitura
                  sem distracao.
                </p>
              </div>
            </div>

            <form action="/" className="flex w-full max-w-3xl gap-2">
              <div className="relative flex-1">
                <Search
                  className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-faint"
                  aria-hidden
                />
                <Input
                  className="h-11 border-border bg-surface-2/40 pl-9 font-mono"
                  name="q"
                  placeholder="buscar em docs sincronizadas"
                  defaultValue={q}
                />
              </div>
              <Button className="h-11 px-5" type="submit">
                Buscar
              </Button>
            </form>
          </div>

          <div className="grid grid-cols-2 border border-border bg-surface-2/20 sm:grid-cols-4 lg:grid-cols-2">
            <Metric label="fontes prontas" value={readySources.length} />
            <Metric label="paginas" value={totalDocs.toLocaleString("pt-BR")} />
            <Metric label="areas" value={unique(readySources.map((s) => s.category)).length} />
            <Metric
              label="api"
              value={online ? "ok" : "off"}
              tone={online ? "good" : "bad"}
            />
          </div>
        </div>
      </section>

      <section className="mx-auto grid w-full max-w-7xl gap-8 px-5 py-8 sm:px-6 lg:grid-cols-[minmax(0,1fr)_300px] lg:px-8">
        <div className="space-y-6">
          {q ? (
            <>
              <SectionHeading
                title={`Resultados para "${q}"`}
                detail={`${results.length} encontrados`}
              />
              <SearchResults results={results} />
            </>
          ) : (
            <>
              <SectionHeading
                title="Fontes sincronizadas"
                detail={`${filteredSources.length} de ${readySources.length} prontas`}
              />
              <FilterBar
                sources={readySources}
                selectedKind={selectedKind}
                selectedCategory={selectedCategory}
              />
              <SourceGrid sources={filteredSources} />
            </>
          )}
        </div>

        <aside className="space-y-4">
          <div className="border border-border p-4">
            <div className="mb-3 flex items-center gap-2">
              <Database className="size-4 text-gold" aria-hidden />
              <h2 className="font-semibold">Estado do acervo</h2>
            </div>
            <div className="space-y-3 text-sm leading-6 text-muted-foreground">
              <p>
                A busca usa Postgres FTS nativo e ignora fontes que ainda nao
                estao ativas, evitando resultados parciais ou instaveis.
              </p>
              <p>
                Cada pagina mostra atribuicao, licenca e link para a fonte
                oficial.
              </p>
            </div>
          </div>

          <PendingSources sources={pendingSources} />

          <div className="border border-border p-4">
            <h2 className="mb-3 font-semibold">Atalhos de API</h2>
            <div className="grid gap-2 text-sm">
              <ApiLink href="http://localhost:8080/api/sources">
                /api/sources
              </ApiLink>
              <ApiLink href="http://localhost:8080/api/search?q=function">
                /api/search?q=function
              </ApiLink>
              <ApiLink href="http://localhost:8080/api/health">
                /api/health
              </ApiLink>
            </div>
          </div>
        </aside>
      </section>
    </main>
  );
}

function Metric({
  label,
  value,
  tone = "neutral",
}: {
  label: string;
  value: ReactNode;
  tone?: "neutral" | "good" | "bad";
}) {
  return (
    <div className="border-b border-r border-border p-4 last:border-r-0 sm:last:border-r lg:[&:nth-child(even)]:border-r-0">
      <p className="label">{label}</p>
      <p
        className={`mt-2 font-mono text-2xl font-medium ${
          tone === "good"
            ? "text-gold"
            : tone === "bad"
              ? "text-destructive"
              : "text-foreground"
        }`}
      >
        {value}
      </p>
    </div>
  );
}

function SectionHeading({ title, detail }: { title: string; detail: string }) {
  return (
    <div className="flex flex-wrap items-end justify-between gap-3 border-b border-border pb-3">
      <h2 className="text-xl font-semibold">{title}</h2>
      <span className="font-mono text-xs text-faint">{detail}</span>
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
  const kinds = unique(sources.map((source) => source.kind));
  const categories = unique(sources.map((source) => source.category));

  return (
    <div className="space-y-3 border border-border bg-surface-2/20 p-3">
      <FilterGroup
        label="tipo"
        param="tipo"
        options={kinds}
        selected={selectedKind}
        preserve={{ area: selectedCategory }}
        formatter={formatKind}
      />
      <FilterGroup
        label="area"
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
      <span className="label mr-1">{label}</span>
      <FilterLink
        href={filterHref(param, "", preserve)}
        active={!selected}
      >
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

function SourceGrid({ sources }: { sources: Source[] }) {
  if (sources.length === 0) {
    return (
      <div className="border border-dashed border-border p-8 text-sm text-muted-foreground">
        Nenhuma fonte pronta para esses filtros.
      </div>
    );
  }

  return (
    <div className="grid gap-3 md:grid-cols-2">
      {sources.map((source) => (
        <article
          key={source.slug}
          className="border border-border bg-card p-4 transition-colors hover:border-line-strong"
        >
          <div className="mb-3 flex flex-wrap items-center gap-2">
            <Badge variant="secondary">{formatKind(source.kind)}</Badge>
            <Badge variant="outline">{formatCategory(source.category)}</Badge>
            <Badge variant="outline">{source.doc_count} paginas</Badge>
          </div>
          <h3 className="text-lg font-semibold">{source.name}</h3>
          <p className="mt-2 min-h-12 text-sm leading-6 text-muted-foreground">
            {source.description}
          </p>
          <div className="mt-4 flex flex-wrap items-center gap-3 text-sm">
            <Link
              className="font-medium text-gold underline-offset-4 hover:underline"
              href={`/docs/${source.slug}`}
            >
              Abrir leitor
            </Link>
            <a
              className="inline-flex items-center gap-1 text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
              href={source.official_url}
              target="_blank"
              rel="noreferrer"
            >
              Original <ExternalLink className="size-3.5" aria-hidden />
            </a>
          </div>
        </article>
      ))}
    </div>
  );
}

function SearchResults({
  results,
}: {
  results: Awaited<ReturnType<typeof searchDocuments>>;
}) {
  if (results.length === 0) {
    return (
      <div className="border border-dashed border-border p-8 text-sm text-muted-foreground">
        Nenhum resultado encontrado nas fontes ativas.
      </div>
    );
  }

  return (
    <div className="divide-y divide-border border border-border">
      {results.map((result) => (
        <article key={`${result.source.slug}/${result.slug}`} className="p-4">
          <div className="mb-2 flex flex-wrap items-center gap-2">
            <Badge variant="secondary">{result.source.name}</Badge>
            {result.source.license ? (
              <Badge variant="outline">{result.source.license}</Badge>
            ) : null}
          </div>
          <h3 className="text-lg font-semibold">{result.title}</h3>
          <p className="mt-2 text-sm leading-6 text-muted-foreground">
            {highlightExcerpt(result.excerpt)}
          </p>
          <p className="mt-3 font-mono text-xs text-faint">
            <Link
              className="underline-offset-4 hover:text-gold hover:underline"
              href={`/docs/${result.source.slug}/${result.slug}`}
            >
              {result.source.slug}/{result.slug}
            </Link>
          </p>
        </article>
      ))}
    </div>
  );
}

function PendingSources({ sources }: { sources: Source[] }) {
  if (sources.length === 0) {
    return null;
  }

  return (
    <div className="border border-border p-4">
      <div className="mb-3 flex items-center gap-2">
        <Activity className="size-4 text-gold" aria-hidden />
        <h2 className="font-semibold">Pendentes</h2>
      </div>
      <div className="space-y-3">
        {sources.map((source) => (
          <div key={source.slug} className="border-l border-border pl-3">
            <p className="text-sm font-medium">{source.name}</p>
            <p className="font-mono text-xs text-faint">
              {source.status} · {source.doc_count} paginas
            </p>
          </div>
        ))}
      </div>
    </div>
  );
}

function ApiLink({
  href,
  children,
}: {
  href: string;
  children: ReactNode;
}) {
  return (
    <a
      className="inline-flex items-center justify-between gap-3 border border-border px-3 py-2 font-mono text-xs text-muted-foreground underline-offset-4 hover:border-line-strong hover:text-foreground hover:underline"
      href={href}
      target="_blank"
      rel="noreferrer"
    >
      {children}
      <ExternalLink className="size-3.5" aria-hidden />
    </a>
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
  } else {
    params.delete(param);
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
  return Array.from(new Set(values)).sort((a, b) =>
    formatCategory(a).localeCompare(formatCategory(b), "pt-BR"),
  );
}

function firstParam(value: string | string[] | undefined): string {
  if (Array.isArray(value)) {
    return value[0] ?? "";
  }
  return value ?? "";
}
