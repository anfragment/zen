import { expect, test, describe, afterEach } from '@jest/globals';

import { abortCurrentInlineScript } from './abort-current-inline-script';

describe('abort-current-inline-script', () => {
  afterEach(() => {
    delete (window as any).PROPERTY;
    delete (window as any).test;
  });

  test('test', () => {
    (window.test as any).prop = '123';
    abortCurrentInlineScript('test.prop');

    expect(() => {
      try {
        // (window.test as any).prop;
        (window.test as any).prop = '456';
      } catch (ex) {
        console.log(ex);
        throw ex;
      }
    }).toThrow();
  });
});
