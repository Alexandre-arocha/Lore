"use client";

import { useEffect, useState } from "react";

import type { DocumentPage } from "@/lib/api";

type Toc = DocumentPage["toc"];

export function TableOfContents({ toc }: { toc: Toc }) {
  const [active, setActive] = useState<string>("");

  useEffect(() => {
    const anchors = toc.flatMap((h) => [
      h.anchor,
      ...(h.children?.map((c) => c.anchor) ?? []),
    ]);
    const els = anchors
      .map((a) => document.getElementById(a))
      .filter((el): el is HTMLElement => el !== null);

    if (els.length === 0) {
      return;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((e) => e.isIntersecting)
          .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top);
        if (visible[0]) {
          setActive(visible[0].target.id);
        }
      },
      { rootMargin: "-80px 0px -70% 0px", threshold: 0 },
    );

    els.forEach((el) => observer.observe(el));
    return () => observer.disconnect();
  }, [toc]);

  if (!toc || toc.length === 0) {
    return null;
  }

  return (
    <aside className="hidden py-8 xl:block xl:border-l xl:border-border xl:pl-6">
      <div className="sticky top-[4.5rem]">
        <p className="label mb-3">nesta página</p>
        <nav className="border-l border-border">
          {toc.map((h) => (
            <div key={h.anchor}>
              <TocLink anchor={h.anchor} title={h.title} indent="pl-3" active={active === h.anchor} />
              {h.children?.map((c) => (
                <TocLink
                  key={c.anchor}
                  anchor={c.anchor}
                  title={c.title}
                  indent="pl-6"
                  active={active === c.anchor}
                />
              ))}
            </div>
          ))}
        </nav>
      </div>
    </aside>
  );
}

function TocLink({
  anchor,
  title,
  indent,
  active,
}: {
  anchor: string;
  title: string;
  indent: "pl-3" | "pl-6";
  active: boolean;
}) {
  return (
    <a
      href={`#${anchor}`}
      aria-current={active ? "location" : undefined}
      className={`-ml-px block border-l-2 py-1 font-mono text-xs ${indent} transition-colors ${
        active
          ? "border-l-gold text-gold"
          : "border-transparent text-muted-foreground hover:border-l-line-strong hover:text-foreground"
      }`}
    >
      {title}
    </a>
  );
}
