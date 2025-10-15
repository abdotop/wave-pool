import { Signal } from "@preact/signals";
import { useMemo } from "preact/hooks";
import { defs } from "./routes.ts";
import type { Def } from "./validator.ts";

export type HttpMethod = "GET" | "POST" | "PUT" | "DELETE" | "PATCH";
export type Awaitable<T> = Promise<T> | T;
export type Asserted<T extends Def> = ReturnType<T["assert"]>;

export type Handler<TInput, TOutput> = {
  description?: string;
  input?: TInput;
  output?: TOutput;
};

export class ErrorWithData extends Error {
  public data: Record<string, unknown>;
  constructor(message: string, data: Record<string, unknown>) {
    super(message);
    this.name = "ErrorWithData";
    this.data = data;
  }
}

export class ErrorWithBody extends ErrorWithData {
  public body: string;
  constructor(body: string, data: Record<string, unknown>) {
    super("Failed to parse body", data);
    this.name = "ErrorWithBody";
    this.body = body;
  }
}

type ExtractAndAssert<T> = T extends Def ? Asserted<T>
  : undefined;

// I made the other field always explicitly undefined and optional
// this way we do not have to check all the time that they exists
type RequestState<T> =
  | { data: T; pending?: undefined; controller?: undefined; error?: undefined }
  | {
    data?: T | undefined;
    pending: number;
    controller?: AbortController;
    error?: undefined;
  }
  | {
    data?: T | undefined;
    pending?: undefined;
    controller?: undefined;
    error: ErrorWithBody | ErrorWithData | Error;
  };

type Options = {
  headers?: HeadersInit;
  signal?: AbortSignal;
};

const withoutBody = new Set([
  204, // NoContent
  205, // ResetContent
  304, // NotModified
]);

function createApiClient<T>(def: T, baseUrl = "") {
  type HandlerIO<K extends keyof T> = T[K] extends
    Handler<infer TInput, infer TOutput>
    ? [ExtractAndAssert<TInput>, ExtractAndAssert<TOutput>]
    : never;

  function makeClientCall<K extends keyof T>(urlKey: K) {
    type IO = HandlerIO<K>;
    type Input = IO[0];
    type Output = IO[1];
    const key = urlKey as string;
    const slashIndex = key.indexOf("/");
    const method = key.slice(0, slashIndex) as HttpMethod;
    const path = key.slice(slashIndex);
    const defaultHeaders = { "Content-Type": "application/json" };

    async function fetcher(input?: Input, options?: Options | undefined) {
      let url = `${baseUrl}${path}`;
      let headers = options?.headers;
      if (!headers) {
        headers = defaultHeaders;
      } else {
        headers instanceof Headers || (headers = new Headers(headers));
        for (const [key, value] of Object.entries(defaultHeaders)) {
          headers.set(key, value);
        }
      }

      let bodyInput: string | undefined = undefined;
      if (input) {
        method === "GET"
          ? (url += `?${new URLSearchParams(input as Record<string, string>)}`)
          : (bodyInput = JSON.stringify(input));
      }

      const response = await fetch(
        url,
        { ...options, method, headers, body: bodyInput },
      );
      if (withoutBody.has(response.status)) return null as unknown as Output;
      const body = await response.text();
      let payload;
      try {
        payload = JSON.parse(body);
        if (response.ok) return payload as Output;
      } catch {
        try {
          const result = (def[urlKey] as unknown as Handler<any, any>).output
            ?.assert(body);
          if (response.ok && result !== undefined) return result as Output;
        } catch {
          // ignore json parse error, we throw below
        }
        throw new ErrorWithBody(body, { response });
      }
      const { message, ...data } = payload;
      throw new ErrorWithData(message, data);
    }

    const signal = () => {
      const $ = new Signal<RequestState<Output>>({ pending: 0 });
      return {
        $,
        reset: () => {
          $.peek().controller?.abort();
          $.value = { pending: 0 };
        },
        fetch: async (input, headers: HeadersInit) => {
          const prev = $.peek();
          try {
            const controller = new AbortController();
            prev.controller?.abort();
            $.value = { pending: Date.now(), controller, data: prev.data };
            const { signal } = controller;
            $.value = { data: await fetcher(input, { signal, headers }) };
          } catch (err) {
            $.value = (err instanceof DOMException && err.name === "AbortError")
              ? { pending: 0, data: prev.data }
              : {
                error: err as (ErrorWithBody | ErrorWithData | Error),
                data: prev.data,
              };
          }
        },
        get data() {
          return $.value.data;
        },
        get error() {
          return $.value.error;
        },
        get pending() {
          return $.value.pending;
        },
      } as RequestState<Output> & {
        $: Signal<RequestState<Output>>;
        reset: () => void;
        fetch: (input?: Input, options?: Options | undefined) => Promise<void>;
      };
    };

    const use = () => useMemo(signal, []);
    return { fetch: fetcher, use, signal };
  }

  const client = {} as { [K in keyof T]: ReturnType<typeof makeClientCall<K>> };
  const lazy = (k: keyof T) => client[k] || (client[k] = makeClientCall(k));
  return new Proxy(client, {
    get: (_, key: string) => lazy(key as keyof T),
  });
}

export const api = createApiClient(defs);

export type ApiOutput = {
  [K in keyof typeof api]: Awaited<ReturnType<typeof api[K]["fetch"]>>;
};

export type ApiInput = {
  [K in keyof typeof api]: Awaited<Parameters<typeof api[K]["fetch"]>[0]>;
};
