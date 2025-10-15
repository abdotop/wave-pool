import type { Handler } from "./api";
import { type Def, STR } from "./validator";

export const route = <TInput, TOutput>(
  h: TInput extends Def ? TOutput extends Def ? Handler<TInput, TOutput>
    : Handler<TInput, undefined>
    : TOutput extends Def ? Handler<undefined, TOutput>
    : Handler<undefined, undefined>,
) => h;

export const defs = {
  "GET/api/health": route({
    output: STR("the api is healthy"),
    description: "Health check endpoint",
  }),
} as const;
