import { apiSchemas } from "./api-schemas.js";

export type ApiSchemaName = keyof typeof apiSchemas;

export type ApiSchemaValidationResult =
  | { valid: true }
  | { valid: false; path: string; message: string };

type JsonSchema = Record<string, unknown>;

export function validateApiSchema(
  schemaName: ApiSchemaName,
  value: unknown,
): ApiSchemaValidationResult {
  return validateAgainstSchema(apiSchemas[schemaName] as JsonSchema, value, "$");
}

function validateAgainstSchema(
  schema: JsonSchema,
  value: unknown,
  path: string,
): ApiSchemaValidationResult {
  const resolvedSchema = resolveSchema(schema);

  const combinedSchemaResult = validateCombinedSchemas(resolvedSchema, value, path);
  if (combinedSchemaResult !== null) {
    return combinedSchemaResult;
  }

  if ("const" in resolvedSchema && !Object.is(value, resolvedSchema.const)) {
    return invalid(path, `must equal ${String(resolvedSchema.const)}`);
  }

  if (
    Array.isArray(resolvedSchema.enum) &&
    !resolvedSchema.enum.some((item) => Object.is(item, value))
  ) {
    return invalid(path, "must be one of the allowed values");
  }

  const typeResult = validateType(resolvedSchema.type, value, path);
  if (!typeResult.valid) {
    return typeResult;
  }

  if (typeof value === "number") {
    const rangeResult = validateNumberRange(resolvedSchema, value, path);
    if (!rangeResult.valid) {
      return rangeResult;
    }
  }

  if (typeof value === "string") {
    const stringResult = validateString(resolvedSchema, value, path);
    if (!stringResult.valid) {
      return stringResult;
    }
  }

  if (Array.isArray(value)) {
    return validateArray(resolvedSchema, value, path);
  }

  if (isPlainObject(value)) {
    return validateObject(resolvedSchema, value, path);
  }

  return { valid: true };
}

function validateCombinedSchemas(
  schema: JsonSchema,
  value: unknown,
  path: string,
): ApiSchemaValidationResult | null {
  if (Array.isArray(schema.anyOf)) {
    const matches = schema.anyOf.filter(
      (candidate) => validateAgainstSchema(candidate as JsonSchema, value, path).valid,
    );
    return matches.length > 0 ? { valid: true } : invalid(path, "must match at least one schema");
  }

  if (Array.isArray(schema.oneOf)) {
    const matches = schema.oneOf.filter(
      (candidate) => validateAgainstSchema(candidate as JsonSchema, value, path).valid,
    );
    return matches.length === 1 ? { valid: true } : invalid(path, "must match exactly one schema");
  }

  return null;
}

function validateType(type: unknown, value: unknown, path: string): ApiSchemaValidationResult {
  if (type === undefined) {
    return { valid: true };
  }

  const allowedTypes = Array.isArray(type) ? type : [type];
  const matched = allowedTypes.some((allowedType) => matchesType(allowedType, value));
  return matched ? { valid: true } : invalid(path, `must be ${allowedTypes.join(" or ")}`);
}

function matchesType(type: unknown, value: unknown): boolean {
  switch (type) {
    case "array":
      return Array.isArray(value);
    case "boolean":
      return typeof value === "boolean";
    case "integer":
      return typeof value === "number" && Number.isInteger(value);
    case "null":
      return value === null;
    case "number":
      return typeof value === "number" && Number.isFinite(value);
    case "object":
      return isPlainObject(value);
    case "string":
      return typeof value === "string";
    default:
      return false;
  }
}

function validateNumberRange(
  schema: JsonSchema,
  value: number,
  path: string,
): ApiSchemaValidationResult {
  if (typeof schema.minimum === "number" && value < schema.minimum) {
    return invalid(path, `must be greater than or equal to ${schema.minimum}`);
  }
  if (typeof schema.maximum === "number" && value > schema.maximum) {
    return invalid(path, `must be less than or equal to ${schema.maximum}`);
  }
  return { valid: true };
}

function validateString(
  schema: JsonSchema,
  value: string,
  path: string,
): ApiSchemaValidationResult {
  if (typeof schema.minLength === "number" && value.length < schema.minLength) {
    return invalid(path, `must have at least ${schema.minLength} characters`);
  }
  if (typeof schema.maxLength === "number" && value.length > schema.maxLength) {
    return invalid(path, `must have at most ${schema.maxLength} characters`);
  }
  if (typeof schema.pattern === "string" && !new RegExp(schema.pattern).test(value)) {
    return invalid(path, `must match pattern ${schema.pattern}`);
  }
  return { valid: true };
}

function validateArray(
  schema: JsonSchema,
  value: unknown[],
  path: string,
): ApiSchemaValidationResult {
  if (typeof schema.minItems === "number" && value.length < schema.minItems) {
    return invalid(path, `must contain at least ${schema.minItems} items`);
  }

  if (schema.uniqueItems === true && new Set(value).size !== value.length) {
    return invalid(path, "must contain unique items");
  }

  if (isPlainObject(schema.items)) {
    for (const [index, item] of value.entries()) {
      const result = validateAgainstSchema(schema.items, item, `${path}[${index}]`);
      if (!result.valid) {
        return result;
      }
    }
  }

  return { valid: true };
}

function validateObject(
  schema: JsonSchema,
  value: Record<string, unknown>,
  path: string,
): ApiSchemaValidationResult {
  const properties = isPlainObject(schema.properties) ? schema.properties : {};
  const required = Array.isArray(schema.required) ? schema.required : [];

  for (const propertyName of required) {
    if (typeof propertyName === "string" && !(propertyName in value)) {
      return invalid(`${path}.${propertyName}`, "is required");
    }
  }

  if (schema.additionalProperties === false) {
    const allowedProperties = new Set(Object.keys(properties));
    const unknownProperty = Object.keys(value).find(
      (propertyName) => !allowedProperties.has(propertyName),
    );
    if (unknownProperty !== undefined) {
      return invalid(`${path}.${unknownProperty}`, "is not allowed");
    }
  }

  for (const [propertyName, propertySchema] of Object.entries(properties)) {
    if (!(propertyName in value)) {
      continue;
    }
    const result = validateAgainstSchema(
      propertySchema as JsonSchema,
      value[propertyName],
      `${path}.${propertyName}`,
    );
    if (!result.valid) {
      return result;
    }
  }

  return { valid: true };
}

function resolveSchema(schema: JsonSchema): JsonSchema {
  if (typeof schema.$ref !== "string") {
    return schema;
  }

  const schemaName = schema.$ref.replace("#/components/schemas/", "");
  if (!(schemaName in apiSchemas)) {
    return schema;
  }

  return apiSchemas[schemaName as ApiSchemaName] as JsonSchema;
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function invalid(path: string, message: string): ApiSchemaValidationResult {
  return { valid: false, path, message };
}
