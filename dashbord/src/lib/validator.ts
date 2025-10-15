// deno-lint-ignore-file no-explicit-any

type ValidatorFailure<T extends Def> = {
  type: T["type"];
  path: (string | number)[];
  value: unknown;
};

type Validator<T extends Def> = (
  value: unknown,
  path?: (string | number)[],
) => ValidatorFailure<T>[];

type DefArray<T extends Def> = {
  type: "array";
  of: Def;
  report: Validator<T>;
  optional?: boolean;
  description?: string;
  assert: (value: unknown) => ReturnType<T["assert"]>[];
};

type DefList<T extends readonly (string | number)[]> = {
  type: "list";
  of: T;
  report: Validator<DefList<T>>;
  optional?: boolean;
  description?: string;
  assert: (value: unknown) => T[number];
};

type DefUnion<T extends readonly Def[]> = {
  type: "union";
  of: T;
  report: Validator<DefUnion<T>>;
  optional?: boolean;
  description?: string;
  assert: (value: unknown) => ReturnType<T[number]["assert"]>;
};

type DefObject<T extends Record<string, Def>> = {
  type: "object";
  properties: { [K in keyof T]: T[K] };
  report: Validator<T[keyof T]>;
  optional?: boolean;
  description?: string;
  assert: (value: unknown) => { [K in keyof T]: ReturnType<T[K]["assert"]> };
};

type DefString = {
  type: "string";
  assert: AssertType<string>;
  report: Validator<DefString>;
  optional?: boolean;
  description?: string;
};

type DefNumber = {
  type: "number";
  assert: AssertType<number>;
  report: Validator<DefNumber>;
  optional?: boolean;
  description?: string;
};

type DefBoolean = {
  type: "boolean";
  assert: AssertType<boolean>;
  report: Validator<DefBoolean>;
  optional?: boolean;
  description?: string;
};

export type DefBase =
  | DefString
  | DefNumber
  | DefBoolean
  | DefArray<any>
  | DefObject<Record<string, any>>
  | DefList<any>
  | DefUnion<any>;

type OptionalAssert<T extends Def["assert"]> = (
  value: unknown,
) => ReturnType<T> | undefined;

type Optional<T extends Def> = T & {
  assert: OptionalAssert<T["assert"]>;
};

type AssertType<T> = (value: unknown) => T;

export type Def<T = unknown> = T extends DefBase ? DefArray<T>
  : T extends Record<string, DefBase> ? DefObject<T>
  : DefBase;

const reportObject = <T extends Record<string, Def>>(properties: T) => {
  const body = [
    'if (!o || typeof o !== "object") return [{ path: p, type: "object", value: o }]',
    "const failures = []",
    ...Object.entries(properties).map(([key, def], i) => {
      const k = JSON.stringify(key);
      const path = `[...p, ${k}]`;
      if (def.type === "object" || def.type === "array") {
        const check = `
            const _${i} = v[${k}].report(o[${k}], ${path});
            _${i}.length && failures.push(..._${i})
          `;
        return def.optional ? `if (o[${k}] !== undefined) {${check}}` : check;
      }
      const opt = def.optional ? `o[${k}] === undefined || ` : "";
      return (`${opt}typeof o[${k}] === "${def.type}" || failures.push({ ${
        [`path: ${path}`, `type: "${def.type}"`, `value: o[${k}]`].join(", ")
      } })`);
    }),
    "return failures",
  ].join("\n");

  return new Function("v, o, p = []", body).bind(
    globalThis,
    properties,
  ) as DefObject<T>["report"];
};

const assertObject = <T extends Record<string, Def>>(properties: T) => {
  const body = [
    'if (!o || typeof o !== "object") throw Error("type assertion failed")',
    ...Object.entries(properties).map(([key, def]) => {
      const k = JSON.stringify(key);
      return `${
        def.optional ? `v[${k}] === undefined ||` : ""
      }v[${k}].assert(o[${k}])`;
    }),
    "return o",
  ].join("\n");

  return new Function("v, o", body).bind(globalThis, properties) as DefObject<
    T
  >["assert"];
};

const reportArray = (def: Def) => {
  const body = [
    'if (!Array.isArray(a)) return [{ path: p, type: "array", value: a }]',
    "const failures = []",
    "let i = -1; const max = a.length",
    "while (++i < max) {",
    "  const e = a[i]",
    def.type === "object" || def.type === "array"
      ? `const _ = v.report(e, [...p, i]); (_.length && failures.push(..._))`
      : `${
        def.optional ? "e === undefined ||" : ""
      }typeof e === "${def.type}" || failures.push({ ${
        [
          `path: [...p, i]`,
          `type: "${def.type}"`,
          `value: e`,
        ].join(", ")
      } })`,
    "  if (failures.length > 9) return failures",
    "}",
    "return failures",
  ].join("\n");

  return new Function("v, a, p = []", body);
};

