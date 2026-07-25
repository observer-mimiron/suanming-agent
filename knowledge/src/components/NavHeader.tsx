"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { GlobalSearch } from "./GlobalSearch";
import { ThemeToggle } from "./folio/ThemeToggle";

const links = [
  { href: "/wiki", label: "文档" },
  { href: "/wiki/graph", label: "图谱" },
  { href: "/query", label: "问答" },
];

export function NavHeader() {
  const pathname = usePathname();

  return (
    <header className="sticky top-0 z-50 border-b bg-white/80 dark:bg-zinc-950/80 backdrop-blur">
      <div className="max-w-4xl mx-auto flex items-center gap-3 px-4 h-12">
        <Link href="/" className="font-bold text-sm shrink-0">
          命理大师
        </Link>
        <nav className="flex gap-1">
          {links.map((l) => (
            <Link
              key={l.href}
              href={l.href}
              className={`px-2 py-1 text-xs rounded ${
                pathname.startsWith(l.href)
                  ? "bg-zinc-100 dark:bg-zinc-800"
                  : "hover:bg-zinc-50 dark:hover:bg-zinc-900"
              }`}
            >
              {l.label}
            </Link>
          ))}
        </nav>
        <div className="flex-1" />
        <GlobalSearch />
        <ThemeToggle />
      </div>
    </header>
  );
}
