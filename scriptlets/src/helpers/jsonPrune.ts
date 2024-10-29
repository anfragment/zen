type PropPath = string[];

export function prunePath(obj: any, path: PropPath) {
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

export function matchesPath(obj: any, path: PropPath): boolean {
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

export function parsePropPaths(propPaths?: string): PropPath[] {
  if (typeof propPaths !== 'string') {
    return [];
  }

  return propPaths
    .split(/\s+/)
    .filter(Boolean)
    .map((prop) => prop.split('.'));
}
