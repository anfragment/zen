import { createPrune } from './helpers/jsonPrune';
import { createLogger } from './helpers/logger';
import { ParsedPropsToMatch, parsePropsToMatch } from './helpers/request';

const logger = createLogger('json-prune-xhr-response');

export function jsonPruneXHRResponse(
  propsToRemove: string,
  requiredProps?: string,
  propsToMatch?: string,
  stack?: string,
): void {
  if (typeof Proxy === 'undefined') {
    logger.warn('Proxy is not supported in this environment');
    return;
  }
  if (typeof propsToRemove !== 'string' || propsToRemove.length === 0) {
    logger.warn('propsToMatch cannot be empty');
    return;
  }

  let parsedProps: ParsedPropsToMatch;
  if (typeof propsToMatch === 'string') {
    try {
      parsedProps = parsePropsToMatch(propsToMatch);
    } catch (ex) {
      logger.warn('error parsing propsToMatch', ex);
      return;
    }
  }

  const prune = createPrune(propsToRemove, requiredProps, stack);
}
