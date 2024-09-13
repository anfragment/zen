export function setConstant(property: string, value: string) {
  const rootChain = property.split('.');

  const get = (chain: string[]) => (target: any, key: any) => {
    if (chain[0] !== key) {
      return target[key];
    }
    if (chain.length === 1) {
      return value;
    }

    const link = target[key];
    // if (link === undefined || link === null) {
    //   return link;
    // }
    return new Proxy(link ?? {}, {
      get: get(chain.slice(1)),
    });
  };

  const rootProp = rootChain.shift() as any;
  // window[rootProp as any] = new Proxy(window[rootProp as any] ?? {}, {
  //   get: get(rootChain),
  // });

  window[('_' + rootProp) as any] = window[rootProp];
  Object.defineProperty(window, rootProp, {
    get: () => {
      if (typeof window[('_' + rootProp) as any] !== 'object') {
        return window[('_' + rootProp) as any];
      }
      return new Proxy(window[('_' + rootProp) as any], {
        get: get(rootChain),
      });
    },
    set: (v) => {
      window[('_' + rootProp) as any] = v;
    },
  });
}
