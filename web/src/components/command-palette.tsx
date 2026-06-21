"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";

import {
  Command,
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { searchDocuments, type SearchResult } from "@/lib/api";

export function CommandPalette() {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setOpen((prev) => !prev);
      }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, []);

  useEffect(() => {
    const q = query.trim();
    if (!q) {
      setResults([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    const timer = setTimeout(async () => {
      const data = await searchDocuments(q);
      setResults(data);
      setLoading(false);
    }, 200);
    return () => clearTimeout(timer);
  }, [query]);

  const go = useCallback(
    (url: string) => {
      setOpen(false);
      setQuery("");
      router.push(url);
    },
    [router],
  );

  const groups = groupBySource(results);
  const trimmed = query.trim();

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="inline-flex h-9 items-center gap-2 border border-border bg-surface-2/40 px-3 font-mono text-xs text-muted-foreground transition-colors hover:border-line-strong hover:text-foreground"
      >
        <span className="text-gold">{">"}</span>
        <span className="hidden sm:inline">buscar</span>
        <kbd className="ml-1 hidden border border-border px-1.5 py-0.5 text-[10px] text-faint sm:inline">
          ⌘K
        </kbd>
      </button>

      <CommandDialog
        open={open}
        onOpenChange={setOpen}
        title="Buscar na documentação"
        description="Busca em todas as fontes do Lore"
        className="border-border"
      >
        <Command shouldFilter={false} className="bg-popover">
          <CommandInput
            value={query}
            onValueChange={setQuery}
            placeholder="buscar em todas as fontes…"
            className="font-mono"
          />
          <CommandList>
            {!trimmed ? (
              <CommandEmpty className="font-mono text-xs text-faint">
                digite para buscar em todas as fontes
              </CommandEmpty>
            ) : loading ? (
              <CommandEmpty className="font-mono text-xs text-faint">
                buscando…
              </CommandEmpty>
            ) : results.length === 0 ? (
              <CommandEmpty className="font-mono text-xs text-faint">
                nenhum resultado
              </CommandEmpty>
            ) : null}

            {groups.map((group) => (
              <CommandGroup
                key={group.source}
                heading={group.source}
                className="[&_[cmdk-group-heading]]:font-mono [&_[cmdk-group-heading]]:uppercase [&_[cmdk-group-heading]]:tracking-[0.14em] [&_[cmdk-group-heading]]:text-faint"
              >
                {group.items.map((r) => (
                  <CommandItem
                    key={`${r.source.slug}/${r.slug}`}
                    value={`${r.source.slug}/${r.slug}`}
                    onSelect={() => go(r.document_url)}
                    className="border-l-2 border-transparent data-[selected=true]:border-gold data-[selected=true]:bg-surface-2"
                  >
                    <div className="flex min-w-0 flex-col gap-0.5">
                      <span className="truncate text-sm font-medium text-foreground">
                        {r.title}
                      </span>
                      <span
                        className="truncate font-mono text-xs text-muted-foreground [&_mark]:bg-transparent [&_mark]:font-semibold [&_mark]:text-gold"
                        dangerouslySetInnerHTML={{ __html: r.excerpt }}
                      />
                    </div>
                  </CommandItem>
                ))}
              </CommandGroup>
            ))}
          </CommandList>
        </Command>
      </CommandDialog>
    </>
  );
}

function groupBySource(results: SearchResult[]) {
  const map = new Map<string, SearchResult[]>();
  for (const r of results) {
    const arr = map.get(r.source.name) ?? [];
    arr.push(r);
    map.set(r.source.name, arr);
  }
  return Array.from(map, ([source, items]) => ({ source, items }));
}
