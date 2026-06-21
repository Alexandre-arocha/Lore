import Link from "next/link";

import { CommandPalette } from "@/components/command-palette";

export function SiteHeader() {
  return (
    <header className="sticky top-0 z-40 border-b border-border bg-background/90 backdrop-blur">
      <div className="mx-auto flex h-14 w-full max-w-7xl items-center gap-4 px-5 sm:px-6 lg:px-8">
        <Link
          href="/"
          className="group flex items-center gap-2.5"
          aria-label="Lore — início"
        >
          <span className="size-3.5 bg-gold" aria-hidden />
          <span className="font-mono text-[15px] font-medium tracking-tight text-foreground">
            lore
          </span>
        </Link>
        <div className="ml-auto flex items-center gap-2">
          <CommandPalette />
        </div>
      </div>
    </header>
  );
}
