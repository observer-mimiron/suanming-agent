import type { Metadata } from "next";
import { Inter, Source_Serif_4, JetBrains_Mono } from "next/font/google";
import { NavHeader } from "@/components/NavHeader";
import { ClientProviders } from "@/components/ClientProviders";
import "./globals.css";

const fontSans = Inter({
  subsets: ["latin"],
  variable: "--font-sans-next",
  display: "swap",
});
const fontSerif = Source_Serif_4({
  subsets: ["latin"],
  variable: "--font-serif-next",
  display: "swap",
});
const fontMono = JetBrains_Mono({
  subsets: ["latin"],
  variable: "--font-mono-next",
  display: "swap",
});

export const metadata: Metadata = {
  title: {
    default: "命理大师 — 八字命理知识库",
    template: "%s · 命理大师",
  },
  description: "八字命理（四柱推命）参考资料知识库",
};

const themeScript = `
(function() {
  try {
    var t = localStorage.getItem('theme');
    if (t === 'dark') {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.add('light');
    }
  } catch(e) {}
})();
`;

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="zh-CN"
      suppressHydrationWarning
      className={`${fontSans.variable} ${fontSerif.variable} ${fontMono.variable}`}
    >
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeScript }} />
      </head>
      <body className="min-h-screen antialiased">
        <ClientProviders>
          <NavHeader />
          <main className="max-w-4xl mx-auto px-4 py-6">{children}</main>
        </ClientProviders>
      </body>
    </html>
  );
}
