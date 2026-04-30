import { readdirSync, statSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

function sourceTestFiles(directory: string): string[] {
  return readdirSync(directory).flatMap((entry) => {
    const path = join(directory, entry);
    if (statSync(path).isDirectory()) {
      return sourceTestFiles(path);
    }
    return entry.endsWith(".test.ts") || entry.endsWith(".spec.ts") ? [path] : [];
  });
}

describe("shared-types source layout", () => {
  it("keeps unit tests outside src so production source stays scannable", () => {
    expect(sourceTestFiles("src")).toEqual([]);
  });
});
