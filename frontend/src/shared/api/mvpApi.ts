export type LockerSize = "s" | "m" | "l" | "xl";

type ApiErrorPayload = {
  code: string;
  message: string;
  details?: Record<string, unknown>;
};

type ApiSuccessResponse<TData, TMeta = unknown> = {
  ok: true;
  data: TData;
  meta?: TMeta;
  requestId?: string;
};

type ApiFailureResponse = {
  ok: false;
  error: ApiErrorPayload;
  requestId?: string;
};

type ApiResponse<TData, TMeta = unknown> = ApiSuccessResponse<TData, TMeta> | ApiFailureResponse;

type RequestOptions = {
  signal?: AbortSignal;
};

type EnvelopeResult<TData, TMeta = unknown> = {
  data: TData;
  meta?: TMeta;
  requestId?: string;
};

const API_BASE_URL =
  (import.meta.env.VITE_API_BASE_URL as string | undefined)?.trim().replace(/\/+$/, "") ?? "";
const API_PREFIX = `${API_BASE_URL}/api/v1`;

const isObject = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null;

const parseJson = async (response: Response): Promise<unknown> => {
  try {
    return await response.json();
  } catch {
    return null;
  }
};

const isFailurePayload = (value: unknown): value is ApiFailureResponse => {
  if (!isObject(value)) {
    return false;
  }

  if (value.ok !== false) {
    return false;
  }

  if (!isObject(value.error)) {
    return false;
  }

  return typeof value.error.code === "string" && typeof value.error.message === "string";
};

const isSuccessPayload = <TData, TMeta>(
  value: unknown,
): value is ApiSuccessResponse<TData, TMeta> => {
  if (!isObject(value)) {
    return false;
  }

  return value.ok === true && "data" in value;
};

const getMessageByStatus = (status: number) => {
  if (status >= 500) {
    return "Server error";
  }

  if (status === 404) {
    return "Resource not found";
  }

  if (status === 0) {
    return "Request was not sent";
  }

  return `Request failed with status ${status}`;
};

export class ApiRequestError extends Error {
  readonly status: number;
  readonly code?: string;
  readonly details?: Record<string, unknown>;
  readonly requestId?: string;

  constructor(
    message: string,
    options: {
      status: number;
      code?: string;
      details?: Record<string, unknown>;
      requestId?: string;
    },
  ) {
    super(message);
    this.name = "ApiRequestError";
    this.status = options.status;
    this.code = options.code;
    this.details = options.details;
    this.requestId = options.requestId;
  }
}

const buildHeaders = (init?: RequestInit) => {
  const headers = new Headers(init?.headers);

  if (!headers.has("Accept")) {
    headers.set("Accept", "application/json");
  }

  if (init?.body !== undefined && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json; charset=utf-8");
  }

  return headers;
};

const request = async <TData, TMeta = unknown>(
  path: string,
  init?: RequestInit,
): Promise<EnvelopeResult<TData, TMeta>> => {
  const response = await fetch(`${API_PREFIX}${path}`, {
    ...init,
    headers: buildHeaders(init),
  });

  const payload = (await parseJson(response)) as ApiResponse<TData, TMeta> | unknown;

  if (!response.ok) {
    if (isFailurePayload(payload)) {
      throw new ApiRequestError(payload.error.message, {
        status: response.status,
        code: payload.error.code,
        details: payload.error.details,
        requestId: payload.requestId,
      });
    }

    throw new ApiRequestError(getMessageByStatus(response.status), {
      status: response.status,
    });
  }

  if (isFailurePayload(payload)) {
    throw new ApiRequestError(payload.error.message, {
      status: response.status,
      code: payload.error.code,
      details: payload.error.details,
      requestId: payload.requestId,
    });
  }

  if (!isSuccessPayload<TData, TMeta>(payload)) {
    throw new ApiRequestError("Unexpected API response format", {
      status: response.status,
    });
  }

  return {
    data: payload.data,
    meta: payload.meta,
    requestId: payload.requestId,
  };
};

export type LockerListItem = {
  id: number;
  street: string;
  freeCells: {
    s: number;
    m: number;
    l: number;
    xl: number;
  };
  updatedAt: string;
};

export type GetLockersParams = {
  city?: string;
  limit?: number;
  offset?: number;
} & RequestOptions;

