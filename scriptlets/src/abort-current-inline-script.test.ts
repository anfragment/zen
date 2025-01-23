import { expect, test, describe } from '@jest/globals';

import { abortCurrentInlineScript } from './abort-current-inline-script';

describe('abort-current-inline-script', () => {
  afterEach(() => {
    delete (window as any).PROPERTY;
    delete (window as any).test;
    delete (window as any).prop1;
  });

  test('test1', () => {
    abortCurrentInlineScript('prop1.prop2.prop3.prop4.prop5');

    expect(() => {
      (window as any).prop1.prop2.prop3.prop4.prop5;
    }).toThrowError(ReferenceError);
  });

  test('existing prop', () => {
    (window as any).prop1 = {};

    abortCurrentInlineScript('prop1.prop2');

    expect(() => {
      (window as any).prop1.prop2;
    }).toThrowError(ReferenceError);
  });

  test('test', () => {
    abortCurrentInlineScript('test');

    expect(() => {
      window.test as any;
    }).toThrowError(ReferenceError);
  });

  test('test.prop setter', () => {
    abortCurrentInlineScript('test.prop');

    expect(() => {
      (window.test as any).prop = '456';
    }).toThrowError(ReferenceError);
  });

  test('test getter', () => {
    abortCurrentInlineScript('test');

    expect(() => {
      (window as any).test;
    }).toThrowError(ReferenceError);
  });

  test('test.prop.prop2 getter', () => {
    (window as any).test = {
      prop: {
        prop2: () => {},
      },
    };

    abortCurrentInlineScript('test.prop.prop2');

    expect(() => {
      (window as any).test.prop.prop2;
    }).toThrowError(ReferenceError);
  });

  test('test.prop.prop2 setter', () => {
    abortCurrentInlineScript('test.prop.prop2');

    expect(() => {
      (window as any).test.prop.prop2 = '456';
    }).toThrowError(ReferenceError);
  });

  test.only('document.querySelectorAll', () => {
    abortCurrentInlineScript('document.querySelectorAll');

    expect(() => {
      window.document.querySelectorAll('test');
    }).toThrowError(ReferenceError);
  });

  // Think how to test this if currentScript is always jest
  test('getter does not abort non-regex values', () => {
    abortCurrentInlineScript('document.querySelectorAll', 'puHref');

    window.document.querySelectorAll('123');
  });
});
