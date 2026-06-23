import Link from "next/link";

export default function NotFound() {
  return (
    <main className="mx-auto flex min-h-[60svh] w-full max-w-7xl flex-col items-start justify-center px-5 sm:px-6 lg:px-8">
      <p className="label mb-3">404</p>
      <h1 className="text-4xl font-semibold tracking-tight">
        Página não encontrada
      </h1>
      <p className="mt-4 max-w-md text-muted-foreground">
        Este endereço não existe no acervo. Pode ter mudado de lugar, ou a fonte
        ainda não foi sincronizada.
      </p>
      <Link
        href="/"
        className="mt-6 inline-flex border border-border px-3 py-2 font-mono text-xs transition-colors hover:border-line-strong hover:text-gold"
      >
        ← voltar ao acervo
      </Link>
    </main>
  );
}
