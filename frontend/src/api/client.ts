export interface FieldError {
  field: string;
  code: string;
}

export interface WebErrorResponse {
  code: string;
  fields?: FieldError[];
}

export class ApiError extends Error {
  public isApiError = true;
  public webError: WebErrorResponse;
  public status: number;

  constructor(webError: WebErrorResponse, status: number) {
    super(webError.code);
    this.name = 'ApiError';
    this.webError = webError;
    this.status = status;
  }
}

export const BASE_URL: string = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

/**
 * Attempts to read the response body as JSON.
 * Returns an empty object or undefined if parsing fails (e.g., non-JSON body).
 */
export async function readBody<T>(response: Response): Promise<T | undefined> {
  const contentLength = response.headers.get('content-length');
  if (response.status === 204 || contentLength === '0' || response.body === null) {
    return {} as T;
  }

  try {
    return await response.json() as T;
  } catch {
    return undefined;
  }
}

/**
 * Executes a fetch request with custom configurations, handling structured WebErrors.
 * @throws {ApiError} for recognized 4xx/5xx responses with a structured JSON body.
 * @throws {Error} for network failures or unstructured server errors.
 */
export async function apiFetch<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const url = `${BASE_URL}${endpoint}`;
  const requestOptions: RequestInit = {
    ...options,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
  };

  const response = await fetch(url, requestOptions);

  if (response.ok) {
    const body = await readBody<T>(response);
    return body as T;
  }

  const webErrorData = await readBody<WebErrorResponse>(response);
  if (webErrorData?.code) {
    throw new ApiError(webErrorData, response.status);
  }

  throw new Error(`API request failed with status ${response.status}. The server returned an unstructured error.`);
}
