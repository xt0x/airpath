import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import Loading from "@/app/(app)/loading";

describe("reload progress", () => {
  it("uses the documented Progress component for the app loading state", () => {
    const html = renderToStaticMarkup(<Loading />);

    expect(html).toContain('role="status"');
    expect(html).toContain('aria-label="Loading map workspace"');
    expect(html).toContain('data-slot="progress"');
    expect(html).toContain('data-slot="progress-indicator"');
    expect(html).toContain("w-[60%]");
    expect(html).toContain("translateX(-34%)");
  });
});
