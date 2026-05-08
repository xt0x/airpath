import { existsSync, readFileSync, readdirSync, statSync } from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

const frontendRoot = path.resolve(__dirname, "../..");
const sourceRoot = path.join(frontendRoot, "src");
const checkedRoots = [sourceRoot, path.join(frontendRoot, "tests")];
const sourceExtensions = [".ts", ".tsx", ".js", ".jsx"];
const importSpecifierPattern =
  /\b(?:import|export)\s+(?:type\s+)?(?:[^"'()]*?\s+from\s+)?["']([^"']+)["']|\bimport\s*\(\s*["']([^"']+)["']\s*\)/g;

function listSourceFiles(directory: string): string[] {
  return readdirSync(directory).flatMap((entry) => {
    const entryPath = path.join(directory, entry);
    const stats = statSync(entryPath);

    if (stats.isDirectory()) {
      return listSourceFiles(entryPath);
    }

    return sourceExtensions.includes(path.extname(entryPath)) ? [entryPath] : [];
  });
}

function resolvesInsideSource(importerPath: string, specifier: string): boolean {
  if (!specifier.startsWith(".")) {
    return false;
  }

  const absoluteBase = path.resolve(path.dirname(importerPath), specifier);
  const candidates = [
    absoluteBase,
    ...sourceExtensions.map((extension) => `${absoluteBase}${extension}`),
    ...sourceExtensions.map((extension) => path.join(absoluteBase, `index${extension}`)),
  ];

  return candidates.some((candidate) => {
    const relativeToSource = path.relative(sourceRoot, candidate);
    return (
      existsSync(candidate) &&
      !relativeToSource.startsWith("..") &&
      !path.isAbsolute(relativeToSource)
    );
  });
}

describe("frontend import aliases", () => {
  it("uses the @ alias for imports that target frontend src", () => {
    const violations = checkedRoots.flatMap(listSourceFiles).flatMap((filePath) => {
      const source = readFileSync(filePath, "utf8");
      return [...source.matchAll(importSpecifierPattern)]
        .map((match) => match[1] ?? match[2] ?? "")
        .filter((specifier) => resolvesInsideSource(filePath, specifier))
        .map((specifier) => `${path.relative(frontendRoot, filePath)} -> ${specifier}`);
    });

    expect(violations).toEqual([]);
  });
});
