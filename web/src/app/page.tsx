import { getHealth } from "@/lib/api";

export default async function Home() {
  const health = await getHealth();
  const online = health?.status === "ok";

  return (
    <main className="mx-auto flex min-h-svh w-full max-w-2xl flex-col justify-center gap-8 px-6 py-16">
      <div className="space-y-3">
        <h1 className="text-5xl font-bold tracking-tight">Atlas</h1>
        <p className="text-lg text-muted-foreground">
          A documentação de várias linguagens e frameworks num só lugar — busca
          unificada, navegação rápida e leitura limpa.
        </p>
      </div>

      <div className="rounded-lg border p-4">
        <div className="flex items-center gap-3">
          <span
            className={`inline-block size-2.5 rounded-full ${
              online ? "bg-green-500" : "bg-red-500"
            }`}
            aria-hidden
          />
          <span className="text-sm font-medium">
            API:{" "}
            {health
              ? `${health.status} (db: ${health.db})`
              : "inacessível — rode a API em http://localhost:8080"}
          </span>
        </div>
      </div>

      <p className="text-sm text-muted-foreground">
        Setup (Fase 0) concluído. As próximas fases adicionam schema, ingestão,
        busca e o leitor de documentação.
      </p>
    </main>
  );
}
