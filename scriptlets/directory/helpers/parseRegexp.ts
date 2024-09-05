const regexpRegexp = /^\/((?:[^\/\\\r\n]|\\.)+)\/([gimsuy]*)$/;

/**
 * Parses a string into a RegExp.
 * @param input The string to parse.
 * @returns The RegExp if the input is a valid RegExp string, null otherwise.
 */
export function parseRegexp(input: string): RegExp | null {
  const match = input.match(regexpRegexp);
  if (match === null) {
    return null;
  }

  return new RegExp(match[1], match[2]);
}