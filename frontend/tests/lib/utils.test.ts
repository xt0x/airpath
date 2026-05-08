import { describe, expect, it } from "vitest";

import { cn } from "@/lib/utils";

describe("class name utilities", () => {
  it("combines conditional class values", () => {
    const result = cn("flex", false && "hidden", ["items-center", { "gap-2": true }]);

    expect(result.split(" ")).toEqual(expect.arrayContaining(["flex", "items-center", "gap-2"]));
    expect(result).not.toContain("hidden");
  });

  it("lets later Tailwind classes resolve conflicts", () => {
    const result = cn("rounded-md px-2 text-sm", "px-4", { "rounded-none": true });

    expect(result).toContain("px-4");
    expect(result).toContain("rounded-none");
    expect(result).toContain("text-sm");
    expect(result).not.toContain("px-2");
    expect(result).not.toContain("rounded-md");
  });
});
