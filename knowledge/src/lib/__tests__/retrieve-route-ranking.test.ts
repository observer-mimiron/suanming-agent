import {describe, expect, it} from "vitest";

import {isSubstantiveBaziChapter, mergeRetrievalResults,} from "@/app/api/wiki/retrieve/route";

describe("runtime retrieval ranking", () => {
  it("keeps only chapter passages when a classical source is specified", () => {
    const results = mergeRetrievalResults(
      "穷通宝鉴 丙火 亥月 调候",
      [
        { slug: "ref-bazi-qiongtong", title: "穷通宝鉴", summary: "书籍入口", snippet: "书籍入口", score: 9 },
        { slug: "ref-bazi-qiongtong-s001", title: "五行总论", summary: "原文", snippet: "可引用原文", quote: "五行贵在折衷，归于中道。", score: 8 },
        { slug: "ref-bazi-ziping-s007", title: "论十干合而不合", summary: "原文", snippet: "可引用原文", score: 10 },
      ],
      [],
      3,
    );

    expect(results.map((result) => result.slug)).toEqual(["ref-bazi-qiongtong-s001"]);
    expect(results[0].quote).toBe("五行贵在折衷，归于中道。");
  });

  it("recognizes only numbered bazi chapters as substantive", () => {
    expect(isSubstantiveBaziChapter("ref-bazi-ziping-s047")).toBe(true);
    expect(isSubstantiveBaziChapter("ref-bazi-ziping")).toBe(false);
    expect(isSubstantiveBaziChapter("ref-bazi-ziping-part1")).toBe(false);
  });
});
