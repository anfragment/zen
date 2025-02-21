import { abortCurrentInlineScript } from './abort-current-inline-script';
import { abortOnPropertyRead } from './abort-on-property-read';
import { abortOnPropertyWrite } from './abort-on-property-write';
import { abortOnStackTrace } from './abort-on-stack-trace';
import { createLogger } from './helpers/logger';
import { jsonPrune } from './json-prune';
import { jsonPruneFetchResponse } from './json-prune-fetch-response';
import { jsonPruneXHRResponse } from './json-prune-xhr-response';
import { nowebrtc } from './nowebrtc';
import { preventFetch } from './prevent-fetch';
import { preventWindowOpen } from './prevent-window-open';
import { preventXHR } from './prevent-xhr';
import { setConstant } from './set-constant';
import { setLocalStorageItem } from './set-local-storage-item';
import { setSessionStorageItem } from './set-session-storage-item';

const logger = createLogger('index');

export default function (name: string, ...args: string[]) {
  let fn;
  switch (name) {
    case 'prevent-fetch':
    case 'no-fetch-if':
      fn = preventFetch;
      break;
    case 'prevent-xhr':
    case 'no-xhr-if':
      fn = preventXHR;
      break;
    case 'nowebrtc':
      fn = nowebrtc;
      break;
    case 'set-local-storage-item':
      fn = setLocalStorageItem;
      break;
    case 'set-session-storage-item':
      fn = setSessionStorageItem;
      break;
    case 'set-constant':
      fn = setConstant;
      break;
    case 'json-prune':
      fn = jsonPrune;
      break;
    case 'json-prune-fetch-response':
      fn = jsonPruneFetchResponse;
      break;
    case 'json-prune-xhr-response':
      fn = jsonPruneXHRResponse;
      break;
    case 'abort-current-inline-script':
      fn = abortCurrentInlineScript;
      break;
    case 'abort-on-property-read':
    case 'aopr':
      fn = abortOnPropertyRead;
      break;
    case 'abort-on-property-write':
    case 'aopw':
      fn = abortOnPropertyWrite;
      break;
    case 'abort-on-stack-trace':
    case 'aost':
      fn = abortOnStackTrace;
      break;
    case 'prevent-window-open':
    case 'nowoif':
      fn = preventWindowOpen;
      break;
    default:
      logger.debug(`Unknown scriptlet: ${name}`);
      return;
  }

  //@ts-ignore
  fn(...args);
  return;
}
