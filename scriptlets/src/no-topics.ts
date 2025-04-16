import { createLogger } from './helpers/logger';

const logger = createLogger('no-topics');

export function noTopics() {
  const browerTopicsKey = 'browsingTopics';

  if (typeof Document !== 'function' || typeof Document.prototype !== 'object') {
    return;
  }

  const descriptor = Object.getOwnPropertyDescriptor(Document.prototype, browerTopicsKey);
  if (!descriptor || !descriptor.configurable || typeof descriptor.value !== 'function') {
    return;
  }

  const original = descriptor.value;

  const fakeFn = new Proxy(original, {
    apply: () => {
      const responseBody = '[]';
      const response = new Response(responseBody, {
        status: 200,
        headers: {
          'Content-Length': responseBody.length.toString(),
          'Content-Type': 'application/json',
          Date: new Date().toUTCString(),
        },
      });

      Object.defineProperties(response, {
        url: { value: '' },
        type: { value: 'basic' },
      });

      return Promise.resolve(response);
    },
    get(target, prop, receiver) {
      return Reflect.get(target, prop, receiver);
    },
    set() {
      return true;
    },
  });

  Object.defineProperty(Document.prototype, browerTopicsKey, {
    configurable: true,
    get: () => {
      return fakeFn;
    },
    set: () => {},
  });

  logger.info('Prevented Topics API usage');
}
