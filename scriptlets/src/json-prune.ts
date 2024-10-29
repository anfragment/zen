import { matchesPath, parsePropPaths, prunePath } from './helpers/jsonPrune';
import { createLogger } from './helpers/logger';
import { matchStack } from './helpers/matchStack';
import { parseRegexpFromString, parseRegexpLiteral } from './helpers/parseRegexp';

const logger = createLogger('json-prune');

export function jsonPrune(propsToRemove: string, requiredProps?: string, stack?: string): void {
  if (typeof Proxy === 'undefined') {
    logger.warn('Proxy not available in this environment');
    return;
  }

  if (typeof propsToRemove !== 'string' || propsToRemove.length === 0) {
    logger.warn('propsToRemove should be a non-empty string');
    return;
  }

  const parsedPropsToRemove = parsePropPaths(propsToRemove);
  const parsedRequiredProps = parsePropPaths(requiredProps);
  let stackRe: RegExp | null = null;
  if (typeof stack === 'string') {
    stackRe = parseRegexpLiteral(stack) || parseRegexpFromString(stack);
  }

  const prune = (obj: any) => {
    if (stackRe !== null && !matchStack(stackRe)) {
      return;
    }

    if (parsedRequiredProps.length > 0) {
      let matched = false;
      for (const propToMatch of parsedRequiredProps) {
        if (matchesPath(obj, propToMatch)) {
          matched = true;
          break;
        }
      }
      if (!matched) {
        return;
      }
    }

    for (const propToRemove of parsedPropsToRemove) {
      prunePath(obj, propToRemove);
    }
  };

  JSON.parse = new Proxy(JSON.parse, {
    apply: (target, thisArg, args) => {
      const obj = Reflect.apply(target, thisArg, args);
      prune(obj);
      return obj;
    },
  });

  if (typeof Response !== 'undefined') {
    Response.prototype.json = new Proxy(Response.prototype.json, {
      apply: (target, thisArg, args) => {
        const promise = Reflect.apply(target, thisArg, args);
        return promise.then((obj: any) => {
          prune(obj);
          return obj;
        });
      },
    });
  }
}
