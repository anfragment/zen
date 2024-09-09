import { createLogger } from './helpers/logger';
import { matchFetch, ParsedPropsToMatch, parsePropsToMatch } from './helpers/request';

type ResponseBody = 'emptyObj' | 'emptyArr' | 'emptyStr' | '';
type ResponseType = 'basic' | 'cors' | 'opaque';

const logger = createLogger('prevent-fetch');

export function preventFetch(
  propsToMatch: string,
  responseBody: ResponseBody = 'emptyObj',
  responseType?: ResponseType,
) {
  if (typeof fetch === 'undefined' || typeof Proxy === 'undefined' || typeof Response === 'undefined') {
    logger.warn('Either fetch, Proxy, or Response is not supported in this environment');
    return;
  }

  let response;
  switch (responseBody) {
    case '':
    case 'emptyObj':
      response = '{}';
      break;
    case 'emptyArr':
      response = '[]';
      break;
    case 'emptyStr':
      response = '';
      break;
    default:
      logger.warn(`Invalid responseBody: ${responseBody}`);
      return;
  }

  if (typeof responseType === 'string') {
    if (responseType !== 'basic' && responseType !== 'cors' && responseType !== 'opaque') {
      logger.warn(`Invalid responseType: ${responseType}`);
      return;
    }
  }

  let parsedProps: ParsedPropsToMatch;
  try {
    parsedProps = parsePropsToMatch(propsToMatch);
  } catch (ex) {
    logger.warn('Error parsing props', ex);
  }

  // @ts-ignore
  fetch = new Proxy(fetch, {
    apply: (target, thisArg, args: Parameters<typeof fetch>) => {
      if (!matchFetch(parsedProps, args)) {
        return Reflect.apply(target, thisArg, args);
      }
    },
  });
}