const assertArray = <T extends Def["assert"]>(assert: T) => (a: unknown) => {
  if (!Array.isArray(a)) throw Error("type assertion failed");
  a.forEach(assert);
  return a as ReturnType<T>[];
};

const assertNumber = (value: unknown) => {
  if (typeof value === "number" && !isNaN(value)) return value;
  throw Error(`type assertion failed`);
};

const assertString = (value: unknown) => {
  if (typeof value === "string") return value;
  throw Error(`type assertion failed`);
};

const assertBoolean = (value: unknown) => {
  if (typeof value === "boolean") return value;
  throw Error(`type assertion failed`);
};

export const NUM = (description?: string) =>
  ({
    type: "number",
    assert: assertNumber,
    description,
    report: (value: unknown) => [{ type: "number", value, path: [] }],
  }) satisfies DefNumber;

export const STR = (description?: string) =>
  ({
    type: "string",
    assert: assertString,
    description,
    report: (value: unknown) => [{ type: "string", value, path: [] }],
  }) satisfies DefString;

export const BOOL = (description?: string) =>
  ({
    type: "boolean",
    assert: assertBoolean,
    description,
    report: (value: unknown) => [{ type: "boolean", value, path: [] }],
  }) satisfies DefBoolean;

export const optional = <T extends Def>(def: T): Optional<T> => {
  const { assert, description, ...rest } = def;
  const optionalAssert: OptionalAssert<typeof assert> = (value: unknown) =>
    value === undefined ? undefined : assert(value);
  return {
    ...rest,
    description,
    optional: true,
    assert: optionalAssert,
  } as Optional<T>;
};

export const OBJ = <T extends Record<string, Def>>(
  properties: T,
  description?: string,
): DefObject<T> => {
  const report = reportObject(properties);
  const assert = assertObject(properties);
  return { type: "object", properties, report, assert, description };
};

export const ARR = <T extends Def>(
  def: T,
  description?: string,
): DefArray<T> => ({
  type: "array",
  of: def,
  report: reportArray(def).bind(globalThis, def),
  assert: assertArray(def.assert) as DefArray<T>["assert"],
  description,
});

export const LIST = <const T extends readonly (string | number)[]>(
  possibleValues: T,
  description?: string,
): DefList<T> => ({
  type: "list",
  of: possibleValues,
  report: (value: unknown, path: (string | number)[] = []) => {
    if (possibleValues.includes(value as T[number])) return [];
    return [{
      path,
      type: "list",
      value,
      expected: possibleValues,
    }];
  },
  assert: (value: unknown): T[number] => {
    if (possibleValues.includes(value as T[number])) {
      return value as T[number];
    }
    throw new Error(
      `Invalid value. Expected one of: ${possibleValues.join(", ")}`,
    );
  },
  description,
});

export const UNION = <T extends readonly Def[]>(...types: T): DefUnion<T> => ({
  type: "union",
  of: types,
  report: (value: unknown, path: (string | number)[] = []) => {
    const failures: ValidatorFailure<DefUnion<T>>[] = [];
    for (const type of types) {
      const result = type.report(value, path);
      if (result.length === 0) return [];
      failures.push(...result);
    }
    return failures;
  },
  assert: (value: unknown): ReturnType<T[number]["assert"]> => {
    for (const type of types) {
      try {
        return type.assert(value);
      } catch {
        // Ignore
      }
    }
    throw new Error(
      `Invalid value. Expected one of: ${types.map((t) => t.type).join(", ")}`,
    );
  },
});

// const Article = OBJ({
//   id: NUM("Unique identifier for the article"),
//   title: STR("Title of the article"),
//   isDraft: optional(BOOL("Whether the article is in draft state")),
//   tags: ARR(STR("A tag name"), "List of tags associated with the article"),
//   author: optional(OBJ({
//     id: NUM("Author's unique identifier"),
//   }, "Author information")),
// }, "Article object containing all article information");

// type ArticleType = ReturnType<typeof Article.assert>;

// const aaa = Article.report({
//   id: 5,
//   title: "hello",
//   isDraft: true,
//   tags: [],
//   // author: { id: 1 },
// });
