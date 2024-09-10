import { Component, ErrorInfo, ReactNode } from 'react';

import { AppToaster } from './common/toaster';

interface Props {
  children: ReactNode;
}

class ErrorBoundary extends Component<Props> {
  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    AppToaster.show({
      message: `Unexpected error: ${error}`,
      intent: 'danger',
    });
    console.error('ErrorBoundary', error, errorInfo);
  }

  render() {
    // eslint-disable-next-line react/destructuring-assignment
    return this.props.children;
  }
}

export default ErrorBoundary;
