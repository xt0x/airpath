import { renderToStaticMarkup } from "react-dom/server";
import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

import UsagePage from "@/app/(app)/usage/page";
import { TooltipProvider } from "@/components/ui/tooltip";

const GLOBAL_CSS_PATH = "src/app/globals.css";
const USAGE_CSS_PATH = "src/features/usage/components/usage-workspace.module.css";

describe("usage route", () => {
  it("renders the app sidebar shell with the Usage & Limits screen", () => {
    const html = renderToStaticMarkup(
      <TooltipProvider>
        <UsagePage />
      </TooltipProvider>,
    );

    expect(html).toContain('data-slot="sidebar-wrapper"');
    expect(html).toContain('data-slot="sidebar"');
    expect(html).toContain('data-slot="sidebar-inset"');
    expect(html).toContain('data-slot="sidebar-trigger"');
    expect(html).toContain("Usage &amp; Limits");
    expect(html).not.toContain("Loading usage status");
    expect(html).toContain('href="/usage"');
    expect(html).toContain('data-active="true"');
  });

  it("keeps the usage inset as the scroll container because the app body is fixed for the map", () => {
    const globalCss = readFileSync(GLOBAL_CSS_PATH, "utf8");
    const css = cssModuleGlobalView(readFileSync(USAGE_CSS_PATH, "utf8"));

    expect(globalCss).toMatch(/body\s*\{[^}]*overflow:\s*hidden;/s);
    expect(css).toMatch(
      /\.usage-workspace__inset\s*\{[^}]*height:\s*100vh;[^}]*height:\s*100dvh;[^}]*overflow-y:\s*auto;/s,
    );
    expect(css).toMatch(/\.usage-workspace__inset\s*\{[^}]*min-height:\s*0;/s);
  });

  it("positions the usage content below the fixed toolbar with deliberate page spacing", () => {
    const css = cssModuleGlobalView(readFileSync(USAGE_CSS_PATH, "utf8"));

    expect(css).toMatch(/\.usage-limits\s*\{[^}]*padding:\s*4\.5rem 0 3rem;/s);
    expect(css).toMatch(
      /@media \(max-width:\s*40rem\)\s*\{[^}]*\.usage-limits\s*\{[^}]*padding-top:\s*4rem;/s,
    );
  });

  it("keeps usage-specific CSS out of the global stylesheet", () => {
    const globalCss = readFileSync(GLOBAL_CSS_PATH, "utf8");
    const usageCss = cssModuleGlobalView(readFileSync(USAGE_CSS_PATH, "utf8"));

    expect(globalCss).not.toContain(".usage-limits");
    expect(globalCss).not.toContain(".usage-workspace__inset");
    expect(globalCss).not.toContain(".usage-sidebar-layout");
    expect(usageCss).toContain(".usage-limits");
    expect(usageCss).toContain(".usage-workspace__inset");
    expect(usageCss).toContain(".usage-sidebar-layout");
  });
});

function cssModuleGlobalView(css: string): string {
  let output = "";
  for (let index = 0; index < css.length; index += 1) {
    if (!css.startsWith(":global(", index)) {
      output += css[index];
      continue;
    }

    index += ":global(".length;
    let depth = 1;
    while (index < css.length && depth > 0) {
      const char = css[index];
      if (char === "(") {
        depth += 1;
        output += char;
      } else if (char === ")") {
        depth -= 1;
        if (depth > 0) {
          output += char;
        }
      } else {
        output += char;
      }
      index += 1;
    }
    index -= 1;
  }
  return output;
}
