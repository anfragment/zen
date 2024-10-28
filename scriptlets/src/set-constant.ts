import { createLogger } from './helpers/logger';
import { matchStack } from './helpers/matchStack';
import { parseRegexpFromString, parseRegexpLiteral } from './helpers/parseRegexp';

const logger = createLogger('set-constant');

export function setConstant(
  property: string,
  value: string,
  stack?: string,
  valueWrapper?: string,
  setProxyTrap?: boolean,
) {
  if (setProxyTrap !== undefined) {
    logger.warn('setProxyTrap will be ignored');
  }

  let fakeValue: any;
  switch (value) {
    case 'undefined':
      fakeValue = undefined;
      break;
    case 'false':
      fakeValue = false;
      break;
    case 'true':
      fakeValue = true;
      break;
    case 'null':
      fakeValue = null;
      break;
    case 'emptyObj':
      fakeValue = {};
      break;
    case 'emptyArr':
      fakeValue = [];
      break;
    case 'noopFunc':
      fakeValue = () => {};
      break;
    case 'noopCallbackFunc':
      fakeValue = () => () => {};
      break;
    case 'trueFunc':
      fakeValue = () => true;
      break;
    case 'falseFunc':
      fakeValue = () => false;
      break;
    case 'throwFunc':
      fakeValue = () => {
        throw new Error();
      };
      break;
    case 'noopPromiseResolve':
      fakeValue = () => {
        return Promise.resolve(
          new Response('', {
            status: 200,
            statusText: 'OK',
          }),
        );
      };
      break;
    case 'noopPromiseReject':
      fakeValue = () => Promise.reject();
      break;
    case '':
      fakeValue = '';
      break;
    case '-1':
      fakeValue = -1;
      break;
    case 'yes':
      fakeValue = 'yes';
      break;
    case 'no':
      fakeValue = 'no';
      break;
    default: {
      const int = parseInt(value, 10);
      if (!isNaN(int) && int >= 0 && int <= 32767) {
        fakeValue = value;
        break;
      }
      throw new Error('Invalid value');
    }
  }

  const wrapped = fakeValue; // Avoid creating recursive functions by creating a temporary variable.
  switch (valueWrapper) {
    case 'asFunction':
      fakeValue = () => wrapped;
      break;
    case 'asCallback':
      fakeValue = () => () => wrapped;
      break;
    case 'asResolved':
      fakeValue = () => Promise.resolve(wrapped);
      break;
    case 'asRejected':
      fakeValue = () => Promise.reject(wrapped);
      break;
  }

  let stackRe: RegExp | null;
  if (stack !== undefined && stack !== '') {
    stackRe = parseRegexpLiteral(stack) || parseRegexpFromString(stack);
  }
  stackRe ??= null;

  const localKey = Symbol() as any;

  if (!property.includes('.')) {
    window[localKey] = window[property as any];
    const thisScript = document.currentScript;
    Object.defineProperty(window, property, {
      configurable: true,
      get: () => {
        if (
          // Allow value overwrite by later scriptlets.
          (document.currentScript !== null && thisScript !== null && document.currentScript === thisScript) ||
          (stackRe !== null && !matchStack(stackRe))
        ) {
          return window[localKey];
        }
        return fakeValue;
      },
      set: (v) => {
        window[localKey] = v;
      },
    });
    return;
  }

  const get = (chain: string[]) => (target: any, key: any) => {
    const link = target[key];

    if (chain.length === 1 && chain[0] === key) {
      return fakeValue;
    }
    if (chain[0] !== key || (typeof link !== 'object' && link != undefined)) {
      return link;
    }

    return new Proxy(link ?? {}, {
      get: get(chain.slice(1)),
    });
  };

  const rootChain = property.split('.');
  const rootProp = rootChain.shift() as any;
  const odesc = Object.getOwnPropertyDescriptor(window, rootProp);
  // Establish a chain of getters to ensure multiple set-constant rules cooperate
  // and always return a correct value when getting the root property of the chain.
  const prevGetter = odesc?.get;
  window[localKey] = window[rootProp];

  Object.defineProperty(window, rootProp, {
    configurable: true,
    get: () => {
      let capturedValue;
      if (typeof prevGetter === 'function') {
        capturedValue = prevGetter();
      } else {
        capturedValue = window[localKey];
      }

      if (typeof capturedValue !== 'object' || (stackRe !== null && !matchStack(stackRe))) {
        return capturedValue;
      }
      return new Proxy(capturedValue, {
        get: get(rootChain),
      });
    },
    set:
      typeof odesc?.set === 'function'
        ? odesc?.set
        : (v) => {
            window[localKey] = v;
          },
  });
}
