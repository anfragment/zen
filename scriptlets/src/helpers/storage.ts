import { parseRegexpLiteral } from './parseRegexp';

const valueMap: Record<string, string> = {
  undefined: 'undefined',
  false: 'false',
  true: 'true',
  null: 'null',
  yes: 'yes',
  no: 'no',
  on: 'on',
  off: 'off',
  accept: 'accept',
  accepted: 'accepted',
  reject: 'reject',
  rejected: 'rejected',
  emptyArr: '[]',
  emptyObj: '{}',
  '': '',
  $remove$: '$remove$',
};

/**
 * Checks whether the value can be used in untrusted set-local-storage-item and set-session-storage-item scriptlets
 * and potentially converts it to a canonical form.
 * @param value The value to validate.
 * @returns The canonical value if it is valid.
 * @throws An error if the value is invalid.
 */
export function validateUntrustedStorageValue(value: string): string {
  if (valueMap[value] !== undefined) {
    return valueMap[value];
  }

  const int = parseInt(value);
  if (!isNaN(int) && int >= 0 && int <= 32767) {
    return value;
  }

  throw new Error('Invalid value');
}

export function removeFromStorage(storage: Storage, key: string): void {
  const regexp = parseRegexpLiteral(key);
  if (regexp !== null) {
    const storageKeys = Object.keys(storage);
    for (const storageKey of storageKeys) {
      if (regexp.test(storageKey)) {
        storage.removeItem(storageKey);
      }
    }
  } else {
    storage.removeItem(key);
  }
}
