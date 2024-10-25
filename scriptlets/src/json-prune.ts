import { createLogger } from './helpers/logger';
import { matchStack } from './helpers/matchStack';
import { parseRegexpFromString, parseRegexpLiteral } from './helpers/parseRegexp';

const logger = createLogger('json-prune');

export function jsonPrune(propsToRemove: string, requiredProps?: string, stack?: string): void {
  propsToRemove = propsToRemove.trim();
  if (typeof propsToRemove !== 'string' || propsToRemove.length === 0) {
    logger.warn('propsToRemove should be a non-empty string');
    return;
  }

  const parsedPropsToRemove = parsePropPaths(propsToRemove);
  const parsedRequiredProps = parsePropPaths(requiredProps);
  let stackRe: RegExp | null;
  if (stack !== undefined && stack !== '') {
    stackRe = parseRegexpLiteral(stack) || parseRegexpFromString(stack);
  }
  stackRe ??= null;

  const prune = (obj: any) => {
    if (stackRe && !matchStack(stackRe)) {
      return;
    }

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

function prunePath(obj: any, path: PropPath) {
  if (path.length === 0) {
    return;
  }

  const [segment, ...rest] = path;

  if (segment === '*') {
    const objProps = Object.getOwnPropertyNames(obj).filter(
      (prop) => typeof obj[prop] === 'object' || Array.isArray(obj[prop]),
    );
    for (const prop of objProps) {
      prunePath(obj[prop], path);
      prunePath(obj[prop], rest);
    }
    return;
  }
  if (segment === '[]') {
    if (!Array.isArray(obj)) {
      return;
    }
    for (let i = 0; i < obj.length; i++) {
      prunePath(obj[i], rest);
    }
    return;
  }

  if (Object.hasOwn(obj, segment)) {
    if (rest.length === 0) {
      delete obj[segment];
    } else {
      prunePath(obj[segment], rest);
    }
  }
}

function matchesPath(obj: any, path: PropPath): boolean {
  if (path.length === 0) {
    return true;
  }

  if (path[0] === '*') {
    const objProps = Object.getOwnPropertyNames(obj).filter(
      (prop) => typeof obj[prop] === 'object' || Array.isArray(obj[prop]),
    );
    for (const prop of objProps) {
      if (matchesPath(obj[prop], path) || matchesPath(obj[prop], path.slice(1))) {
        return true;
      }
    }
    return false;
  }
  if (path[0] === '[]') {
    if (!Array.isArray(obj)) {
      return false;
    }
    for (let i = 0; i < obj.length; i++) {
      if (matchesPath(obj[i], path.slice(1))) {
        return true;
      }
    }
    return false;
  }

  if (Object.hasOwn(obj, path[0])) {
    return matchesPath(obj[path[0]], path.slice(1));
  }
  return false;
}

function parsePropPaths(propPaths?: string): PropPath[] {
  if (typeof propPaths !== 'string') {
    return [];
  }

  return propPaths
    .split(/\s+/)
    .filter(Boolean)
    .map((prop) => prop.split('.'));
}

type PropPath = string[];
