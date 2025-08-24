import { err, ok, type Result } from "neverthrow";
import { log } from "./logger";

// ArgoCD API error structure from swagger
export interface ArgoApiError {
  code?: number;
  error?: string;
  details?: any[];
}

// Our standardized error interface
export interface ApiError {
  message: string;
  code?: number;
  type: "network" | "auth" | "forbidden" | "not-found" | "server" | "unknown";
  originalError?: any;
}

// Helper to classify and translate API errors into user-friendly messages
export function translateApiError(error: any): ApiError {
  // Handle network/connection errors
  if (
    error?.message?.includes("ECONNREFUSED") ||
    error?.message?.includes("ENOTFOUND")
  ) {
    return {
      message:
        "Cannot connect to ArgoCD server. Please check if the server is running and accessible.",
      type: "network",
      originalError: error,
    };
  }

  // Handle timeout errors
  if (
    error?.message?.includes("timeout") ||
    error?.message?.includes("ETIMEDOUT")
  ) {
    return {
      message:
        "Request timed out. The ArgoCD server may be slow or unreachable.",
      type: "network",
      originalError: error,
    };
  }

  // Handle ArgoCD API errors
  if (error?.status) {
    const status = error.status;
    const argoError: ArgoApiError = error.data;

    switch (status) {
      case 401:
        return {
          message:
            'Authentication failed. Please run "argocd login" to authenticate.',
          code: status,
          type: "auth",
          originalError: error,
        };

      case 403:
        return {
          message:
            "Access denied. You don't have permission to perform this action.",
          code: status,
          type: "forbidden",
          originalError: error,
        };

      case 404:
        return {
          message:
            "Resource not found. It may have been deleted or you may not have access to it.",
          code: status,
          type: "not-found",
          originalError: error,
        };

      case 409:
        return {
          message:
            argoError?.error ||
            "Resource conflict. The resource may have been modified by someone else.",
          code: status,
          type: "server",
          originalError: error,
        };

      case 422:
        return {
          message:
            argoError?.error ||
            "Invalid request data. Please check your input.",
          code: status,
          type: "server",
          originalError: error,
        };

      case 500:
      case 502:
      case 503:
      case 504:
        return {
          message:
            "ArgoCD server error. The server may be experiencing issues.",
          code: status,
          type: "server",
          originalError: error,
        };

      default:
        return {
          message: argoError?.error || `Server returned error ${status}`,
          code: status,
          type: "server",
          originalError: error,
        };
    }
  }

  // Handle unknown errors
  return {
    message: error?.message || "An unexpected error occurred",
    type: "unknown",
    originalError: error,
  };
}

// Wrapper that returns Promise<Result<T, ApiError>>
export async function wrapApiCall<T>(
  apiCall: () => Promise<T>,
): Promise<Result<T, ApiError>> {
  try {
    const result = await apiCall();
    return ok(result);
  } catch (error) {
    const apiError = translateApiError(error);
    log.error(apiError.message);
    return err(apiError);
  }
}

// Helper to get user-friendly error message for UI display
export function getDisplayMessage(apiError: ApiError): string {
  return apiError.message;
}

// Helper to determine if error requires user action (like re-authentication)
export function requiresUserAction(apiError: ApiError): boolean {
  return apiError.type === "auth";
}
