import { Fragment, useCallback, useEffect, useMemo, useState, type FormEvent } from "react";
import Button from "../../components/Button/Button";
import Input from "../../components/Input/Input";
import Logo from "../../components/Logo/Logo";
import {
  adminApi,
  type AdminLockerDetails,
  type AdminLockerSize,
  type AdminLockerStatus,
  type AdminLocationSummary,
  type AdminMe,
  type AdminSessionSummary,
} from "../../shared/api/adminApi";
import { ApiRequestError } from "../../shared/api/mvpApi";
import styles from "./Admin.module.css";

const ADMIN_TOKEN_STORAGE_KEY = "lockit_admin_access_token";
const ADMIN_PROFILE_STORAGE_KEY = "lockit_admin_profile";

const LOCKER_STATUS_OPTIONS: AdminLockerStatus[] = [
  "free",
  "open",
  "occupied",
  "maintenance",
  "out_of_service",
];

const LOCKER_SIZE_OPTIONS: AdminLockerSize[] = ["S", "M", "L", "XL"];

const SESSION_STATUS_OPTIONS = [
  "all",
  "created",
  "waiting_payment",
  "paid",
  "active",
  "closed",
  "expired",
  "cancelled",
  "error",
] as const;

type LocationActiveFilter = "all" | "active" | "inactive";
type LockerStatusFilter = "all" | AdminLockerStatus;
type LockerSizeFilter = "all" | AdminLockerSize;
type SessionStatusFilter = (typeof SESSION_STATUS_OPTIONS)[number];

const statusClassMap: Record<AdminLockerStatus, string> = {
  free: styles["status--free"],
  reserved: styles["status--reserved"],
  occupied: styles["status--occupied"],
  locked: styles["status--locked"],
  open: styles["status--open"],
  maintenance: styles["status--maintenance"],
  out_of_service: styles["status--out_of_service"],
};

const getErrorStatus = (error: unknown) => {
  if (error instanceof ApiRequestError) {
    return error.status;
  }

  return null;
};

const getErrorMessage = (error: unknown, fallbackMessage: string) => {
  if (error instanceof ApiRequestError && error.message) {
    return error.message;
  }

  if (error instanceof Error && error.message) {
    return error.message;
  }

  return fallbackMessage;
};

const getCellCount = (location: AdminLocationSummary, status: AdminLockerStatus) =>
  location.cellsByStatus[status] ?? 0;

const formatDateTime = (value: string | number | null | undefined) => {
  if (value === null || value === undefined || value === "" || value === 0) {
    return "-";
  }

  const date =
    typeof value === "number" ? new Date(value * 1000) : new Date(value.replace(" ", "T"));

  if (Number.isNaN(date.getTime())) {
    return "-";
  }

  return date.toLocaleString("ru-RU");
};

const formatMoney = (amount: number | null | undefined, currency = "RUB") => {
  if (amount === null || amount === undefined) {
    return "-";
  }

  const normalizedCurrency = currency.toUpperCase();
  const symbol = normalizedCurrency === "RUB" ? "RUB" : normalizedCurrency;

  return `${amount} ${symbol}`;
};

const createDateInputValue = (offsetDays: number) => {
  const date = new Date();
  date.setDate(date.getDate() + offsetDays);
  return date.toISOString().slice(0, 10);
};

const parseStoredProfile = (): AdminMe | null => {
  if (typeof window === "undefined") {
    return null;
  }

  const raw = window.localStorage.getItem(ADMIN_PROFILE_STORAGE_KEY);

  if (!raw) {
    return null;
  }

  try {
    const parsed = JSON.parse(raw) as AdminMe;

    if (
      typeof parsed.id === "number" &&
      typeof parsed.login === "string" &&
      typeof parsed.role === "string" &&
      typeof parsed.isActive === "boolean"
    ) {
      return parsed;
    }

    return null;
  } catch {
    return null;
  }
};

const saveStoredProfile = (profile: AdminMe | null) => {
  if (typeof window === "undefined") {
    return;
  }

  if (!profile) {
    window.localStorage.removeItem(ADMIN_PROFILE_STORAGE_KEY);
    return;
  }

  window.localStorage.setItem(ADMIN_PROFILE_STORAGE_KEY, JSON.stringify(profile));
};