export type CellDimensions = {
  length: number;
  width: number;
  height: number;
  unit: "cm";
};

export type CreateCellSelectionRequest =
  | {
      size: LockerSize;
    }
  | {
      dimensions: CellDimensions;
    };

export type CellSelection = {
  selectionId: string;
  lockerId: number;
  size: LockerSize;
  cellNumber: number;
  holdExpiresAt: string;
};

export type Booking = {
  bookingId: string;
  rentalId: string;
  lockerId: number;
  cellNumber: number;
  phone: string;
  accessCode: string;
  state: string;
  openedAt: string;
};

export type PaymentSnapshot = {
  paymentId: string;
  amount: number;
  currency: string;
  status: string;
  qrPayload: string;
  paymentExpiresAt: string;
};

export type AccessCodeCheckResult = {
  rentalId: string;
  lockerId: number;
  cellNumber: number;
  phone: string;
  accessCode: string;
  paymentRequired: boolean;
  state: string;
  payment?: PaymentSnapshot;
};

export type PaymentStatus = {
  paymentId: string;
  status: string;
  amount: number;
  currency: string;
  paidAt: string | null;
};

export type OpenRentalResult = {
  rentalId: string;
  cellNumber: number;
  opened: boolean;
  openedAt: string;
};

export type FinishRentalResult = {
  rentalId: string;
  state: string;
  finishedAt: string;
};

export type RentalState = {
  bookingId: string;
  rentalId: string;
  lockerId: number;
  cellNumber: number;
  phone: string;
  accessCode: string;
  state: string;
  openedAt: string;
  finishedAt: string | null;
};

const getLockers = async (params: GetLockersParams = {}) => {
  const { city, limit, offset, signal } = params;
  const query = new URLSearchParams();

  if (city) {
    query.set("city", city);
  }

  if (typeof limit === "number") {
    query.set("limit", String(limit));
  }

  if (typeof offset === "number") {
    query.set("offset", String(offset));
  }

  const queryString = query.toString();
  const path = queryString ? `/lockers?${queryString}` : "/lockers";

  const result = await request<LockerListItem[], { total?: number }>(path, {
    method: "GET",
    signal,
  });

  return {
    items: result.data,
    total: result.meta?.total ?? result.data.length,
  };
};

const createCellSelection = async (
  lockerId: number,
  body: CreateCellSelectionRequest,
  options: RequestOptions = {},
) => {
  const result = await request<CellSelection>(`/lockers/${lockerId}/cell-selection`, {
    method: "POST",
    body: JSON.stringify(body),
    signal: options.signal,
  });

  return result.data;
};

const createBooking = async (
  lockerId: number,
  body: { selectionId: string; phone: string },
  options: RequestOptions = {},
) => {
  const result = await request<Booking>(`/lockers/${lockerId}/bookings`, {
    method: "POST",
    body: JSON.stringify(body),
    signal: options.signal,
  });

  return result.data;
};

const checkAccessCode = async (
  lockerId: number,
  body: { accessCode: string },
  options: RequestOptions = {},
) => {
  const result = await request<AccessCodeCheckResult>(`/lockers/${lockerId}/access-code/check`, {
    method: "POST",
    body: JSON.stringify(body),
    signal: options.signal,
  });

  return result.data;
};

const getPayment = async (paymentId: string, options: RequestOptions = {}) => {
  const result = await request<PaymentStatus>(`/payments/${paymentId}`, {
    method: "GET",
    signal: options.signal,
  });

  return result.data;
};

const openRental = async (rentalId: string, options: RequestOptions = {}) => {
  const result = await request<OpenRentalResult>(`/rentals/${rentalId}/open`, {
    method: "POST",
    signal: options.signal,
  });

  return result.data;
};

const finishRental = async (rentalId: string, options: RequestOptions = {}) => {
  const result = await request<FinishRentalResult>(`/rentals/${rentalId}/finish`, {
    method: "POST",
    signal: options.signal,
  });

  return result.data;
};

const getRental = async (rentalId: string, options: RequestOptions = {}) => {
  const result = await request<RentalState>(`/rentals/${rentalId}`, {
    method: "GET",
    signal: options.signal,
  });

  return result.data;
};

export const mvpApi = {
  getLockers,
  createCellSelection,
  createBooking,
  checkAccessCode,
  getPayment,
  openRental,
  finishRental,
  getRental,
};
