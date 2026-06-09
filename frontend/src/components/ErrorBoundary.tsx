import { Component, type ReactNode, type ErrorInfo } from "react";
import logger from "../lib/logger";

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error?: Error;
}

export default class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    logger.error(`ErrorBoundary caught: ${error.message}\n${info.componentStack}`);
  }

  render() {
    if (this.state.hasError) {
      return (
        this.props.fallback || (
          <div className="p-6 text-center">
            <p className="text-red-400 mb-2">页面出错了</p>
            <p className="text-gray-400 text-xs">{this.state.error?.message}</p>
            <button
              onClick={() => this.setState({ hasError: false, error: undefined })}
              className="mt-3 px-3 py-1 text-sm bg-blue-600 rounded hover:bg-blue-500"
            >
              重试
            </button>
          </div>
        )
      );
    }
    return this.props.children;
  }
}
