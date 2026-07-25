import Link from "next/link";
import { listCommonsPages } from "@/lib/commons";

export const dynamic = "force-dynamic";

export default async function Home() {
  const pages = await listCommonsPages();
  const pageCount = pages.length;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-lg font-bold">八字命理知识库</h1>
        <span className="text-xs text-zinc-500">{pageCount} 个页面</span>
      </div>

      <div className="grid gap-2">
        {pages.map((p) => (
          <Link
            key={p.slug}
            href={`/wiki/${p.slug}`}
            className="flex items-center gap-3 px-3 py-2 rounded-lg border border-zinc-200 dark:border-zinc-800 hover:bg-zinc-50 dark:hover:bg-zinc-900 transition-colors"
          >
            <span className="text-sm font-medium truncate">{p.title}</span>
            <span className="text-xs text-zinc-400 shrink-0">{p.slug}</span>
            {p.tags && p.tags.length > 0 && (
              <span className="flex gap-1 ml-auto">
                {p.tags.slice(0, 3).map((t: string) => (
                  <span key={t} className="text-[10px] px-1.5 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-500">{t}</span>
                ))}
              </span>
            )}
          </Link>
        ))}
      </div>

      {pageCount === 0 && (
        <div className="text-center py-16 text-zinc-400">
          <p className="text-sm">知识库为空</p>
          <p className="text-xs mt-1">运行 make knowledge-import 导入资料</p>
        </div>
      )}
    </div>
  );
}
