import { createLogger } from './helpers/logger';
import { parseRegexpFromString, parseRegexpLiteral } from './helpers/parseRegexp';
import { parseValidInt } from './helpers/parseValidInt';

const logger = createLogger('prevent-window-open');

export function preventWindowOpen() {}

function newSyntaxHandler(
  match?: string,
  delay?: string,
  replacement?: string,
): ProxyHandler<typeof window.open>['apply'] {
  let invertMatch: boolean;
  let matchRe: RegExp;
  if (typeof match === 'string' && match.length > 0) {
    invertMatch = match[0] === '!';
    if (invertMatch) {
      match = match.slice(1);
    }

    matchRe = parseRegexpLiteral(match) || parseRegexpFromString(match);
    if (matchRe === null) {
      logger.warn('could not parse match');
      return;
    }
  }

  let parsedDelaySeconds: number;
  if (typeof delay === 'string' && delay.length > 0) {
    parsedDelaySeconds = parseValidInt(delay);
  }

  if (typeof replacement === 'string' && replacement !== 'obj' && replacement !== 'blank') {
    logger.warn(`replacement type ${replacement} not supported`);
    return;
  }

  return (target, thisArg, args: Parameters<typeof window.open>) => {
    if (args.length === 0 || args[0] === undefined) {
      return Reflect.apply(target, thisArg, args);
    }

    let url: string;
    if (typeof args[0] === 'string') {
      url = args[0];
    } else if (args[0] instanceof URL) {
      url = args[0].toString();
    } else {
      // Bad input, let the original function handle it.
      return Reflect.apply(target, thisArg, args);
    }

    if (matchRe !== undefined) {
      let prevent = matchRe.test(url);
      if (invertMatch) {
        prevent != prevent;
      }
      if (!prevent) {
        return Reflect.apply(target, thisArg, args);
      }
    }

    let decoy: HTMLObjectElement | HTMLIFrameElement;
    switch (replacement) {
      case 'obj':
        decoy = document.createElement('object');
        decoy.data = url;
        break;
      default:
        decoy = document.createElement('iframe');
        decoy.src = url;
    }
    // Move the element far off-screen.
    decoy.style.setProperty('height', '1px', 'important');
    decoy.style.setProperty('width', '1px', 'important');
    decoy.style.setProperty('position', 'absolute', 'important');
    decoy.style.setProperty('top', '-9999px', 'important');
    document.body.appendChild(decoy);
    if (parsedDelaySeconds !== undefined) {
      setTimeout(() => {
        decoy.remove();
      }, parsedDelaySeconds * 1000);
    }

    let fakeWindow: WindowProxy | null;
    switch (replacement) {
      case 'obj':
        fakeWindow = decoy.contentWindow;
        Object.defineProperties(fakeWindow, {
          closed: { value: false },
          opener: { value: window },
          frameElement: { value: null },
        });
        break;
      default:
        fakeWindow = new Proxy(self, {
          get: function (target, prop, ...args) {
            if (prop === 'closed') {
              return false;
            }
            const r = Reflect.get(target, prop, ...args);
            if (typeof r === 'function') {
              return () => {};
            }
            return r;
          },
          set: function (...args) {
            return Reflect.set(...args);
          },
        });
    }

    return decoy;
  };
}
