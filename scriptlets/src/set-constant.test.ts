import { expect, jest, test, describe, beforeEach, afterEach } from '@jest/globals';

import { setConstant } from './set-constant';

describe('set-constant', () => {
  let originalWindow: typeof window;

  beforeEach(() => {
    originalWindow = window;
  });

  afterEach(() => {
    window = originalWindow;
  });

  test('test', () => {
    (window as any).test1 = {};
    setConstant('test1.test2', '123');
    setConstant('test1.test3', '321');
    setConstant('test1.test4.test7', '56');
    setConstant('test1.test4.test5', '516');

    expect((window as any).test1.test2).toEqual('123');
    expect((window as any).test1.test3).toEqual('321');
    expect((window as any).test1.test4.test5).toEqual('516');
    expect((window as any).test1.test4.test7).toEqual('56');
  });
});
