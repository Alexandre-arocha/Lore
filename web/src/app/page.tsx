import type { ReactNode } from "react";
import Link from "next/link";
import { BookOpen, Database, ExternalLink, Search } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { getHealth, getSources, searchDocuments } from "@/lib/api";

type HomeProps = {
  searchParams: Promise<{ q?: string | string[] }>;
};

export default async function Home({ searchParams }: HomeProps) {
  const params = await searchParams;
  const q = firstParam(params.q).trim();
  const [health, sources, results] = await Promise.all([
    getHealth(),
    getSources(),
    searchDocuments(q),
  ]);
  const online = health?.status === "ok";

  return (
    <main className="min-h-svh bg-background">
      <section className="border-b">
        <div className="mx-auto flex w-full max-w-6xl flex-col gap-8 px-5 py-8 sm:px-6 lg:px-8">
          <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
            <div className="max-w-2xl space-y-3">
              <div className="flex items-center gap-2">
                <BookOpen className="size-5" aria-hidden />
                <span className="text-sm font-medium text-muted-foreground">
                  Hub público de documentação
                </span>
              </div>
              <h1 className="text-4xl font-bold tracking-tight sm:text-5xl">
                Atlas
              </h1>
              <p className="text-base text-muted-foreground sm:text-lg">
                Documentação técnica em um só lugar, com curadoria em PT-BR,
                leitura limpa e busca nativa no Postgres.
              </p>
            </div>

            <div className="flex w-full items-center gap-3 rounded-lg border px-3 py-2 md:w-auto">
              <span
                className={`inline-block size-2.5 rounded-full ${
                  online ? "bg-emerald-500" : "bg-red-500"
                }`}
                aria-hidden
              />
              <div className="min-w-0 text-sm">
                <p className="font-medium">API {health?.status ?? "offline"}</p>
                <p className="text-muted-foreground">
                  Banco: {health?.db ?? "inacessível"}
                </p>
              </div>
            </div>
          </div>

          <form action="/" className="flex w-full max-w-3xl gap-2">
            <div className="relative flex-1">
              <Search
                className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
                aria-hidden
              />
              <Input
                className="h-10 pl-9"
                name="q"
                placeholder="Buscar em docs sincronizadas"
                defaultValue={q}
              />
            </div>
            <Button className="h-10 px-4" type="submit">
              Buscar
            </Button>
          </form>
        </div>
      </section>

      <section className="mx-auto grid w-full max-w-6xl gap-8 px-5 py-8 sm:px-6 lg:grid-cols-[1.5fr_1fr] lg:px-8">
        <div className="space-y-4">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-xl font-semibold">
              {q ? `Resultados para "${q}"` : "Fontes configuradas"}
            </h2>
            <Badge variant="outline">
              {q ? `${results.length} resultados` : `${sources.length} fontes`}
            </Badge>
          </div>

          {q ? (
            <SearchResults results={results} />
          ) : (
            <SourceGrid sources={sources} />
          )}
        </div>

        <aside className="space-y-4">
          <div className="rounded-lg border p-4">
            <div className="mb-3 flex items-center gap-2">
              <Database className="size-4" aria-hidden />
              <h2 className="font-semibold">Estado do MVP</h2>
            </div>
            <div className="space-y-3 text-sm text-muted-foreground">
              <p>
                A API pública já expõe fontes, detalhes de fonte, documentos e
                busca FTS. Se o banco ainda não foi sincronizado, a busca fica
                vazia até rodar ingestão.
              </p>
              <p>
                Cada documento servido pela API traz licença, URL oficial e
                atribuição da fonte para manter a procedência visível.
              </p>
            </div>
          </div>

          <div className="rounded-lg border p-4">
            <h2 className="mb-3 font-semibold">Atalhos de API</h2>
            <div className="grid gap-2 text-sm">
              <ApiLink href="http://localhost:8080/api/sources">
                /api/sources
              </ApiLink>
              <ApiLink href="http://localhost:8080/api/search?q=install">
                /api/search?q=install
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

function SourceGrid({
  sources,
}: {
  sources: Awaited<ReturnType<typeof getSources>>;
}) {
  if (sources.length === 0) {
    return (
      <div className="rounded-lg border border-dashed p-8 text-sm text-muted-foreground">
        Nenhuma fonte apareceu ainda. Rode as migrations e o seed para carregar
        as definições iniciais.
      </div>
    );
  }

  return (
    <div className="grid gap-3 sm:grid-cols-2">
      {sources.map((source) => (
        <article key={source.slug} className="rounded-lg border p-4">
          <div className="mb-3 flex flex-wrap items-center gap-2">
            <Badge variant="secondary">{source.kind}</Badge>
            <Badge variant="outline">{source.category}</Badge>
            <Badge
              variant={source.status === "active" ? "outline" : "destructive"}
            >
              {source.status}
            </Badge>
          </div>
          <h3 className="text-lg font-semibold">{source.name}</h3>
          <p className="mt-2 text-sm text-muted-foreground">
            {source.description}
          </p>
          <div className="mt-4 flex flex-wrap items-center gap-3 text-sm">
            <Link
              className="font-medium underline-offset-4 hover:underline"
              href={`/docs/${source.slug}`}
            >
              Abrir leitor
            </Link>
            <a
              className="inline-flex items-center gap-1 font-medium underline-offset-4 hover:underline"
              href={source.official_url}
              target="_blank"
              rel="noreferrer"
            >
              Ver original <ExternalLink className="size-3.5" aria-hidden />
            </a>
            {source.license ? (
              <span className="text-muted-foreground">
                Licença: {source.license}
              </span>
            ) : null}
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
      <div className="rounded-lg border border-dashed p-8 text-sm text-muted-foreground">
        Nenhum resultado encontrado. Se o banco acabou de ser criado, sincronize
        uma fonte para popular os documentos.
      </div>
    );
  }

  return (
    <div className="divide-y rounded-lg border">
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
          <p className="mt-3 text-xs text-muted-foreground">
            <Link
              className="underline-offset-4 hover:underline"
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

function ApiLink({
  href,
  children,
}: {
  href: string;
  children: ReactNode;
}) {
  return (
    <a
      className="inline-flex items-center justify-between gap-3 rounded-md border px-3 py-2 font-mono text-xs underline-offset-4 hover:bg-muted hover:underline"
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
          <mark key={index} className="rounded-sm bg-yellow-200 px-0.5">
            {part}
          </mark>
        ) : (
          <span key={index}>{part}</span>
        )
      );
      return parts;
    },
    []
  );
}

function firstParam(value: string | string[] | undefined): string {
  if (Array.isArray(value)) {
    return value[0] ?? "";
  }
  return value ?? "";
}
