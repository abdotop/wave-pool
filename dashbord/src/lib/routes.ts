import type { Handler } from "./api";
import { type Def, NUM, OBJ, STR, UNION } from "./validator";

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
    output: UNION(
      OBJ({
        access_token: STR("JWT access token"),
        refresh_token: STR("JWT refresh token"),
        expires_in: NUM("token expiration time in seconds"),
      }, "response body"),
      STR("error message"),
    ),
    description: "Register a new user",
  }),
  "GET/api/v1/me": route({
    output: OBJ({
      id: STR("user id"),
      phone: STR("user phone number"),
      created_at: STR("user creation timestamp"),
    }, "response body"),
    description: "Get current authenticated user",
  }),
  "POST/api/v1/auth/refresh": route({
    input: OBJ({
      refresh_token: STR("JWT refresh token"),
    }, "request body"),
    output: OBJ({
      access_token: STR("new JWT access token"),
      refresh_token: STR("new JWT refresh token"),
      expires_in: NUM("token expiration time in seconds"),
    }, "response body"),
    description: "Refresh access token using refresh token",
  }),
  "DELETE/api/v1/auth/logout": route({
    input: OBJ({
      refresh_token: STR("JWT refresh token"),
    }, "request body"),
    output: STR("logout successful"),
    description: "Logout user and invalidate refresh token",
  }),
} as const;