function Admin() {
  const [authToken, setAuthToken] = useState<string | null>(() => {
    if (typeof window === "undefined") {
      return null;
    }

    return window.localStorage.getItem(ADMIN_TOKEN_STORAGE_KEY);
  });
  const [adminProfile, setAdminProfile] = useState<AdminMe | null>(() => parseStoredProfile());
  const [isAuthChecking, setIsAuthChecking] = useState(() => Boolean(authToken));
  const [authMessage, setAuthMessage] = useState<string | null>(null);
  const [loginInput, setLoginInput] = useState("");
  const [passwordInput, setPasswordInput] = useState("");
  const [isLoginPending, setIsLoginPending] = useState(false);

  const [locationSearchInput, setLocationSearchInput] = useState("");
  const [locationSearch, setLocationSearch] = useState("");
  const [locationActiveFilter, setLocationActiveFilter] = useState<LocationActiveFilter>("all");
  const [locations, setLocations] = useState<AdminLocationSummary[]>([]);
  const [isLocationsLoading, setIsLocationsLoading] = useState(false);
  const [locationsError, setLocationsError] = useState<string | null>(null);
  const [selectedLocationId, setSelectedLocationId] = useState<number | null>(null);

  const [lockerStatusFilter, setLockerStatusFilter] = useState<LockerStatusFilter>("all");
  const [lockerSizeFilter, setLockerSizeFilter] = useState<LockerSizeFilter>("all");
  const [lockers, setLockers] = useState<
    Array<{
      lockerId: number;
      lockerNo: number;
      size: AdminLockerSize;
      status: AdminLockerStatus;
      isActive: boolean;
      price: number;
      hardwareId: string | null;
      lastEventAt: number | null;
      updatedAt: number | string;
    }>
  >([]);
  const [isLockersLoading, setIsLockersLoading] = useState(false);
  const [lockersError, setLockersError] = useState<string | null>(null);
  const [selectedLockerId, setSelectedLockerId] = useState<number | null>(null);

  const [lockerDetails, setLockerDetails] = useState<AdminLockerDetails | null>(null);
  const [isLockerDetailsLoading, setIsLockerDetailsLoading] = useState(false);
  const [lockerDetailsError, setLockerDetailsError] = useState<string | null>(null);
  const [statusDraft, setStatusDraft] = useState<AdminLockerStatus>("maintenance");
  const [statusReason, setStatusReason] = useState("");
  const [isStatusUpdatePending, setIsStatusUpdatePending] = useState(false);
  const [isManualOpenPending, setIsManualOpenPending] = useState(false);
  const [lockerActionMessage, setLockerActionMessage] = useState<string | null>(null);

  const [sessions, setSessions] = useState<AdminSessionSummary[]>([]);
  const [sessionStatusFilter, setSessionStatusFilter] = useState<SessionStatusFilter>("all");
  const [isSessionsLoading, setIsSessionsLoading] = useState(false);
  const [sessionsError, setSessionsError] = useState<string | null>(null);

  const [revenueFrom, setRevenueFrom] = useState(() => createDateInputValue(-7));
  const [revenueTo, setRevenueTo] = useState(() => createDateInputValue(0));
  const [isRevenueExportPending, setIsRevenueExportPending] = useState(false);
  const [revenueMessage, setRevenueMessage] = useState<string | null>(null);

  const [reloadKey, setReloadKey] = useState(0);

  const selectedLocation = useMemo(
    () => locations.find((location) => location.locationId === selectedLocationId) ?? null,
    [locations, selectedLocationId],
  );

  const clearAuth = useCallback((message?: string) => {
    if (typeof window !== "undefined") {
      window.localStorage.removeItem(ADMIN_TOKEN_STORAGE_KEY);
    }

    saveStoredProfile(null);

    setAuthToken(null);
    setAdminProfile(null);
    setAuthMessage(message ?? null);
    setLocations([]);
    setSelectedLocationId(null);
    setLockers([]);
    setSelectedLockerId(null);
    setLockerDetails(null);
    setSessions([]);
  }, []);

  useEffect(() => {
    if (!authToken) {
      setIsAuthChecking(false);
      return;
    }

    let isDisposed = false;
    const controller = new AbortController();

    const loadProfile = async () => {
      setIsAuthChecking(true);

      try {
        const profile = await adminApi.getMe({
          token: authToken,
          signal: controller.signal,
        });

        if (isDisposed) {
          return;
        }

        setAdminProfile(profile);
        saveStoredProfile(profile);
        setAuthMessage(null);
      } catch (error) {
        if (isDisposed || (error as Error).name === "AbortError") {
          return;
        }

        const status = getErrorStatus(error);

        if (status === 401 || status === 403) {
          clearAuth("Сессия администратора истекла. Выполните вход заново.");
          return;
        }

        if (status === 404) {
          const cachedProfile = parseStoredProfile();

          if (cachedProfile) {
            setAdminProfile(cachedProfile);
            setAuthMessage("Endpoint /admin/me пока не готов. Используется сохраненный профиль.");
          } else {
            setAuthMessage(
              "Endpoint /admin/me пока не готов. Вход возможен, если backend вернет профиль в /admin/login.",
            );
          }

          return;
        }

        setAuthMessage(getErrorMessage(error, "Не удалось проверить текущую админ-сессию"));
      } finally {
        if (!isDisposed) {
          setIsAuthChecking(false);
        }
      }
    };

    void loadProfile();

    return () => {
      isDisposed = true;
      controller.abort();
    };
  }, [authToken, clearAuth]);

  useEffect(() => {
    if (!authToken || !adminProfile) {
      return;
    }

    let isDisposed = false;
    const controller = new AbortController();

    const loadLocations = async () => {
      setIsLocationsLoading(true);
      setLocationsError(null);

      try {
        const result = await adminApi.getLocations(
          {
            search: locationSearch || undefined,
            isActive:
              locationActiveFilter === "all" ? undefined : locationActiveFilter === "active",
            limit: 200,
            offset: 0,
          },
          {
            token: authToken,
            signal: controller.signal,
          },
        );

        if (isDisposed) {
          return;
        }

        setLocations(result.items);
        setSelectedLocationId((current) => {
          if (result.items.length === 0) {
            return null;
          }

          if (current && result.items.some((item) => item.locationId === current)) {
            return current;
          }

          return result.items[0].locationId;
        });
      } catch (error) {
        if (isDisposed || (error as Error).name === "AbortError") {
          return;
        }

        const status = getErrorStatus(error);

        if (status === 401 || status === 403) {
          clearAuth("Сессия администратора истекла. Выполните вход заново.");
          return;
        }

        setLocationsError(getErrorMessage(error, "Не удалось загрузить список камер хранения"));
      } finally {
        if (!isDisposed) {
          setIsLocationsLoading(false);
        }
      }
    };

    void loadLocations();

    return () => {
      isDisposed = true;
      controller.abort();
    };
  }, [adminProfile, authToken, clearAuth, locationActiveFilter, locationSearch, reloadKey]);

  useEffect(() => {
    if (!authToken || !adminProfile || !selectedLocationId) {
      setLockers([]);
      setSelectedLockerId(null);
      return;
    }

    let isDisposed = false;
    const controller = new AbortController();

    const loadLockers = async () => {
      setIsLockersLoading(true);
      setLockersError(null);

      try {
        const result = await adminApi.getLocationLockers(
          selectedLocationId,
          {
            status: lockerStatusFilter === "all" ? undefined : [lockerStatusFilter],
            size: lockerSizeFilter === "all" ? undefined : [lockerSizeFilter],
            limit: 500,
            offset: 0,
          },
          {
            token: authToken,
            signal: controller.signal,
          },
        );

        if (isDisposed) {
          return;
        }

        setLockers(result.items);
        setSelectedLockerId((current) => {
          if (result.items.length === 0) {
            return null;
          }

          if (current && result.items.some((item) => item.lockerId === current)) {
            return current;
          }

          return result.items[0].lockerId;
        });
      } catch (error) {
        if (isDisposed || (error as Error).name === "AbortError") {
          return;
        }

        const status = getErrorStatus(error);

        if (status === 401 || status === 403) {
          clearAuth("Сессия администратора истекла. Выполните вход заново.");
          return;
        }

        setLockersError(getErrorMessage(error, "Не удалось загрузить список ячеек"));
      } finally {
        if (!isDisposed) {
          setIsLockersLoading(false);
        }
      }
    };

    void loadLockers();

    return () => {
      isDisposed = true;
      controller.abort();
    };
  }, [
    adminProfile,
    authToken,
    clearAuth,
    lockerSizeFilter,
    lockerStatusFilter,
    reloadKey,
    selectedLocationId,
  ]);

  useEffect(() => {
    if (!authToken || !adminProfile || !selectedLockerId) {
      setLockerDetails(null);
      return;
    }

    let isDisposed = false;
    const controller = new AbortController();

    const loadLockerDetails = async () => {
      setIsLockerDetailsLoading(true);
      setLockerDetailsError(null);

      try {
        const result = await adminApi.getLockerDetails(selectedLockerId, {
          token: authToken,
          signal: controller.signal,
        });

        if (isDisposed) {
          return;
        }

        setLockerDetails(result);
        setStatusDraft(result.locker.status);
      } catch (error) {
        if (isDisposed || (error as Error).name === "AbortError") {
          return;
        }

        const status = getErrorStatus(error);

        if (status === 401 || status === 403) {
          clearAuth("Сессия администратора истекла. Выполните вход заново.");
          return;
        }

        setLockerDetailsError(getErrorMessage(error, "Не удалось загрузить карточку ячейки"));
      } finally {
        if (!isDisposed) {
          setIsLockerDetailsLoading(false);
        }
      }
    };

    void loadLockerDetails();

    return () => {
      isDisposed = true;
      controller.abort();
    };
  }, [adminProfile, authToken, clearAuth, reloadKey, selectedLockerId]);

  useEffect(() => {
    if (!authToken || !adminProfile) {
      setSessions([]);
      return;
    }

    let isDisposed = false;
    const controller = new AbortController();

    const loadSessions = async () => {
      setIsSessionsLoading(true);
      setSessionsError(null);

      try {
        const result = await adminApi.getSessions(
          {
            locationId: selectedLocationId ?? undefined,
            status: sessionStatusFilter === "all" ? undefined : [sessionStatusFilter],
            limit: 50,
            offset: 0,
          },
          {
            token: authToken,
            signal: controller.signal,
          },
        );

        if (isDisposed) {
          return;
        }

        setSessions(result.items);
      } catch (error) {
        if (isDisposed || (error as Error).name === "AbortError") {
          return;
        }

        const status = getErrorStatus(error);

        if (status === 401 || status === 403) {
          clearAuth("Сессия администратора истекла. Выполните вход заново.");
          return;
        }

        setSessionsError(getErrorMessage(error, "Не удалось загрузить сессии"));
      } finally {
        if (!isDisposed) {
          setIsSessionsLoading(false);
        }
      }
    };

    void loadSessions();

    return () => {
      isDisposed = true;
      controller.abort();
    };
  }, [adminProfile, authToken, clearAuth, reloadKey, selectedLocationId, sessionStatusFilter]);

  const handleLoginSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    const normalizedLogin = loginInput.trim();

    if (!normalizedLogin || !passwordInput) {
      setAuthMessage("Введите логин и пароль");
      return;
    }

    setIsLoginPending(true);
    setAuthMessage(null);

    try {
      const result = await adminApi.login({
        login: normalizedLogin,
        password: passwordInput,
      });

      if (typeof window !== "undefined") {
        window.localStorage.setItem(ADMIN_TOKEN_STORAGE_KEY, result.accessToken);
      }

      setAuthToken(result.accessToken);

      const profileFromLogin = result.admin
        ? {
            id: result.admin.id,
            login: result.admin.login,
            role: result.admin.role,
            isActive: true,
          }
        : {
            id: 0,
            login: normalizedLogin,
            role: "admin" as const,
            isActive: true,
          };

      setAdminProfile(profileFromLogin);
      saveStoredProfile(profileFromLogin);

      if (!result.admin) {
        setAuthMessage(
          "Backend вернул токен без профиля. Используется временный профиль до реализации /admin/me.",
        );
      }

      setLoginInput("");
      setPasswordInput("");
    } catch (error) {
      setAuthMessage(getErrorMessage(error, "Не удалось выполнить вход"));
    } finally {
      setIsLoginPending(false);
    }
  };

  const handleLocationSearch = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setLocationSearch(locationSearchInput.trim());
  };

  const handleUpdateLockerStatus = async () => {
    if (!authToken || !selectedLockerId) {
      return;
    }

    setIsStatusUpdatePending(true);
    setLockerActionMessage(null);

    try {
      await adminApi.updateLockerStatus(
        selectedLockerId,
        {
          status: statusDraft,
          reason: statusReason.trim() || undefined,
        },
        { token: authToken },
      );

      setLockerActionMessage("Статус ячейки обновлен");
      setStatusReason("");
      setReloadKey((value) => value + 1);
    } catch (error) {
      const status = getErrorStatus(error);

      if (status === 401 || status === 403) {
        clearAuth("Сессия администратора истекла. Выполните вход заново.");
        return;
      }

      setLockerActionMessage(getErrorMessage(error, "Не удалось изменить статус ячейки"));
    } finally {
      setIsStatusUpdatePending(false);
    }
  };

  const handleManualOpen = async () => {
    if (!authToken || !selectedLockerId) {
      return;
    }

    setIsManualOpenPending(true);
    setLockerActionMessage(null);

    try {
      const result = await adminApi.manualOpenLocker(
        selectedLockerId,
        {
          reason: statusReason.trim() || undefined,
        },
        { token: authToken },
      );

      setLockerActionMessage(`Команда отправлена: #${result.commandId}`);
      setReloadKey((value) => value + 1);
    } catch (error) {
      const status = getErrorStatus(error);

      if (status === 401 || status === 403) {
        clearAuth("Сессия администратора истекла. Выполните вход заново.");
        return;
      }

      setLockerActionMessage(getErrorMessage(error, "Не удалось отправить команду открытия"));
    } finally {
      setIsManualOpenPending(false);
    }
  };

  const handleExportRevenue = async () => {
    if (!authToken) {
      return;
    }

    if (!revenueFrom || !revenueTo || revenueFrom > revenueTo) {
      setRevenueMessage("Проверьте корректность диапазона дат");
      return;
    }

    setIsRevenueExportPending(true);
    setRevenueMessage(null);

    try {
      const result = await adminApi.exportRevenue(
        {
          from: revenueFrom,
          to: revenueTo,
          locationId: selectedLocationId ?? undefined,
          groupBy: "location",
          tz: "UTC",
        },
        { token: authToken },
      );

      const objectUrl = window.URL.createObjectURL(result.blob);
      const link = document.createElement("a");
      link.href = objectUrl;
      link.download = result.fileName;
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(objectUrl);

      setRevenueMessage(`Файл ${result.fileName} скачан`);
    } catch (error) {
      const status = getErrorStatus(error);

      if (status === 401 || status === 403) {
        clearAuth("Сессия администратора истекла. Выполните вход заново.");
        return;
      }

      setRevenueMessage(getErrorMessage(error, "Не удалось сформировать Excel отчет"));
    } finally {
      setIsRevenueExportPending(false);
    }
  };

  const handleLogout = () => {
    clearAuth("Вы вышли из админ-панели");
  };

  if (isAuthChecking) {
    return (
      <main className={styles.page}>
        <section className={styles.loginShell}>
          <Logo className={styles.logo} />
          <h1 className={styles.loginTitle}>Проверка админ-сессии</h1>
          <p className={styles.mutedText}>Подождите, загружаем данные администратора...</p>
        </section>
      </main>
    );
  }

  if (!authToken || !adminProfile) {
    return (
      <main className={styles.page}>
        <section className={styles.loginShell}>
          <Logo className={styles.logo} />
          <h1 className={styles.loginTitle}>Админ-панель LOCK&apos;IT</h1>

          <form className={styles.loginForm} onSubmit={handleLoginSubmit}>
            <Input
              variant="compact"
              placeholder="Логин"
              value={loginInput}
              onChange={(event) => setLoginInput(event.target.value)}
              disabled={isLoginPending}
            />

            <Input
              variant="compact"
              type="password"
              placeholder="Пароль"
              value={passwordInput}
              onChange={(event) => setPasswordInput(event.target.value)}
              disabled={isLoginPending}
            />

            <Button variant="compact" type="submit" disabled={isLoginPending}>
              {isLoginPending ? "Входим..." : "Войти"}
            </Button>
          </form>

          {authMessage && <p className={styles.errorText}>{authMessage}</p>}
        </section>
      </main>
    );
  }

  return (
    <main className={styles.page}>
      <header className={styles.header}>
        <div className={styles.headerLeft}>
          <Logo size="compact" />
          <span className={styles.adminBadge}>
            {adminProfile.login} ({adminProfile.role})
          </span>
        </div>

        <div className={styles.headerActions}>
          <Button variant="compact" onClick={handleLogout}>
            Выйти
          </Button>
        </div>
      </header>

      {authMessage && <p className={styles.infoText}>{authMessage}</p>}

      <section className={styles.dashboard}>
        <article className={styles.panel}>
          <h2 className={styles.panelTitle}>Камеры хранения</h2>

          <form className={styles.inlineForm} onSubmit={handleLocationSearch}>
            <Input
              variant="compact"
              placeholder="Поиск по адресу или имени"
              value={locationSearchInput}
              onChange={(event) => setLocationSearchInput(event.target.value)}
            />
            <Button variant="compact" type="submit">
              Найти
            </Button>
          </form>

          <div className={styles.controlRow}>
            <label className={styles.fieldLabel} htmlFor="location-active-filter">
              Активность:
            </label>
            <select
              id="location-active-filter"
              className={styles.select}
              value={locationActiveFilter}
              onChange={(event) =>
                setLocationActiveFilter(event.target.value as LocationActiveFilter)
              }
            >
              <option value="all">Все</option>
              <option value="active">Активные</option>
              <option value="inactive">Неактивные</option>
            </select>
          </div>

          {isLocationsLoading && <p className={styles.loadingText}>Загружаем камеры хранения...</p>}
          {locationsError && <p className={styles.errorText}>{locationsError}</p>}

          {!isLocationsLoading && !locationsError && (
            <ul className={styles.locationList}>
              {locations.map((location) => {
                const isSelected = selectedLocationId === location.locationId;

                return (
                  <li key={location.locationId} className={styles.locationItem}>
                    <button
                      type="button"
                      className={`${styles.locationButton} ${
                        isSelected ? styles.locationButtonActive : ""
                      }`}
                      onClick={() => setSelectedLocationId(location.locationId)}
                    >
                      <div className={styles.locationMain}>
                        <h3 className={styles.locationName}>{location.name}</h3>
                        <p className={styles.locationAddress}>{location.address}</p>
                        <p className={styles.mutedText}>Всего ячеек: {location.cellsTotal}</p>
                      </div>

                      <div className={styles.locationStats}>
                        <span className={styles.statItem}>
                          Free: {getCellCount(location, "free")}
                        </span>
                        <span className={styles.statItem}>
                          Occupied: {getCellCount(location, "occupied")}
                        </span>
                        <span className={styles.statItem}>
                          Maintenance: {getCellCount(location, "maintenance")}
                        </span>
                      </div>
                    </button>
                  </li>
                );
              })}
            </ul>
          )}

          {!isLocationsLoading && !locationsError && locations.length === 0 && (
            <p className={styles.emptyState}>По текущим фильтрам камер хранения нет</p>
          )}

          <div className={styles.revenueBlock}>
            <h3 className={styles.panelSubtitle}>Выгрузка выручки</h3>
            <div className={styles.dateRow}>
              <label className={styles.fieldLabel} htmlFor="revenue-from">
                С:
              </label>
              <input
                id="revenue-from"
                className={styles.dateInput}
                type="date"
                value={revenueFrom}
                onChange={(event) => setRevenueFrom(event.target.value)}
              />
            </div>

            <div className={styles.dateRow}>
              <label className={styles.fieldLabel} htmlFor="revenue-to">
                По:
              </label>
              <input
                id="revenue-to"
                className={styles.dateInput}
                type="date"
                value={revenueTo}
                onChange={(event) => setRevenueTo(event.target.value)}
              />
            </div>

            <p className={styles.mutedText}>
              Локация: {selectedLocation ? selectedLocation.name : "Все локации"}
            </p>

            <Button
              variant="compact"
              className={styles.wideButton}
              onClick={handleExportRevenue}
              disabled={isRevenueExportPending}
            >
              {isRevenueExportPending ? "Формируем..." : "Скачать Excel"}
            </Button>

            {revenueMessage && <p className={styles.infoText}>{revenueMessage}</p>}
          </div>
        </article>

        <article className={styles.panel}>
          <h2 className={styles.panelTitle}>Ячейки в локации</h2>

          <div className={styles.controlRow}>
            <label className={styles.fieldLabel} htmlFor="locker-status-filter">
              Статус:
            </label>
            <select
              id="locker-status-filter"
              className={styles.select}
              value={lockerStatusFilter}
              onChange={(event) => setLockerStatusFilter(event.target.value as LockerStatusFilter)}
            >
              <option value="all">Все</option>
              {LOCKER_STATUS_OPTIONS.map((status) => (
                <option key={status} value={status}>
                  {status}
                </option>
              ))}
            </select>
          </div>

          <div className={styles.controlRow}>
            <label className={styles.fieldLabel} htmlFor="locker-size-filter">
              Размер:
            </label>
            <select
              id="locker-size-filter"
              className={styles.select}
              value={lockerSizeFilter}
              onChange={(event) => setLockerSizeFilter(event.target.value as LockerSizeFilter)}
            >
              <option value="all">Все</option>
              {LOCKER_SIZE_OPTIONS.map((size) => (
                <option key={size} value={size}>
                  {size}
                </option>
              ))}
            </select>
          </div>

          {isLockersLoading && <p className={styles.loadingText}>Загружаем ячейки...</p>}
          {lockersError && <p className={styles.errorText}>{lockersError}</p>}

          {!isLockersLoading && !lockersError && (
            <ul className={styles.lockerList}>
              {lockers.map((locker) => {
                const isSelected = locker.lockerId === selectedLockerId;

                return (
                  <li key={locker.lockerId}>
                    <button
                      type="button"
                      className={`${styles.lockerButton} ${
                        isSelected ? styles.lockerButtonActive : ""
                      }`}
                      onClick={() => setSelectedLockerId(locker.lockerId)}
                    >
                      <span className={styles.lockerNo}>#{locker.lockerNo}</span>
                      <span className={styles.lockerSize}>{locker.size}</span>
                      <span className={`${styles.statusChip} ${statusClassMap[locker.status]}`}>
                        {locker.status}
                      </span>
                      <span className={styles.mutedText}>
                        {locker.isActive ? "active" : "inactive"}
                      </span>
                    </button>
                  </li>
                );
              })}
            </ul>
          )}

          {!isLockersLoading && !lockersError && lockers.length === 0 && (
            <p className={styles.emptyState}>Ячейки по текущим фильтрам не найдены</p>
          )}
        </article>

        <article className={styles.panel}>
          <h2 className={styles.panelTitle}>Карточка ячейки</h2>

          {!selectedLockerId && <p className={styles.emptyState}>Выберите ячейку в списке</p>}
          {selectedLockerId && isLockerDetailsLoading && (
            <p className={styles.loadingText}>Загружаем детали...</p>
          )}
          {selectedLockerId && lockerDetailsError && (
            <p className={styles.errorText}>{lockerDetailsError}</p>
          )}

          {selectedLockerId && !isLockerDetailsLoading && !lockerDetailsError && lockerDetails && (
            <>
              <div className={styles.detailGrid}>
                <div className={styles.detailRow}>
                  <span className={styles.detailLabel}>Ячейка</span>
                  <span className={styles.detailValue}>#{lockerDetails.locker.lockerNo}</span>
                </div>
                <div className={styles.detailRow}>
                  <span className={styles.detailLabel}>Размер</span>
                  <span className={styles.detailValue}>{lockerDetails.locker.size}</span>
                </div>
                <div className={styles.detailRow}>
                  <span className={styles.detailLabel}>Статус</span>
                  <span
                    className={`${styles.statusChip} ${statusClassMap[lockerDetails.locker.status]}`}
                  >
                    {lockerDetails.locker.status}
                  </span>
                </div>
                <div className={styles.detailRow}>
                  <span className={styles.detailLabel}>Hardware ID</span>
                  <span className={styles.detailValue}>
                    {lockerDetails.locker.hardwareId ?? "-"}
                  </span>
                </div>
                <div className={styles.detailRow}>
                  <span className={styles.detailLabel}>Цена</span>
                  <span className={styles.detailValue}>
                    {formatMoney(lockerDetails.locker.price)}
                  </span>
                </div>
              </div>

              <div className={styles.actionBlock}>
                <h3 className={styles.panelSubtitle}>Изменить статус</h3>
                <div className={styles.controlRow}>
                  <label className={styles.fieldLabel} htmlFor="status-draft">
                    Новый статус:
                  </label>
                  <select
                    id="status-draft"
                    className={styles.select}
                    value={statusDraft}
                    onChange={(event) => setStatusDraft(event.target.value as AdminLockerStatus)}
                  >
                    {LOCKER_STATUS_OPTIONS.map((status) => (
                      <option key={status} value={status}>
                        {status}
                      </option>
                    ))}
                  </select>
                </div>

                <textarea
                  className={styles.textarea}
                  placeholder="Причина (опционально)"
                  value={statusReason}
                  onChange={(event) => setStatusReason(event.target.value)}
                />

                <Button
                  variant="compact"
                  className={styles.wideButton}
                  onClick={handleUpdateLockerStatus}
                  disabled={isStatusUpdatePending}
                >
                  {isStatusUpdatePending ? "Сохраняем..." : "Обновить статус"}
                </Button>

                <Button
                  variant="compact"
                  className={styles.wideButton}
                  onClick={handleManualOpen}
                  disabled={isManualOpenPending}
                >
                  {isManualOpenPending ? "Отправляем..." : "Ручное открытие"}
                </Button>

                {lockerActionMessage && <p className={styles.infoText}>{lockerActionMessage}</p>}
              </div>

              <div className={styles.actionBlock}>
                <h3 className={styles.panelSubtitle}>Активная аренда</h3>
                {lockerDetails.activeRental ? (
                  <div className={styles.detailGrid}>
                    <div className={styles.detailRow}>
                      <span className={styles.detailLabel}>Rental ID</span>
                      <span className={styles.detailValue}>
                        {lockerDetails.activeRental.rentalId}
                      </span>
                    </div>
                    <div className={styles.detailRow}>
                      <span className={styles.detailLabel}>Статус</span>
                      <span className={styles.detailValue}>{lockerDetails.activeRental.state}</span>
                    </div>
                    <div className={styles.detailRow}>
                      <span className={styles.detailLabel}>Телефон</span>
                      <span className={styles.detailValue}>
                        {lockerDetails.activeRental.phoneMasked}
                      </span>
                    </div>
                    <div className={styles.detailRow}>
                      <span className={styles.detailLabel}>Открыта</span>
                      <span className={styles.detailValue}>
                        {formatDateTime(lockerDetails.activeRental.openedAt)}
                      </span>
                    </div>
                  </div>
                ) : (
                  <p className={styles.emptyState}>Активной аренды нет</p>
                )}
              </div>

              <div className={styles.actionBlock}>
                <h3 className={styles.panelSubtitle}>Последний платеж</h3>
                {lockerDetails.lastPayment ? (
                  <div className={styles.detailGrid}>
                    <div className={styles.detailRow}>
                      <span className={styles.detailLabel}>Payment ID</span>
                      <span className={styles.detailValue}>
                        {lockerDetails.lastPayment.paymentId}
                      </span>
                    </div>
                    <div className={styles.detailRow}>
                      <span className={styles.detailLabel}>Статус</span>
                      <span className={styles.detailValue}>{lockerDetails.lastPayment.status}</span>
                    </div>
                    <div className={styles.detailRow}>
                      <span className={styles.detailLabel}>Сумма</span>
                      <span className={styles.detailValue}>
                        {formatMoney(
                          lockerDetails.lastPayment.amount,
                          lockerDetails.lastPayment.currency,
                        )}
                      </span>
                    </div>
                    <div className={styles.detailRow}>
                      <span className={styles.detailLabel}>Оплачен</span>
                      <span className={styles.detailValue}>
                        {formatDateTime(lockerDetails.lastPayment.paidAt)}
                      </span>
                    </div>
                  </div>
                ) : (
                  <p className={styles.emptyState}>Платежей не найдено</p>
                )}
              </div>

              <div className={styles.actionBlock}>
                <h3 className={styles.panelSubtitle}>Последние события</h3>
                {lockerDetails.recentEvents.length > 0 ? (
                  <ul className={styles.eventsList}>
                    {lockerDetails.recentEvents.map((event) => (
                      <li key={event.id} className={styles.eventItem}>
                        <span className={styles.eventType}>{event.eventType}</span>
                        <span className={styles.eventDate}>{formatDateTime(event.createdAt)}</span>
                      </li>
                    ))}
                  </ul>
                ) : (
                  <p className={styles.emptyState}>Событий пока нет</p>
                )}
              </div>
            </>
          )}
        </article>
      </section>

      <section className={`${styles.panel} ${styles.sessionsPanel}`}>
        <div className={styles.sessionFilterRow}>
          <h2 className={styles.panelTitle}>Сессии</h2>
          <select
            className={styles.select}
            value={sessionStatusFilter}
            onChange={(event) => setSessionStatusFilter(event.target.value as SessionStatusFilter)}
          >
            {SESSION_STATUS_OPTIONS.map((status) => (
              <option key={status} value={status}>
                {status}
              </option>
            ))}
          </select>
        </div>

        {isSessionsLoading && <p className={styles.loadingText}>Загружаем сессии...</p>}
        {sessionsError && <p className={styles.errorText}>{sessionsError}</p>}

        {!isSessionsLoading && !sessionsError && sessions.length > 0 && (
          <div className={styles.sessionGrid}>
            <div className={styles.sessionGridHeader}>Session</div>
            <div className={styles.sessionGridHeader}>Ячейка</div>
            <div className={styles.sessionGridHeader}>Телефон</div>
            <div className={styles.sessionGridHeader}>Статус</div>
            <div className={styles.sessionGridHeader}>Начало</div>
            <div className={styles.sessionGridHeader}>Закрыта</div>

            {sessions.map((session) => (
              <Fragment key={session.sessionId}>
                <div className={styles.sessionGridRow}>{session.sessionId}</div>
                <div className={styles.sessionGridRow}>#{session.lockerNo}</div>
                <div className={styles.sessionGridRow}>{session.phoneMasked}</div>
                <div className={styles.sessionGridRow}>{session.status}</div>
                <div className={styles.sessionGridRow}>{formatDateTime(session.startedAt)}</div>
                <div className={styles.sessionGridRow}>{formatDateTime(session.closedAt)}</div>
              </Fragment>
            ))}
          </div>
        )}

        {!isSessionsLoading && !sessionsError && sessions.length === 0 && (
          <p className={styles.emptyState}>Сессии по фильтрам не найдены</p>
        )}
      </section>
    </main>
  );
}

export default Admin;
