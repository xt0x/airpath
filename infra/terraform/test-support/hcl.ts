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
  return scanBlocks(source, kind).filter((block) => block.labels.length > 0);
}

export function unlabeledHclBlocks(source: string, kind: string): HclBlock[] {
  return scanBlocks(source, kind).filter((block) => block.labels.length === 0);
}

function scanBlocks(source: string, kind: string): HclBlock[] {
  const blocks: HclBlock[] = [];
  let index = 0;
  while (index < source.length) {
    const lineStart = index === 0 || source[index - 1] === "\n";
    const headerStart = lineStart ? skipLineWhitespace(source, index) : index;
    if (!lineStart || !startsWithWord(source, headerStart, kind)) {
      index += 1;
      continue;
    }
    const header = readBlockHeader(source, headerStart + kind.length);
    if (header === null) {
      index += 1;
      continue;
    }
    const openBrace = header.openBrace;
    const closeBrace = matchingBraceIndex(source, openBrace);
    blocks.push({
      kind,
      labels: header.labels,
      body: source.slice(openBrace + 1, closeBrace),
    });
    index = closeBrace + 1;
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
  const value = attributeValue(block.body, name);
  if (value === null) {
    return false;
  }
  return valuePattern === undefined || valuePattern.test(value);
}

export function bodyIncludes(block: HclBlock, expected: string): boolean {
  return (
    block.body.includes(expected) ||
    normalizeHclText(block.body).includes(normalizeHclText(expected))
  );
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
    if (!inString && source.startsWith("<<", index)) {
      index = skipHeredoc(source, index);
      continue;
    }
    if (!inString && source.startsWith("//", index)) {
      index = skipLineComment(source, index);
      continue;
    }
    if (!inString && source.startsWith("#", index)) {
      index = skipLineComment(source, index);
      continue;
    }
    if (!inString && source.startsWith("/*", index)) {
      index = skipBlockComment(source, index);
      continue;
    }
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

function attributeValue(source: string, name: string): string | null {
  let index = 0;
  while (index < source.length) {
    const lineStart = index === 0 || source[index - 1] === "\n";
    const nameStart = lineStart ? skipLineWhitespace(source, index) : index;
    if (!lineStart || !startsWithWord(source, nameStart, name)) {
      index += 1;
      continue;
    }
    let cursor = skipLineWhitespace(source, nameStart + name.length);
    if (source[cursor] !== "=") {
      index += 1;
      continue;
    }
    cursor = skipLineWhitespace(source, cursor + 1);
    const end = attributeValueEnd(source, cursor);
    return source.slice(cursor, end).trim();
  }
  return null;
}

function attributeValueEnd(source: string, start: number): number {
  let depth = 0;
  let inString = false;
  let escaped = false;
  for (let index = start; index < source.length; index += 1) {
    if (!inString && source.startsWith("<<", index)) {
      return skipHeredoc(source, index) + 1;
    }
    if (!inString && source.startsWith("//", index)) {
      index = skipLineComment(source, index);
      if (depth === 0) {
        return index;
      }
      continue;
    }
    if (!inString && source.startsWith("#", index)) {
      index = skipLineComment(source, index);
      if (depth === 0) {
        return index;
      }
      continue;
    }
    if (!inString && source.startsWith("/*", index)) {
      index = skipBlockComment(source, index);
      continue;
    }
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
    if (character === "[" || character === "{" || character === "(") {
      depth += 1;
      continue;
    }
    if (character === "]" || character === "}" || character === ")") {
      depth -= 1;
      continue;
    }
    if (character === "\n" && depth === 0) {
      return index;
    }
  }
  return source.length;
}

function skipHeredoc(source: string, start: number): number {
  const headerEnd = source.indexOf("\n", start);
  if (headerEnd === -1) {
    return source.length - 1;
  }
  const header = source.slice(start + 2, headerEnd).trim();
  const marker = header.replace(/^-/, "").trim();
  if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(marker)) {
    return start;
  }
  const endPattern = new RegExp(`(^|\\n)[ \\t]*${escapeRegExp(marker)}[ \\t]*(\\n|$)`, "g");
  endPattern.lastIndex = headerEnd + 1;
  const match = endPattern.exec(source);
  if (match === null) {
    return source.length - 1;
  }
  return match.index + match[0].length - 1;
}

function skipLineComment(source: string, start: number): number {
  const newline = source.indexOf("\n", start);
  return newline === -1 ? source.length - 1 : newline;
}

function skipBlockComment(source: string, start: number): number {
  const end = source.indexOf("*/", start + 2);
  return end === -1 ? source.length - 1 : end + 1;
}

function normalizeHclText(value: string): string {
  return value.replace(/\s+/g, " ").trim();
}

function startsWithWord(source: string, index: number, word: string): boolean {
  if (!source.startsWith(word, index)) {
    return false;
  }
  const before = source[index - 1];
  const after = source[index + word.length];
  return (
    (before === undefined || /[\s{]/.test(before)) && (after === undefined || /[\s"{]/.test(after))
  );
}

function readBlockHeader(
  source: string,
  index: number,
): { labels: string[]; openBrace: number } | null {
  const labels: string[] = [];
  let cursor = skipWhitespace(source, index);
  while (source[cursor] === '"') {
    const end = source.indexOf('"', cursor + 1);
    if (end === -1) {
      return null;
    }
    labels.push(source.slice(cursor + 1, end));
    cursor = skipWhitespace(source, end + 1);
  }
  return source[cursor] === "{" ? { labels, openBrace: cursor } : null;
}

function skipWhitespace(source: string, index: number): number {
  let cursor = index;
  while (
    source[cursor] === " " ||
    source[cursor] === "\t" ||
    source[cursor] === "\r" ||
    source[cursor] === "\n"
  ) {
    cursor += 1;
  }
  return cursor;
}

function skipLineWhitespace(source: string, index: number): number {
  let cursor = index;
  while (source[cursor] === " " || source[cursor] === "\t") {
    cursor += 1;
  }
  return cursor;
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
