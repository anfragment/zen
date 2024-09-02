import { createLogger } from "./helpers/logger";
import { genRandomResponse, matchXhr, ParsedPropsToMatch, parsePropsToMatch } from "./helpers/request";

const logger = createLogger('prevent-xhr');

interface ExtendedXHR extends XMLHttpRequest {
  prevent?: boolean;
  collectedHeaders?: [string, string][];
}

export function preventXHR(propsToMatch: string, randomizeResponseTextPattern?: string): void {
  if (typeof Proxy === 'undefined') {
    logger('Proxy is undefined');
    return;
  }
  if (typeof propsToMatch !== 'string') {
    logger('propsToMatch is undefined');
    return;
  }

  let parsedProps = parsePropsToMatch(propsToMatch);

  const openOverride: ProxyHandler<typeof XMLHttpRequest.prototype.open>['apply'] = (target, thisArg: ExtendedXHR, args: Parameters<typeof XMLHttpRequest.prototype.open>) => {
    if (!matchXhr(parsedProps, ...args)) {
      thisArg.prevent = false;
      return Reflect.apply(target, thisArg, args);
    }

    thisArg.prevent = true;
    thisArg.collectedHeaders = [];
    thisArg.setRequestHeader = new Proxy(thisArg.setRequestHeader, {
      apply: (target, thisArg, args) => {
        thisArg.collectedHeaders.push(args);
        return Reflect.apply(target, thisArg, args);
      }
    });
    return Reflect.apply(target, thisArg, args);
  }

  const sendOverride: ProxyHandler<typeof XMLHttpRequest.prototype.send>['apply'] = (target, thisArg: ExtendedXHR, args: Parameters<typeof XMLHttpRequest.prototype.send>) => {
    if (!thisArg.prevent) {
      return Reflect.apply(target, thisArg, args);
    }

    let response: string | Blob | ArrayBuffer = '';
    if (thisArg.responseType === 'blob') {
      response = new Blob();
    } else if (thisArg.responseType === 'arraybuffer') {
      response = new ArrayBuffer(8);
    }

    let responseText;
    if (randomizeResponseTextPattern) {
      try {
        responseText = genRandomResponse(randomizeResponseTextPattern);
      } catch (ex) {
        logger('Error generating random responseText', ex);
      }
    }
  }
}