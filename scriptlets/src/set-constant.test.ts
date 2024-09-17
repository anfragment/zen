import { expect, test, describe, afterEach } from '@jest/globals';

import { setConstant } from './set-constant';

describe('set-constant', () => {
  afterEach(() => {
    delete (window as any).PROPERTY;
    delete (window as any).test;
  });

  test('sets a non-nested property', () => {
    setConstant('PROPERTY', 'yes');

    expect((window as any).PROPERTY).toBe('yes');
  });

  test('non-nested property survives an overwrite', () => {
    setConstant('PROPERTY', 'yes');

    (window as any).PROPERTY = 'no';
    expect((window as any).PROPERTY).toBe('yes');
  });

  test('sets multiple nested properties', () => {
    (window as any).test = {};
    setConstant('test.prop1', '123');
    setConstant('test.prop2', '321');
    setConstant('test.prop3.prop4', '56');
    setConstant('test.prop3.prop5', '516');

    expect((window as any).test.prop1).toBe('123');
    expect((window as any).test.prop2).toBe('321');
    expect((window as any).test.prop3.prop5).toBe('516');
    expect((window as any).test.prop3.prop4).toBe('56');
  });

  test('nested properties survive an overwrite', () => {
    (window as any).test = {};
    setConstant('test.prop1', '123');
    setConstant('test.prop2', '321');
    setConstant('test.prop3.prop4', '4');
    setConstant('test.prop3.prop5', '5');

    (window as any).test.prop1 = '';
    (window as any).test.prop2 = '';
    (window as any).test.prop3 = {};

    expect((window as any).test.prop1).toBe('123');
    expect((window as any).test.prop2).toBe('321');
    expect((window as any).test.prop3.prop4).toBe('4');
    expect((window as any).test.prop3.prop5).toBe('5');
  });

  test("doesn't modify the value if the stack doesn't match", () => {
    setConstant('PROPERTY', 'no', 'definitely-doesnt-match');

    expect((window as any).PROPERTY).toBe(undefined);
  });

  test('modifies the value if the stack does match', () => {
    setConstant('PROPERTY', 'yes', 'jest');

    expect((window as any).PROPERTY).toBe('yes');
  });

  test('wraps the value if valueWrapper is set', () => {
    setConstant('PROPERTY', 'yes', '', 'asFunction');

    expect(typeof (window as any).PROPERTY).toBe('function');
    expect((window as any).PROPERTY()).toBe('yes');
  });

  test('errors out on invalid value', () => {
    expect(() => {
      setConstant('PROPERTY', 'invalid');
    }).toThrow();
  });
});
