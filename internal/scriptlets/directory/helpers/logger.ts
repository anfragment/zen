const PRODUCT_NAME = "Zen";

export function createLogger(scriptletName: string) {
  return function logger(line: string, ...context: any[]) {
    console.log(`${PRODUCT_NAME} (${scriptletName}): ${line}`, context);
  }
} 