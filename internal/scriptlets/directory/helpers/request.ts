import { parseRegexp } from "./parseRegexp";

type RequestProp = typeof REQUEST_PROPS[number];
type ParsedPropsToMatch = Partial<Record<RequestProp, string | RegExp>>;

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
    }
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

    res[key] = parseRegexp(value) || value;
  }

  return res;
}

export function matchRequest(props: ParsedPropsToMatch, requestArgs: Parameters<typeof window.fetch>): boolean {
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

  
  return Object.entries(props).every(([k, v]) => typeof request[k] === 'string' && matchProp(v, request[k]));
}

function matchProp(prop: string | RegExp, value: string): boolean {
  return typeof prop === 'string' ? prop === value : prop.test(value);
}
