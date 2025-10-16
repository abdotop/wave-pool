import type { Handler } from "./api";
import { A } from "./router";
import { ARR, type Def, NUM, OBJ, STR, UNION } from "./validator";

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
      business: OBJ({
        id: STR("business id"),
        name: STR("business name"),
        country: STR("business country"),
        currency: STR("business currency"),
      }, "business information"),
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
  "POST/api/v1/api-keys": route({
    input: OBJ({
      env: STR("environment for the API key, e.g., 'production', 'staging'"),
      scopes: ARR(STR("scopes for the API key"), "array of scopes"),
    }, "request body"),
    output: OBJ({
      id: STR("API key ID"),
      secret_key: STR("API key secret (only shown once)"),
      prefix: STR("API key prefix"),
      scopes: ARR(STR("scopes for the API key"), "array of scopes"),
      env: STR("environment for the API key"),
    }, "response body"),
    description: "Create a new API key",
  }),
  "GET/api/v1/api-keys": route({
    output: ARR(
      OBJ({
        id: STR("API key ID"),
        business_id: STR("associated business ID"),
        prefix: STR("API key prefix"),
        scopes: ARR(STR("scopes for the API key"), "array of scopes"),
        env: STR("environment for the API key"),
        status: STR("API key status: active or revoked"),
        created_at: STR("API key creation timestamp"),
      }, "API key object"),
      "array of API keys",
    ),
    description: "List all API keys for the authenticated user's business",
  }),
  "DELETE/api/v1/api-keys/{key_id}": route({
    input: OBJ({
      key_id: STR("ID of the API key to revoke"),
    }, "path parameter"),
    description: "Revoke (delete) an API key by its ID",
  }),
} as const;
