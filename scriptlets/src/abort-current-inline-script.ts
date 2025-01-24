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
      logger.info(`Blocked ${property} in currentScript`);
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
          },
        });
      } else {
        const isConfigurable = isPropertyConfigurable(current, part);
        const propExists = Object.prototype.hasOwnProperty.call(current, part);

        if (isConfigurable && !propExists) {
          let internalValue: any;

          const createProxy = (target: AnyObject, chainParts: string[]) => {
            return new Proxy(target, {
              get(target, prop) {
                if (!(prop in target)) {
                  return undefined;
                }
                const value = Reflect.get(target, prop);
                if (chainParts.length === 1 && prop === chainParts[0]) {
                  abort();
                  return value;
                }
                if (prop === chainParts[0] && value instanceof Object) {
                  return createProxy(value, chainParts.slice(1));
                }
                return value;
              },
              set(target, prop, value) {
                if (chainParts.length === 1 && prop === chainParts[0]) {
                  abort();
                }
                return Reflect.set(target, prop, value);
              },
            });
          };

          Object.defineProperty(current, part, {
            configurable: true,
            enumerable: true,
            get() {
              if (internalValue == undefined) {
                return internalValue;
              } else if (internalValue instanceof Object) {
                return createProxy(internalValue, parts.slice(i + 1));
              } else {
                return internalValue;
              }
            },
            set(newValue) {
              internalValue = newValue;
            },
          });
          return;
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
