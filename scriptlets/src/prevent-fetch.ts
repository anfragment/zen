type ResponseBody = 'emptyObj' | 'emptyArr' | 'emptyStr';

export function preventFetch(propsToMatch: string, responseBody: ResponseBody = 'emptyObj', responseType?: string) {
  if (typeof fetch === 'undefined' || typeof Proxy === 'undefined' || typeof Response === 'undefined') {
    return;
  }
}
