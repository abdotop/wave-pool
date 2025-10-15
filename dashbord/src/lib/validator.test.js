import { eq, test, throws } from "/api/lib/test.ts";
import { ARR, NUM, OBJ, optional, STR } from "/api/lib/validator.ts";

test("Validator - Success cases", async (step) => {
  await step("should validate simple object", () => {
    const schema = OBJ({
      name: STR("User name"),
      age: NUM("User age"),
    });

    const data = { name: "John", age: 30 };
    const result = schema.assert(data);
    eq(result, data);
  });

  await step("should validate nested object with arrays", () => {
    const schema = OBJ({
      user: OBJ({
        name: STR(),
        tags: ARR(STR()),
      }),
    });

    const data = {
      user: {
        name: "John",
        tags: ["admin", "user"],
      },
    };
    const result = schema.assert(data);
    eq(result, data);
  });

  await step("should handle optional fields", () => {
    const schema = OBJ({
      name: STR(),
      age: optional(NUM()),
    });

    const data = { name: "John" };
    const result = schema.assert(data);
    eq(result, data);
  });
});

test("Validator - Error cases", async (step) => {
  await step("should throw on missing required field", () => {
    const schema = OBJ({
      name: STR(),
      age: NUM(),
    });

    throws(
      () => schema.assert({ name: "John" }),
      Error,
      "type assertion failed",
    );
  });

  await step("should throw on invalid type", () => {
    const schema = OBJ({
      name: STR(),
      age: NUM(),
    });

    throws(
      () => schema.assert({ name: "John", age: "30kds" }),
      Error,
      "type assertion failed",
    );
  });
});
