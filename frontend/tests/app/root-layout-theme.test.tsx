import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

describe("RootLayout theme", () => {
  it("enables the shadcn default dark color tokens at the document root", () => {
    const source = readFileSync("src/app/layout.tsx", "utf8");

    expect(source).toContain('"dark font-sans"');
    expect(source).toContain("oklch(0.145 0 0)");
    expect(source).not.toContain("rgb(30 30 30)");
  });

  it("declares dark browser chrome for the shadcn dark theme", () => {
    const css = readFileSync("src/app/globals.css", "utf8");

    expect(css).toMatch(/\.dark\s*\{[^}]*color-scheme:\s*dark;/s);
  });
});
