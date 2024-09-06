import { parseRegexp } from '../parseRegexp';

export type RequestProp = (typeof REQUEST_PROPS)[number];
export type ParsedPropsToMatch = Partial<Record<RequestProp, string | RegExp>>;

const REQUEST_PROPS = [
  'url',
  'method',
  'credentials',
  'cache',
  'redirect',
  'referrer',
  'referrerPolicy',
  'integrity',
  'mode',
] as const;

export function parsePropsToMatch(propsToMatch: string): ParsedPropsToMatch {
  if (propsToMatch === '' || propsToMatch === '*') {
    return {};
  }

  const wholeRegexp = parseRegexp(propsToMatch);
  if (wholeRegexp !== null) {
    return {
      url: wholeRegexp,
    };
  }

  const res: ParsedPropsToMatch = {};
  const segments = propsToMatch.split(' ');
  for (const segment of segments) {
    if (!segment.includes(':')) {
      res.url = parseRegexp(segment) || segment;
      continue;
    }

    const [key, value] = segment.split(':');
    if (key === '' || value === undefined || value === '') {
      throw new Error(`Invalid segment: "${segment}"`);
    }
    if (!REQUEST_PROPS.includes(key as RequestProp)) {
      throw new Error(`Invalid segment key: "${key}"`);
    }

    res[key as RequestProp] = parseRegexp(value) || value;
  }

  return res;
}

export function matchFetch(props: ParsedPropsToMatch, requestArgs: Parameters<typeof window.fetch>): boolean {
  let request: (Request | RequestInit) & { url: string };
  if (requestArgs[0] instanceof Request) {
    console.assert(requestArgs[1] === undefined);
    request = requestArgs[0];
  } else {
    if (requestArgs[1] === undefined) {
      throw new Error('Malformed requestArgs, missing element at index 1');
    }
    request = {
      ...requestArgs[1],
      url: requestArgs[0].toString(),
    };
  }

  for (const prop of Object.keys(props) as RequestProp[]) {
    if (
      request[prop] === undefined ||
      (prop === 'url' && !matchURL(props[prop]!, request[prop])) ||
      !matchProp(props[prop]!, request[prop])
    ) {
      return false;
    }
  }

  return true;
}

export function matchXhr(
  props: ParsedPropsToMatch,
  ...args: Parameters<typeof XMLHttpRequest.prototype.open>
): boolean {
  const request: Partial<Record<RequestProp, string>> = {
    method: args[0],
    url: args[1].toString(),
    // Other arguments are skipped intentionally.
  };

  for (const prop of Object.keys(props) as RequestProp[]) {
    if (
      request[prop] === undefined ||
      (prop === 'url' && !matchURL(props[prop]!, request[prop])) ||
      !matchProp(props[prop]!, request[prop])
    ) {
      return false;
    }
  }

  return true;
}

function matchURL(urlProp: string | RegExp, url: string): boolean {
  if (urlProp instanceof RegExp) {
    return urlProp.test(url);
  }
  return urlProp === url || urlProp === getHostnameFromUrl(url);
}

function matchProp(prop: string | RegExp, value: string): boolean {
  if (typeof prop === 'string') {
    return prop === value;
  }
  return prop.test(value);
}

function getHostnameFromUrl(input: string): string | null {
  try {
    if (!/^https?:\/\//i.test(input)) {
      input = `http://${input}`;
    }

    const url = new URL(input);
    return url.hostname;
  } catch {
    return null;
  }
}
