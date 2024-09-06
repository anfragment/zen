import { expect, jest, it, describe, beforeEach, afterEach } from '@jest/globals';

import { preventXHR } from './prevent-xhr';

describe('preventXHR', () => {
  let originalOpen: typeof XMLHttpRequest.prototype.open;
  let originalSend: typeof XMLHttpRequest.prototype.send;
  let originalGetResponseHeader: typeof XMLHttpRequest.prototype.getResponseHeader;
  let originalGetAllResponseHeaders: typeof XMLHttpRequest.prototype.getAllResponseHeaders;
  let send: ReturnType<typeof jest.fn>;

  beforeEach(() => {
    originalOpen = XMLHttpRequest.prototype.open;
    originalSend = XMLHttpRequest.prototype.send;
    originalGetResponseHeader = XMLHttpRequest.prototype.getResponseHeader;
    originalGetAllResponseHeaders = XMLHttpRequest.prototype.getAllResponseHeaders;
    send = jest.fn();
    XMLHttpRequest.prototype.send = new Proxy(XMLHttpRequest.prototype.send, { apply: send as any });
  });

  afterEach(() => {
    XMLHttpRequest.prototype.open = originalOpen;
    XMLHttpRequest.prototype.send = originalSend;
    XMLHttpRequest.prototype.getResponseHeader = originalGetResponseHeader;
    XMLHttpRequest.prototype.getAllResponseHeaders = originalGetAllResponseHeaders;
    jest.clearAllMocks();
  });

  it("should prevent a request if called with '*'", () => {
    preventXHR('*');

    const xhr = new XMLHttpRequest();
    xhr.open('GET', 'example.org');
    xhr.send();

    expect(send).not.toHaveBeenCalled();
  });

  it("should prevent a request if called with ''", () => {
    preventXHR('');

    const xhr = new XMLHttpRequest();
    xhr.open('GET', 'example.org');
    xhr.send();

    expect(send).not.toHaveBeenCalled();
  });

  it('should prevent a request on matching domain name', () => {
    preventXHR('example.org');

    const xhr = new XMLHttpRequest();
    xhr.open('GET', 'example.org/test');
    xhr.send();

    expect(send).not.toHaveBeenCalled();
  });

  it('should prevent a request on matching domain name if url is fully qualified and contains port', () => {
    preventXHR('example.org');

    const xhr = new XMLHttpRequest();
    xhr.open('GET', 'https://example.org:443/test');
    xhr.send();

    expect(send).not.toHaveBeenCalled();
  });

  it('should not prevent a request non-matching domain name', () => {
    preventXHR('example.org');

    const xhr = new XMLHttpRequest();
    xhr.open('GET', 'example.net');
    xhr.send();

    expect(send).toHaveBeenCalled();
  });

  it('should prevent a request on a matching method', () => {
    preventXHR('method:GET');

    const xhr1 = new XMLHttpRequest();
    xhr1.open('GET', 'example.org');
    xhr1.send();

    expect(send).not.toHaveBeenCalled();
  });

  it('should not prevent a request on a non-matching method', () => {
    preventXHR('method:GET');

    const xhr = new XMLHttpRequest();
    xhr.open('POST', 'example.org');
    xhr.send();

    expect(send).toHaveBeenCalled();
  });

  it('should prevent a request on a matching domain name and method', () => {
    preventXHR('example.org method:GET');

    const xhr = new XMLHttpRequest();
    xhr.open('GET', 'example.org/');
    xhr.send();

    expect(send).not.toHaveBeenCalled();
  });

  it('should not prevent a request if only url gets matched', () => {
    preventXHR('example.org method:GET');

    const xhr = new XMLHttpRequest();
    xhr.open('POST', 'example.org');
    xhr.send();

    expect(send).toHaveBeenCalled();
  });
});
