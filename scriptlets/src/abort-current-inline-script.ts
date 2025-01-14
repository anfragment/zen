import { createLogger } from './helpers/logger';
import { parseRegexpFromString, parseRegexpLiteral } from './helpers/parseRegexp';

const logger = createLogger('abort-current-inline-script');

export function abortCurrentInlineScript(property: string, search?: string | null): void {
  if (typeof property !== 'string' || property.length === 0) {
    logger.warn('property should be a non-empty string');
    return;
  }

  let searchRe: RegExp | null;
  if (typeof search === 'string' && search.length > 0) {
    searchRe = parseRegexpLiteral(search) || parseRegexpFromString(search);
  }
  const rid = generateRandomId();

  const currentScript = document.currentScript;

  const abort = () => {
    const element = document.currentScript;

    if (
      true
      // element instanceof HTMLScriptElement &&
      // // !element.src &&
      // element !== currentScript &&
      // (!searchRe || searchRe.test(element.textContent || ''))
    ) {
      logger.info(`Blocked ${property} in`, element);
      throw new ReferenceError(`Aborted script with ID: ${rid}`);
    }
  };

  if (!property.includes('.')) {
    const descriptor = Object.getOwnPropertyDescriptor(window, property) || {};
    const originalGetter = descriptor.get;
    const originalSetter = descriptor.set;

    let currentValue = window[property as any];
    Object.defineProperty(window, property, {
      configurable: true,
      get() {
        abort();
        return originalGetter ? originalGetter.call(this) : currentValue;
      },
      set(value) {
        abort();
        if (originalSetter) {
          originalSetter.call(this, value);
        } else {
          currentValue = value;
        }
      },
    });
    return;
  }

  const path = property.split('.');
  const rootProp = path.shift() as string;

  const get = (chain: string[]) => (target: any, key: any) => {
    const link = target[key];
    if (link == undefined) {
      return link;
    }

    if (chain.length === 1 && chain[0] === key) {
      abort();
      return link;
    }
    if (chain[0] !== key || typeof link !== 'object') {
      if (
        typeof link === 'function' &&
        // Prevent rebinding if the function is already bound.
        // Bound functions can be identified by the "bound " prefix in their name. See:
        // https://262.ecma-international.org/6.0/index.html#sec-function.prototype.bind
        !link.name.startsWith('bound ')
      ) {
        // Fixes https://github.com/anfragment/zen/issues/201
        return link.bind(target);
      }
      return link;
    }

    const newChain = chain.slice(1);
    const handler: ProxyHandler<typeof link> = {
      get: get(newChain),
    };
    if (newChain.length === 1) {
      handler.set = (target, prop, value) => {
        if (prop === newChain[0]) {
          abort();
        }
        target[prop] = value;

        return true;
      };
    }

    return new Proxy(link ?? {}, handler);
  };

  let currentValue = window[rootProp as any];

  Object.defineProperty(window, rootProp as any, {
    configurable: true,
    get: () => {
      return new Proxy(currentValue, {
        get: get(path),
      });
    },
    set: (v) => {
      currentValue = v;
    },
  });

  // Enhance error handling for the thrown ReferenceError
  window.addEventListener('error', (event) => {
    if (event.error instanceof ReferenceError && event.error.message.includes(rid)) {
      event.preventDefault();
    }
  });
}

/**
 * Generates a random unique ID.
 *
 * @returns {string} - A random string ID.
 */
function generateRandomId(): string {
  return Math.random().toString(36).substring(2, 15);
}
