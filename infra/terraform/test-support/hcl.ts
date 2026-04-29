import { readFileSync } from "node:fs";
import { join } from "node:path";

export interface HclBlock {
  kind: string;
  labels: string[];
  body: string;
}

export const readRepoFile = (path: string): string => readFileSync(path, "utf8");

export const readTerraformFile = (path: string): string =>
  readRepoFile(join("infra/terraform", path));

export function hclBlocks(source: string, kind: string): HclBlock[] {
  const blocks: HclBlock[] = [];
  const headerPattern = new RegExp(`(^|\\n)\\s*${kind}\\s+((?:"[^"]+"\\s*)+)\\{`, "g");
  let match: RegExpExecArray | null;
  while ((match = headerPattern.exec(source)) !== null) {
    const openBrace = headerPattern.lastIndex - 1;
    const closeBrace = matchingBraceIndex(source, openBrace);
    blocks.push({
      kind,
      labels: [...(match[2] ?? "").matchAll(/"([^"]+)"/g)].map((label) => label[1] ?? ""),
      body: source.slice(openBrace + 1, closeBrace),
    });
    headerPattern.lastIndex = closeBrace + 1;
  }
  return blocks;
}

export function unlabeledHclBlocks(source: string, kind: string): HclBlock[] {
  const blocks: HclBlock[] = [];
  const headerPattern = new RegExp(`(^|\\n)\\s*${kind}\\s*\\{`, "g");
  let match: RegExpExecArray | null;
  while ((match = headerPattern.exec(source)) !== null) {
    const openBrace = headerPattern.lastIndex - 1;
    const closeBrace = matchingBraceIndex(source, openBrace);
    blocks.push({
      kind,
      labels: [],
      body: source.slice(openBrace + 1, closeBrace),
    });
    headerPattern.lastIndex = closeBrace + 1;
  }
  return blocks;
}

export function resourceBlock(source: string, type: string, name: string): HclBlock {
  return requiredBlock(source, "resource", [type, name]);
}

export function localsBlock(source: string): HclBlock {
  const block = unlabeledHclBlocks(source, "locals")[0];
  if (block === undefined) {
    throw new Error("locals block not found");
  }
  return block;
}

export function dataBlock(source: string, type: string, name: string): HclBlock {
  return requiredBlock(source, "data", [type, name]);
}

export function moduleBlock(source: string, name: string): HclBlock {
  return requiredBlock(source, "module", [name]);
}

export function outputBlock(source: string, name: string): HclBlock {
  return requiredBlock(source, "output", [name]);
}

export function variableBlock(source: string, name: string): HclBlock {
  return requiredBlock(source, "variable", [name]);
}

export function hasAttribute(block: HclBlock, name: string, valuePattern?: RegExp): boolean {
  const pattern = new RegExp(`(^|\\n)\\s*${escapeRegExp(name)}\\s*=\\s*([^\\n]+)`);
  const match = pattern.exec(block.body);
  if (match === null) {
    return false;
  }
  return valuePattern === undefined || valuePattern.test(match[2] ?? "");
}

export function bodyIncludes(block: HclBlock, expected: string): boolean {
  return block.body.includes(expected);
}

function requiredBlock(source: string, kind: string, labels: string[]): HclBlock {
  const block = hclBlocks(source, kind).find((candidate) =>
    labels.every((label, index) => candidate.labels[index] === label),
  );
  if (block === undefined) {
    throw new Error(`${kind} ${labels.map((label) => `"${label}"`).join(" ")} not found`);
  }
  return block;
}

function matchingBraceIndex(source: string, openBrace: number): number {
  let depth = 0;
  let inString = false;
  let escaped = false;
  for (let index = openBrace; index < source.length; index += 1) {
    const character = source[index];
    if (inString) {
      escaped = character === "\\" && !escaped;
      if (character === '"' && !escaped) {
        inString = false;
      }
      if (character !== "\\") {
        escaped = false;
      }
      continue;
    }
    if (character === '"') {
      inString = true;
      continue;
    }
    if (character === "{") {
      depth += 1;
      continue;
    }
    if (character === "}") {
      depth -= 1;
      if (depth === 0) {
        return index;
      }
    }
  }
  throw new Error("unclosed HCL block");
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
