import { ApiRequestError } from "./mvpApi";

export type AdminRole = "admin" | "operator" | "support";

export type AdminLockerStatus =
  | "free"
  | "reserved"
  | "occupied"
  | "locked"
  | "open"
  | "maintenance"
  | "out_of_service";

export type AdminLockerSize = "S" | "M" | "L" | "XL";

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
  token?: string;
};

type EnvelopeResult<TData, TMeta = unknown> = {
  data: TData;
  meta?: TMeta;
  requestId?: string;
};

type QueryValue = string | number | boolean | Array<string | number | boolean> | null | undefined;

const API_BASE_URL =
  (import.meta.env.VITE_API_BASE_URL as string | undefined)?.trim().replace(/\/+$/, "") ?? "";
const ADMIN_API_PREFIX = `${API_BASE_URL}/api/v1/admin`;

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

  if (status === 401) {
    return "Unauthorized";
  }

  if (status === 403) {
    return "Forbidden";
  }

  return `Request failed with status ${status}`;
};

const buildHeaders = (init: RequestInit | undefined, token: string | undefined) => {
  const headers = new Headers(init?.headers);

  if (!headers.has("Accept")) {
    headers.set("Accept", "application/json");
  }

  if (token && !headers.has("Authorization")) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  if (init?.body !== undefined && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json; charset=utf-8");
  }

  return headers;
};

const toQueryString = (params: Record<string, QueryValue>) => {
  const query = new URLSearchParams();

  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === "") {
      continue;
    }

    if (Array.isArray(value)) {
      for (const item of value) {
        query.append(key, String(item));
      }
      continue;
    }

    query.set(key, String(value));
  }

  const queryString = query.toString();

  return queryString ? `?${queryString}` : "";
};

