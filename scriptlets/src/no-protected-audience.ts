import { createLogger } from './helpers/logger';

const logger = createLogger('no-protected-audience');

export function noProtectedAudience() {
  logger.warn('not implemented');
}
