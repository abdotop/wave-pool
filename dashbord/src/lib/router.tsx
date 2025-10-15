import type { JSX } from "preact";
import { computed, Signal } from "@preact/signals";

const isCurrentURL = (alt: URL) => {
  const url = urlSignal.value;
  if (url.href === alt.href) return true;
  if (url.origin !== alt.origin) return false;
  if (url.pathname !== alt.pathname) return false;
  // handle special case, same params but different order.
  // must still be equal
  if (alt.searchParams.size !== url.searchParams.size) return false;
  if (alt.searchParams.size === 0) return false; // no params -> not the same
  // both urls have the same numbers of params, now let's confirm they are
  // all the same values
  for (const [k, v] of alt.searchParams) {
    if (url.searchParams.get(k) !== v) return false;
  }
  return true;
};

// ensure we never have trailing /
const initialUrl = new URL(location.href);
if (location.pathname.at(-1) === "/") {
  initialUrl.pathname = initialUrl.pathname.slice(0, -1);
  history.replaceState({}, "", initialUrl.href);
}

const urlSignal = new Signal(initialUrl);
const { origin } = initialUrl;

const dispatchNavigation = () => {
  // If the path did change, we update the local state and trigger the change
  const url = new URL(location.href);
  url.pathname.at(-1) === "/" && (url.pathname = url.pathname.slice(0, -1));
  if (isCurrentURL(url)) return;
  urlSignal.value = url;
};

addEventListener("popstate", dispatchNavigation);
addEventListener("hashchange", dispatchNavigation);

const navigateUrl = (to: string, replace = false) => {
  history[replace ? "replaceState" : "pushState"]({}, "", to);
  dispatchNavigation();
};

type ParamPrimitive = string | number | boolean;
type ParamValue = ParamPrimitive | null | undefined | ParamPrimitive[];
type GetUrlProps = {
  href?: string;
  hash?: string;
  // params supports arrays to allow multiple identical keys: { tag: ['a','b'] } -> ?tag=a&tag=b
  params?: URLSearchParams | Record<string, ParamValue>;
};

const getUrl = ({ href, hash, params }: GetUrlProps) => {
  const currentUrl = urlSignal.value;
  const url = new URL(href || currentUrl, origin);
  hash != null && (url.hash = hash);
  url.pathname.at(-1) === "/" && (url.pathname = url.pathname.slice(0, -1));
  if (!params) {
    if (url.pathname !== currentUrl.pathname) return url;
    url.search = `?${currentUrl.searchParams}`;
    return url;
  }
  for (const [key, value] of Object.entries(params)) {
    if (Array.isArray(value)) {
      // Remove existing then append each to preserve ordering
      url.searchParams.delete(key);
      for (const v of value) {
        if (v === false || v == null) continue; // skip deletions inside arrays
        if (v === true) {
          url.searchParams.append(key, "");
        } else {
          url.searchParams.append(key, v);
        }
      }
      continue;
    }
    if (value === true) {
      url.searchParams.set(key, "");
    } else if (value === false || value == null) {
      url.searchParams.delete(key);
    } else {
      url.searchParams.set(key, value);
    }
  }
  return url;
};

export const navigate = (props: GetUrlProps & { replace?: boolean }) =>
  navigateUrl(getUrl(props).href, props.replace);

export type LinkProps =
  & { replace?: boolean }
  & JSX.HTMLAttributes<HTMLAnchorElement>
  & GetUrlProps;

export const A = ({
  href,
  hash,
  params,
  replace,
  onClick,
  onMouseDown,
  ...props
}: LinkProps) => {
  const url = getUrl({ href, hash, params });
  const noRouting = url.origin !== origin ||
    url.pathname.startsWith("/api/");

  if (noRouting) {
    return (
      <a
        href={url.href}
        onClick={onClick}
        onMouseDown={onMouseDown}
        {...props}
      />
    );
  }

  const mouseDownHandler = (
    event: JSX.TargetedMouseEvent<HTMLAnchorElement>,
  ) => {
    // Experimenting with using `onMouseDown` to trigger
    // it's faster but not cancellable, let's see how it's percieved
    if (typeof onMouseDown === "function") {
      onMouseDown(event);
    }
    // We don't want to skip if it's a special click
    // that would break the default browser behaviour
    const shouldSkip = event.defaultPrevented ||
      event.button ||
      event.metaKey ||
      event.altKey ||
      event.ctrlKey ||
      event.shiftKey;

    if (shouldSkip) return;

    // In the normal case we handle the routing internally
    event.preventDefault();
    navigateUrl(url.href, replace);
  };

  return (
    <a
      href={url.href}
      onMouseDown={mouseDownHandler}
      onClick={(event) => {
        if (typeof onClick === "function") {
          onClick(event);
        }

        const notMouse = event.clientX === 0 && event.clientY === 0;
        if (notMouse) {
          mouseDownHandler(event);
        } else {
          event.preventDefault();
        }
      }}
      {...props}
    />
  );
};

const toDeletedParam = (k: string) => [k, null];
export const replaceParams = (
  newParams: Record<string, string | number | boolean | null | undefined>,
) => ({
  ...Object.fromEntries(
    [...urlSignal.value.searchParams.keys()].map(toDeletedParam),
  ),
  ...newParams,
});

// wrap params behind a proxy to allow accessing
const params = new Proxy({} as Record<string, Signal<string | null>>, {
  // this allow enumeration to work, so Object.keys(), {...params} will work
  ownKeys: () => [...urlSignal.value.searchParams.keys()],
  getOwnPropertyDescriptor: (_, key) => ({
    enumerable: true,
    configurable: true,
    value: urlSignal.value.searchParams.get(key as string),
  }),

  // this is when we get a single key
  get: (cache, key) =>
    (typeof key !== "string" || !key) ? null : (cache[key] || (
      cache[key] = computed(() => urlSignal.value.searchParams.get(key))
    )).value,
}) as unknown as Record<string, string | null>;

// http://localhost:8000/user/settings?id=454&options=open#display
// url.path: 'user/settings'
// url.hash: 'display'
// url.params: { id: 454, option: 'open' }
const hashSignal = computed(() => urlSignal.value.hash);
const pathSignal = computed(() => urlSignal.value.pathname);
export const url = {
  get path() {
    return pathSignal.value;
  },
  get hash() {
    return hashSignal.value;
  },
  params,
  // Retrieve all values (including duplicates) for a given key
  getAll: (key: string) => urlSignal.value.searchParams.getAll(key),
  // All param entries preserving duplicates & order
  paramEntries: () => [...urlSignal.value.searchParams.entries()],
  equals: (url: URL) => isCurrentURL(url),
};