const requestJson = async <TData, TMeta = unknown>(
  path: string,
  init?: RequestInit,
  options: RequestOptions = {},
): Promise<EnvelopeResult<TData, TMeta>> => {
  const response = await fetch(`${ADMIN_API_PREFIX}${path}`, {
    ...init,
    signal: options.signal,
    headers: buildHeaders(init, options.token),
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

  if (isSuccessPayload<TData, TMeta>(payload)) {
    return {
      data: payload.data,
      meta: payload.meta,
      requestId: payload.requestId,
    };
  }

  // Allow legacy plain JSON responses during transition.
  return {
    data: payload as TData,
  };
};

const parseFileNameFromDisposition = (contentDisposition: string | null) => {
  if (!contentDisposition) {
    return null;
  }

  const utfMatch = contentDisposition.match(/filename\*=UTF-8''([^;]+)/i);
  if (utfMatch?.[1]) {
    return decodeURIComponent(utfMatch[1]);
  }

  const simpleMatch = contentDisposition.match(/filename="?([^";]+)"?/i);
  if (simpleMatch?.[1]) {
    return simpleMatch[1];
  }

  return null;
};

const requestBlob = async (
  path: string,
  init?: RequestInit,
  options: RequestOptions = {},
): Promise<{ blob: Blob; fileName: string | null }> => {
  const response = await fetch(`${ADMIN_API_PREFIX}${path}`, {
    ...init,
    signal: options.signal,
    headers: buildHeaders(init, options.token),
  });

  if (!response.ok) {
    const payload = (await parseJson(response)) as ApiResponse<unknown> | unknown;

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

  const blob = await response.blob();
  const fileName = parseFileNameFromDisposition(response.headers.get("Content-Disposition"));

  return {
    blob,
    fileName,
  };
};

export type AdminMe = {
  id: number;
  login: string;
  role: AdminRole;
  isActive: boolean;
};

export type AdminLoginResult = {
  accessToken: string;
  tokenType: string;
  expiresIn: number;
  admin: {
    id: number;
    login: string;
    role: AdminRole;
  } | null;
};

export type AdminLocationSummary = {
  locationId: number;
  name: string;
  address: string;
  isActive: boolean;
  cellsTotal: number;
  cellsByStatus: Partial<Record<AdminLockerStatus, number>>;
  updatedAt: string;
};

export type AdminLockerListItem = {
  lockerId: number;
  lockerNo: number;
  size: AdminLockerSize;
  status: AdminLockerStatus;
  isActive: boolean;
  price: number;
  hardwareId: string | null;
  lastEventAt: number | null;
  updatedAt: number | string;
};

export type AdminLockerEvent = {
  id: number;
  eventType: string;
  payload: Record<string, unknown>;
  createdAt: number | string;
};

export type AdminLockerDetails = {
  locker: {
    lockerId: number;
    locationId: number;
    lockerNo: number;
    size: AdminLockerSize;
    status: AdminLockerStatus;
    isActive: boolean;
    price: number;
    hardwareId: string | null;
  };
  activeRental: {
    rentalId: string;
    state: string;
    phoneMasked: string;
    openedAt: string;
    finishedAt: string | null;
  } | null;
  lastPayment: {
    paymentId: string;
    status: string;
    amount: number;
    currency: string;
    paidAt: string | null;
  } | null;
  recentEvents: AdminLockerEvent[];
};

export type AdminSessionSummary = {
  sessionId: number;
  lockerId: number;
  lockerNo: number;
  locationId: number;
  phoneMasked: string;
  status: string;
  startedAt: number | string;
  paidUntil: number | string | null;
  closedAt: number | string | null;
};

export type GetAdminLocationsParams = {
  search?: string;
  isActive?: boolean;
  limit?: number;
  offset?: number;
};

export type GetAdminLockersParams = {
  status?: AdminLockerStatus[];
  size?: AdminLockerSize[];
  isActive?: boolean;
  limit?: number;
  offset?: number;
};

export type GetAdminSessionsParams = {
  locationId?: number;
  lockerId?: number;
  status?: string[];
  phone?: string;
  from?: string;
  to?: string;
  limit?: number;
  offset?: number;
};

export type RevenueExportParams = {
  from: string;
  to: string;
  locationId?: number;
  groupBy?: "location" | "day";
  tz?: string;
};

const login = async (
  credentials: { login: string; password: string },
  options: RequestOptions = {},
): Promise<AdminLoginResult> => {
  const result = await requestJson<{
    accessToken?: string;
    token?: string;
    tokenType?: string;
    expiresIn?: number;
    admin?: {
      id: number;
      login: string;
      role: AdminRole;
    };
  }>(
    "/login",
    {
      method: "POST",
      body: JSON.stringify(credentials),
    },
    options,
  );

  const token = result.data.accessToken ?? result.data.token;

  if (!token) {
    throw new ApiRequestError("Backend did not return access token", {
      status: 200,
    });
  }

  return {
    accessToken: token,
    tokenType: result.data.tokenType ?? "Bearer",
    expiresIn: result.data.expiresIn ?? 3600,
    admin: result.data.admin ?? null,
  };
};

const getMe = async (options: RequestOptions): Promise<AdminMe> => {
  const result = await requestJson<AdminMe>("/me", { method: "GET" }, options);
  return result.data;
};

const getLocations = async (
  params: GetAdminLocationsParams,
  options: RequestOptions,
): Promise<{ items: AdminLocationSummary[]; total: number }> => {
  const path = `/locations${toQueryString({
    search: params.search,
    isActive: params.isActive,
    limit: params.limit,
    offset: params.offset,
  })}`;

  const result = await requestJson<AdminLocationSummary[], { total?: number }>(
    path,
    { method: "GET" },
    options,
  );

  return {
    items: result.data,
    total: result.meta?.total ?? result.data.length,
  };
};

const getLocationLockers = async (
  locationId: number,
  params: GetAdminLockersParams,
  options: RequestOptions,
): Promise<{ items: AdminLockerListItem[]; total: number }> => {
  const path = `/locations/${locationId}/lockers${toQueryString({
    status: params.status,
    size: params.size,
    isActive: params.isActive,
    limit: params.limit,
    offset: params.offset,
  })}`;

  const result = await requestJson<AdminLockerListItem[], { total?: number }>(
    path,
    { method: "GET" },
    options,
  );

  return {
    items: result.data,
    total: result.meta?.total ?? result.data.length,
  };
};

const getLockerDetails = async (
  lockerId: number,
  options: RequestOptions,
): Promise<AdminLockerDetails> => {
  const result = await requestJson<AdminLockerDetails>(
    `/lockers/${lockerId}`,
    { method: "GET" },
    options,
  );
  return result.data;
};

const updateLockerStatus = async (
  lockerId: number,
  body: {
    status: AdminLockerStatus;
    reason?: string;
  },
  options: RequestOptions,
) => {
  const result = await requestJson<{ lockerId: number; previousStatus: string; newStatus: string }>(
    `/lockers/${lockerId}/status`,
    {
      method: "PATCH",
      body: JSON.stringify(body),
    },
    options,
  );

  return result.data;
};

const manualOpenLocker = async (
  lockerId: number,
  body: { reason?: string },
  options: RequestOptions,
) => {
  const result = await requestJson<{ lockerId: number; commandId: number; status: string }>(
    `/lockers/${lockerId}/open`,
    {
      method: "POST",
      body: JSON.stringify(body),
    },
    options,
  );

  return result.data;
};

const getSessions = async (
  params: GetAdminSessionsParams,
  options: RequestOptions,
): Promise<{ items: AdminSessionSummary[]; total: number }> => {
  const path = `/sessions${toQueryString({
    locationId: params.locationId,
    lockerId: params.lockerId,
    status: params.status,
    phone: params.phone,
    from: params.from,
    to: params.to,
    limit: params.limit,
    offset: params.offset,
  })}`;

  const result = await requestJson<AdminSessionSummary[], { total?: number }>(
    path,
    { method: "GET" },
    options,
  );

  return {
    items: result.data,
    total: result.meta?.total ?? result.data.length,
  };
};

const exportRevenue = async (
  params: RevenueExportParams,
  options: RequestOptions,
): Promise<{ blob: Blob; fileName: string }> => {
  const path = `/revenue/export${toQueryString({
    from: params.from,
    to: params.to,
    locationId: params.locationId,
    groupBy: params.groupBy,
    tz: params.tz,
  })}`;

  const result = await requestBlob(path, { method: "GET" }, options);

  return {
    blob: result.blob,
    fileName: result.fileName ?? `revenue_${params.from}_${params.to}.xlsx`,
  };
};

export const adminApi = {
  login,
  getMe,
  getLocations,
  getLocationLockers,
  getLockerDetails,
  updateLockerStatus,
  manualOpenLocker,
  getSessions,
  exportRevenue,
};
