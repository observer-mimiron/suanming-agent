import { describe, it, expect } from "vitest";
import { listWikiPages } from "@/lib/wiki";

describe("smoke test", () => {
  it("should verify vitest works", () => {
    expect(1 + 1).toBe(2);
  });

  it("listWikiPages returns an array with expected shape", async () => {
    const pages = await listWikiPages();
    expect(Array.isArray(pages)).toBe(true);
    for (const entry of pages) {
      expect(typeof entry.slug).toBe("string");
      expect(typeof entry.title).toBe("string");
      expect(typeof entry.summary).toBe("string");
    }
  });
});
