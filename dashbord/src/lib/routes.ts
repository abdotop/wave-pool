import type { Handler } from "./api";
import { type Def, NUM, OBJ, STR } from "./validator";

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
  "POST/api/v1/auth": route({
    input: OBJ({
      phone: STR("user phone number"),
      pin: STR("user pin code"),
    }, "request body"),
    output: OBJ({
      access_token: STR("JWT access token"),
      refresh_token: STR("JWT refresh token"),
      expires_in: NUM("token expiration time in seconds"),
    }, "response body"),
    description: "Register a new user",
  }),
} as const;
