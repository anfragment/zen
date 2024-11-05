import { createPrune } from './helpers/jsonPrune';
import { createLogger } from './helpers/logger';
import { matchXhrArgs, ParsedPropsToMatch, parsePropsToMatch } from './helpers/request';

const logger = createLogger('json-prune-xhr-response');

// Use Symbols to avoid interference with any other scriptlets or libraries.
const requestHeaders = Symbol('requestHeaders');
const shouldPrune = Symbol('shouldPrune');

interface Uninitialized extends XMLHttpRequest {
  [shouldPrune]?: boolean;
  [requestHeaders]?: [string, string][];
}

interface ToPrune extends XMLHttpRequest {
  [shouldPrune]: true;
  [requestHeaders]: [string, string][];
}

type ExtendedXHR = Uninitialized & ToPrune;

export function jsonPruneXHRResponse(
  propsToRemove: string,
  requiredProps?: string,
  propsToMatch?: string,
  stack?: string,
): void {
  if (typeof Proxy === 'undefined') {
    logger.warn('Proxy is not supported in this environment');
    return;
  }
  if (typeof propsToRemove !== 'string' || propsToRemove.length === 0) {
    logger.warn('propsToMatch cannot be empty');
    return;
  }

  let parsedProps: ParsedPropsToMatch;
  if (typeof propsToMatch === 'string') {
    try {
      parsedProps = parsePropsToMatch(propsToMatch);
    } catch (ex) {
      logger.warn('error parsing propsToMatch', ex);
      return;
    }
  }

  const prune = createPrune(propsToRemove, requiredProps, stack);

  const { open: nativeOpen, send: nativeSend } = window.XMLHttpRequest.prototype;

  XMLHttpRequest.prototype.open = new Proxy(XMLHttpRequest.prototype.open, {
    apply: (target, thisArg: ExtendedXHR, args: Parameters<typeof XMLHttpRequest.prototype.open>) => {
      if (!matchXhrArgs(parsedProps, ...args)) {
        return Reflect.apply(target, thisArg, args);
      }

      thisArg[shouldPrune] = true;
      thisArg[requestHeaders] = [];

      return Reflect.apply(target, thisArg, args);
    },
  });

  XMLHttpRequest.prototype.setRequestHeader = new Proxy(XMLHttpRequest.prototype.setRequestHeader, {
    apply: (target, thisArg: ExtendedXHR, args: Parameters<typeof XMLHttpRequest.prototype.setRequestHeader>) => {
      if (!thisArg[shouldPrune]) {
        return Reflect.apply(target, thisArg, args);
      }

      thisArg[requestHeaders].push(args);
      return Reflect.apply(target, thisArg, args);
    },
  });

  XMLHttpRequest.prototype.open = new Proxy(XMLHttpRequest.prototype.open, {
    apply: (target, thisArg: ExtendedXHR, args: Parameters<typeof XMLHttpRequest.prototype.open>) => {
      if (!thisArg[shouldPrune]) {
        return Reflect.apply(target, thisArg, args);
      }

      const req = new XMLHttpRequest();
      req.addEventListener('readystatechange', async () => {
        if (req.readyState !== XMLHttpRequest.DONE) {
          return;
        }

        let modifiedResponse;
        try {
          let response = req.responseText || req.response;
          if (response instanceof ArrayBuffer) {
            const decoder = new TextDecoder(); // assume utf-8
            response = decoder.decode(response);
          } else if (response instanceof Blob) {
            response = await response.text(); // assume utf-8
          } else if (response instanceof Document) {
            throw new Error('Unable to prune Document-typed response');
          }

          const obj = typeof response === 'object' ? response : JSON.parse(response);
          const pruned = prune(obj);
          
        } catch () {

        }
        
      });
    },
  });
}
