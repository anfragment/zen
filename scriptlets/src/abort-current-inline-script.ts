import { createLogger } from './helpers/logger';
import { parseRegexpFromString, parseRegexpLiteral } from './helpers/parseRegexp';

const logger = createLogger('abort-current-inline-script');

type AnyObject = { [key: string]: any };

function isPropertyConfigurable(o: AnyObject, prop: string): boolean {
  if (!o) {
    return true;
  }

  const descriptor = Object.getOwnPropertyDescriptor(o, prop);
  if (!descriptor) {
    return true;
  }

  return Boolean(descriptor.configurable);
}

function createProxy(chainParts: string[] = []): any {
  const backingStore: AnyObject = {};
  return new Proxy(backingStore, {
    get(target, prop: string) {
      if (!(prop in target)) {
        target[prop] = createProxy([...chainParts, prop]);
      }
      return target[prop];
    },
    set(target, prop: string, value) {
      target[prop] = value;
      return true;
    },
  });
}

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
      element instanceof HTMLScriptElement &&
      element !== currentScript &&
      (!searchRe || searchRe.test(element.textContent || ''))
    ) {
      logger.info(`Blocked ${property} in, currentScript`);
      throw new ReferenceError(`Aborted script with ID: ${rid}`);
    }
  };

  function defineProxyChain(root: AnyObject, chain: string): void {
    const parts = chain.split('.');
    let current = root;

    for (let i = 0; i < parts.length; i++) {
      const part = parts[i];
      const isLast = i === parts.length - 1;
      const chainSoFar = parts.slice(0, i + 1);

      // final property in chain
      if (isLast) {
        const propName = chainSoFar[chainSoFar.length - 1];
        const originalDescriptor = Object.getOwnPropertyDescriptor(current, propName) || {};
        let originalGetter = originalDescriptor.get;
        let originalSetter = originalDescriptor.set;
        let originalValue = originalDescriptor.value;

        Object.defineProperty(current, parts[parts.length - 1], {
          configurable: true,
          enumerable: true,
          get() {
            abort();

            return originalGetter ? originalGetter.call(this) : originalValue;
          },
          set(newValue) {
            abort();

            if (originalSetter) {
              originalSetter.call(this, newValue);
            } else {
              originalValue = newValue;
            }

            // idk yet if thats needed
            // if (originalSetter) {
            //   originalSetter.call(current, v);
            // } else {
            //   // Avoid infinite recursion by directly setting the property on the target.
            //   Object.defineProperty(current, propName, {
            //     configurable: true,
            //     enumerable: true,
            //     writable: true,
            //     value: v,
            //   });
            // }
          },
        });
      } else {
        // intermediate property in chain
        const isConfigurable = isPropertyConfigurable(current, part);
        const propExists = Object.prototype.hasOwnProperty.call(current, part);

        if (isConfigurable && !propExists) {
          let internalValue: any;
          Object.defineProperty(current, part, {
            configurable: true,
            enumerable: true,
            get() {
              if (internalValue === undefined) {
                internalValue = createProxy(chainSoFar);
              }
              return internalValue;
            },
            set(newValue) {
              internalValue = newValue;
            },
          });
        }

        // Move into the next level of the chain.
        current = current[part];
      }
    }
  }

  defineProxyChain(window, property);

  // Enhance error handling for the thrown ReferenceError
  window.addEventListener('error', (event) => {
    if (event.error instanceof ReferenceError && event.error.message.includes(rid)) {
      event.preventDefault();
    }
  });
}

function generateRandomId(): string {
  return Math.random().toString(36).substring(2, 15);
}
