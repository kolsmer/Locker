import { useEffect, useMemo, useReducer, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import Button from "../../components/Button/Button";
import Input from "../../components/Input/Input";
import Logo from "../../components/Logo/Logo";
import {
  ApiRequestError,
  mvpApi,
  type AccessCodeCheckResult,
  type Booking,
  type CellSelection,
  type LockerSize,
} from "../../shared/api/mvpApi";
import styles from "./Locker.module.css";

type FlowStep =
  | "select-size"
  | "no-cells"
  | "phone-entry"
  | "access-code"
  | "active-rent"
  | "payment";

type Dimensions = {
  length: string;
  width: string;
  height: string;
};

type FlowState = {
  step: FlowStep;
  selectedSize: LockerSize | null;
  selectionId: string | null;
  currentCellNumber: number | null;
  dimensions: Dimensions;
  codeInput: string;
  phoneInput: string;
  confirmedPhone: string;
  issuedCode: string;
  currentRentalId: string | null;
  paymentId: string | null;
  paymentAmount: number | null;
  paymentCurrency: string;
  paymentQrPayload: string | null;
  formMessage: string | null;
};

type Action =
  | { type: "selection-created"; selection: CellSelection }
  | { type: "update-dimension"; field: keyof Dimensions; value: string }
  | { type: "update-code"; value: string }
  | { type: "update-phone"; value: string }
  | { type: "set-step"; step: FlowStep }
  | { type: "set-message"; message: string | null }
  | { type: "booking-created"; booking: Booking }
  | { type: "access-granted"; access: AccessCodeCheckResult }
  | { type: "payment-started"; access: AccessCodeCheckResult }
  | { type: "payment-completed" }
  | { type: "reset" };

const SIZE_OPTIONS: LockerSize[] = ["s", "m", "l", "xl"];
const PAYMENT_FALLBACK_TIMEOUT_MS = 5000;
const PAYMENT_POLL_INTERVAL_MS = 1500;

const SIZE_LABELS: Record<LockerSize, string> = {
  s: "S",
  m: "M",
  l: "L",
  xl: "XL",
};

const createInitialFlowState = (): FlowState => ({
  step: "select-size",
  selectedSize: null,
  selectionId: null,
  currentCellNumber: null,
  dimensions: {
    length: "",
    width: "",
    height: "",
  },
  codeInput: "",
  phoneInput: "",
  confirmedPhone: "",
  issuedCode: "",
  currentRentalId: null,
  paymentId: null,
  paymentAmount: null,
  paymentCurrency: "RUB",
  paymentQrPayload: null,
  formMessage: null,
});

const getRequestErrorMessage = (error: unknown, fallbackMessage: string) => {
  if (error instanceof ApiRequestError && error.message) {
    return error.message;
  }

  if (error instanceof Error && error.message) {
    return error.message;
  }

  return fallbackMessage;
};

const getRequestErrorCode = (error: unknown) => {
  if (error instanceof ApiRequestError && error.code) {
    return error.code;
  }

  return null;
};

const isPhoneValid = (phone: string) => {
  const digits = phone.replace(/\D/g, "");
  if (digits.length !== 11) {
    return false;
  }

  return digits.startsWith("7") || digits.startsWith("8");
};

const toDimensionNumber = (value: string) => Number(value.trim().replace(",", "."));

const lockerReducer = (state: FlowState, action: Action): FlowState => {
  switch (action.type) {
    case "selection-created": {
      return {
        ...state,
        step: "phone-entry",
        selectedSize: action.selection.size,
        selectionId: action.selection.selectionId,
        currentCellNumber: action.selection.cellNumber,
        formMessage: null,
      };
    }
    case "update-dimension": {
      return {
        ...state,
        dimensions: {
          ...state.dimensions,
          [action.field]: action.value,
        },
      };
    }
    case "update-code": {
      return {
        ...state,
        codeInput: action.value,
        formMessage: null,
      };
    }
    case "update-phone": {
      return {
        ...state,
        phoneInput: action.value,
        formMessage: null,
      };
    }
    case "set-step": {
      return {
        ...state,
        step: action.step,
        formMessage: null,
      };
    }
    case "set-message": {
      return {
        ...state,
        formMessage: action.message,
      };
    }
    case "booking-created": {
      return {
        ...state,
        step: "access-code",
        selectionId: null,
        currentCellNumber: action.booking.cellNumber,
        confirmedPhone: action.booking.phone,
        phoneInput: action.booking.phone,
        issuedCode: action.booking.accessCode,
        currentRentalId: action.booking.rentalId,
        paymentId: null,
        paymentAmount: null,
        paymentQrPayload: null,
        formMessage: null,
      };
    }
    case "access-granted": {
      return {
        ...state,
        step: "active-rent",
        selectionId: null,
        currentCellNumber: action.access.cellNumber,
        confirmedPhone: action.access.phone,
        issuedCode: action.access.accessCode,
        currentRentalId: action.access.rentalId,
        paymentId: null,
        paymentAmount: null,
        paymentQrPayload: null,
        formMessage: null,
      };
    }
    case "payment-started": {
      const payment = action.access.payment;

      return {
        ...state,
        step: "payment",
        selectionId: null,
        currentCellNumber: action.access.cellNumber,
        confirmedPhone: action.access.phone,
        issuedCode: action.access.accessCode,
        currentRentalId: action.access.rentalId,
        paymentId: payment?.paymentId ?? null,
        paymentAmount: payment?.amount ?? null,
        paymentCurrency: payment?.currency ?? "RUB",
        paymentQrPayload: payment?.qrPayload ?? null,
        formMessage: null,
      };
    }
    case "payment-completed": {
      return {
        ...state,
        step: "active-rent",
        paymentId: null,
        paymentAmount: null,
        paymentQrPayload: null,
        formMessage: null,
      };
    }
    case "reset": {
      return createInitialFlowState();
    }
    default: {
      return state;
    }
  }
};

function Locker() {
  const navigate = useNavigate();
  const { lockerId } = useParams<{ lockerId: string }>();

  const resolvedLockerId = useMemo(() => {
    const parsedLockerId = Number(lockerId);

    if (Number.isFinite(parsedLockerId) && parsedLockerId > 0) {
      return parsedLockerId;
    }

    return 123;
  }, [lockerId]);

  const [state, dispatch] = useReducer(lockerReducer, undefined, createInitialFlowState);
  const [dimensionsMessage, setDimensionsMessage] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [paymentStatusMessage, setPaymentStatusMessage] = useState("Проверяем оплату...");

  useEffect(() => {
    dispatch({ type: "reset" });
    setDimensionsMessage(null);
    setPaymentStatusMessage("Проверяем оплату...");
  }, [resolvedLockerId]);

  useEffect(() => {
    const paymentId = state.paymentId;
    const rentalId = state.currentRentalId;

    if (state.step !== "payment") {
      return;
    }

    setPaymentStatusMessage("Проверяем оплату...");

    let isDisposed = false;
    let isCompleted = false;
    let inFlight = false;
    let intervalId: number | null = null;

    const completePaymentStep = (message = "Проверяем оплату...") => {
      if (isDisposed || isCompleted) {
        return;
      }

      isCompleted = true;
      setPaymentStatusMessage(message);
      dispatch({ type: "payment-completed" });
    };

    const tryOpenRental = async () => {
      if (!rentalId) {
        return;
      }

      try {
        await mvpApi.openRental(rentalId);
      } catch {
        // In fallback mode payment may still be pending; UI should continue by timer.
      }
    };

    const pollPayment = async () => {
      if (isDisposed || isCompleted || inFlight || !paymentId) {
        return;
      }

      inFlight = true;

      try {
        const payment = await mvpApi.getPayment(paymentId);

        if (isDisposed || isCompleted) {
          return;
        }

        if (payment.status === "paid") {
          setPaymentStatusMessage("Оплата подтверждена, открываем ячейку...");
          await tryOpenRental();
          // Keep payment screen visible for the full fallback timeout.
          return;
        }

        if (payment.status === "failed") {
          setPaymentStatusMessage("Оплата не подтверждена, продолжаем по таймеру...");
          return;
        }

        if (payment.status === "expired") {
          setPaymentStatusMessage("Время оплаты истекло, продолжаем по таймеру...");
          return;
        }

        setPaymentStatusMessage("Проверяем оплату...");
      } catch {
        if (!isDisposed && !isCompleted) {
          setPaymentStatusMessage("Платежный endpoint в разработке, продолжаем по таймеру...");
        }
      } finally {
        inFlight = false;
      }
    };

    const timeoutId = window.setTimeout(() => {
      void (async () => {
        await tryOpenRental();
        completePaymentStep();
      })();
    }, PAYMENT_FALLBACK_TIMEOUT_MS);

    if (paymentId) {
      void pollPayment();
      intervalId = window.setInterval(() => {
        void pollPayment();
      }, PAYMENT_POLL_INTERVAL_MS);
    }

    return () => {
      isDisposed = true;
      window.clearTimeout(timeoutId);
      if (intervalId !== null) {
        window.clearInterval(intervalId);
      }
    };
  }, [state.currentRentalId, state.paymentId, state.step]);

  const runSelectionRequest = async (
    requestBody:
      | { size: LockerSize }
      | { dimensions: { length: number; width: number; height: number; unit: "cm" } },
    messageOnFailure: string,
    useDimensionsMessage = false,
  ) => {
    setIsSubmitting(true);
    dispatch({ type: "set-message", message: null });

    try {
      const selection = await mvpApi.createCellSelection(resolvedLockerId, requestBody);
      dispatch({ type: "selection-created", selection });
    } catch (error) {
      const errorCode = getRequestErrorCode(error);

      if (errorCode === "NO_CELLS_AVAILABLE") {
        dispatch({ type: "set-step", step: "no-cells" });
        return;
      }

      const message = getRequestErrorMessage(error, messageOnFailure);

      if (useDimensionsMessage) {
        setDimensionsMessage(message);
      } else {
        dispatch({ type: "set-message", message });
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleSizeSelect = (size: LockerSize) => {
    setDimensionsMessage(null);
    void runSelectionRequest({ size }, "Не удалось выбрать ячейку");
  };

  const handleContinueByDimensions = () => {
    const length = toDimensionNumber(state.dimensions.length);
    const width = toDimensionNumber(state.dimensions.width);
    const height = toDimensionNumber(state.dimensions.height);

    if (
      !Number.isFinite(length) ||
      !Number.isFinite(width) ||
      !Number.isFinite(height) ||
      length <= 0 ||
      width <= 0 ||
      height <= 0
    ) {
      setDimensionsMessage("Введите корректные габариты багажа");
      return;
    }

    setDimensionsMessage(null);

    void runSelectionRequest(
      {
        dimensions: {
          length,
          width,
          height,
          unit: "cm",
        },
      },
      "Не удалось подобрать ячейку",
      true,
    );
  };

  const handleCodeSubmit = async () => {
    const enteredCode = state.codeInput.trim().toUpperCase();

    if (!enteredCode) {
      dispatch({ type: "set-message", message: "Введите код доступа" });
      return;
    }

    setIsSubmitting(true);
    dispatch({ type: "set-message", message: null });

    try {
      const access = await mvpApi.checkAccessCode(resolvedLockerId, {
        accessCode: enteredCode,
      });

      if (access.paymentRequired) {
        if (!access.payment) {
          dispatch({ type: "set-message", message: "Не удалось получить информацию об оплате" });
          return;
        }

        setPaymentStatusMessage("Проверяем оплату...");
        dispatch({ type: "payment-started", access });
        return;
      }

      // Keep a consistent UX: show payment screen for the timer even when backend
      // already considers payment confirmed.
      setPaymentStatusMessage("Оплата подтверждена, открываем ячейку...");
      dispatch({ type: "payment-started", access });
    } catch (error) {
      const errorCode = getRequestErrorCode(error);

      if (errorCode === "INVALID_ACCESS_CODE") {
        dispatch({
          type: "set-message",
          message: "Код не найден или аренда уже завершена",
        });
        return;
      }

      dispatch({
        type: "set-message",
        message: getRequestErrorMessage(error, "Не удалось проверить код доступа"),
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handlePhoneSubmit = async () => {
    const phoneValue = state.phoneInput.trim();

    if (!isPhoneValid(phoneValue)) {
      dispatch({
        type: "set-message",
        message: "Введите корректный номер телефона",
      });
      return;
    }

    if (!state.selectionId) {
      dispatch({
        type: "set-message",
        message: "Бронь истекла, выберите размер заново",
      });
      return;
    }

    setIsSubmitting(true);
    dispatch({ type: "set-message", message: null });

    try {
      const booking = await mvpApi.createBooking(resolvedLockerId, {
        selectionId: state.selectionId,
        phone: phoneValue,
      });

      dispatch({ type: "booking-created", booking });
    } catch (error) {
      const errorCode = getRequestErrorCode(error);

      if (errorCode === "SELECTION_EXPIRED") {
        dispatch({ type: "reset" });
        dispatch({ type: "set-message", message: "Бронь истекла, выберите размер заново" });
        return;
      }

      dispatch({
        type: "set-message",
        message: getRequestErrorMessage(error, "Не удалось открыть ячейку"),
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleExit = () => {
    dispatch({ type: "reset" });
    setDimensionsMessage(null);
    setPaymentStatusMessage("Проверяем оплату...");
    navigate(`/locker/${resolvedLockerId}`, { replace: true });
  };

  const handleEndRent = async () => {
    if (!state.currentRentalId) {
      dispatch({ type: "reset" });
      return;
    }

    setIsSubmitting(true);
    dispatch({ type: "set-message", message: null });

    try {
      await mvpApi.finishRental(state.currentRentalId);
      dispatch({ type: "reset" });
      setDimensionsMessage(null);
      setPaymentStatusMessage("Проверяем оплату...");
    } catch (error) {
      dispatch({
        type: "set-message",
        message: getRequestErrorMessage(error, "Не удалось завершить аренду"),
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const displayedCellNumber = state.currentCellNumber ?? "-";

  const paymentAmountText =
    state.paymentAmount === null
      ? "-"
      : `${state.paymentAmount} ${state.paymentCurrency === "RUB" ? "Р" : state.paymentCurrency}`;

  const renderPhoneSummary = () => (
    <div className={styles.phoneSummary}>
      <p className={styles.label}>НОМЕР ТЕЛЕФОНА: {state.confirmedPhone}</p>
      <p className={styles.helperText}>Если это не ваш номер телефона, обратитесь в поддержку.</p>
    </div>
  );

  if (state.step === "no-cells") {
    return (
      <main className={styles.page}>
        <header className={styles.header}>
          <Logo size="compact" />
        </header>

        <section className={styles.noCellsState}>
          <p className={styles.noCellsText}>
            Свободных ячеек не осталось, попробуйте другой размер
          </p>
          <Button
            variant="compact"
            className={styles.primaryAction}
            onClick={() => dispatch({ type: "reset" })}
            disabled={isSubmitting}
          >
            Назад
          </Button>
        </section>
      </main>
    );
  }

  return (
    <main className={styles.page}>
      <header className={styles.header}>
        <Logo size="compact" />
      </header>

      <section className={styles.panel}>
        <div className={styles.leftColumn}>
          {state.step === "select-size" && (
            <div className={styles.formSection}>
              <h1 className={styles.sectionTitle}>ОСТАВИТЬ ВЕЩИ</h1>

              <p className={styles.subTitle}>Выберите размер ячейки:</p>

              <div className={styles.sizeRow}>
                {SIZE_OPTIONS.map((size) => {
                  const sizeClassName = [
                    styles.sizeButton,
                    size === "xl" ? styles.sizeButtonWide : "",
                    state.selectedSize === size ? styles.sizeButtonActive : "",
                  ]
                    .filter(Boolean)
                    .join(" ");

                  return (
                    <Button
                      key={size}
                      variant="compact"
                      className={sizeClassName}
                      disabled={isSubmitting}
                      onClick={() => handleSizeSelect(size)}
                    >
                      {SIZE_LABELS[size]}
                    </Button>
                  );
                })}
              </div>

              <div className={styles.divider} />

              <p className={styles.subTitle}>
                Не знаете, какая ячейка нужна?
                <br />
                Введите размеры багажа:
              </p>

              <div className={styles.dimensionsRow}>
                <Input
                  variant="compact"
                  className={styles.dimensionInput}
                  placeholder="Длина"
                  value={state.dimensions.length}
                  disabled={isSubmitting}
                  onChange={(event) =>
                    dispatch({
                      type: "update-dimension",
                      field: "length",
                      value: event.target.value,
                    })
                  }
                />
                <Input
                  variant="compact"
                  className={styles.dimensionInput}
                  placeholder="Ширина"
                  value={state.dimensions.width}
                  disabled={isSubmitting}
                  onChange={(event) =>
                    dispatch({
                      type: "update-dimension",
                      field: "width",
                      value: event.target.value,
                    })
                  }
                />
                <Input
                  variant="compact"
                  className={styles.dimensionInput}
                  placeholder="Высота"
                  value={state.dimensions.height}
                  disabled={isSubmitting}
                  onChange={(event) =>
                    dispatch({
                      type: "update-dimension",
                      field: "height",
                      value: event.target.value,
                    })
                  }
                />
              </div>

              <Button
                variant="compact"
                className={styles.primaryAction}
                disabled={isSubmitting}
                onClick={handleContinueByDimensions}
              >
                Далее
              </Button>

              {dimensionsMessage && <p className={styles.statusMessage}>{dimensionsMessage}</p>}
            </div>
          )}

          {state.step === "phone-entry" && (
            <div className={styles.formSectionTall}>
              <p className={styles.lockerTitle}>ЯЧЕЙКА #{displayedCellNumber}</p>

              <div className={styles.stackGap}>
                <Input
                  variant="compact"
                  className={styles.phoneInput}
                  placeholder="Номер телефона"
                  value={state.phoneInput}
                  disabled={isSubmitting}
                  onChange={(event) =>
                    dispatch({ type: "update-phone", value: event.target.value })
                  }
                />

                <Button
                  variant="compact"
                  className={styles.primaryAction}
                  disabled={isSubmitting}
                  onClick={handlePhoneSubmit}
                >
                  Открыть ячейку
                </Button>
                {state.formMessage && <p className={styles.statusMessage}>{state.formMessage}</p>}
              </div>

              <Button
                variant="compact"
                className={styles.secondaryAction}
                disabled={isSubmitting}
                onClick={handleExit}
              >
                Выйти
              </Button>
            </div>
          )}

          {state.step === "access-code" && (
            <div className={styles.formSectionTall}>
              <p className={styles.lockerTitle}>ЯЧЕЙКА #{displayedCellNumber}</p>

              {renderPhoneSummary()}

              <Button
                variant="compact"
                className={styles.secondaryAction}
                disabled={isSubmitting}
                onClick={handleExit}
              >
                Выйти
              </Button>

              {state.formMessage && <p className={styles.statusMessage}>{state.formMessage}</p>}
            </div>
          )}

          {state.step === "active-rent" && (
            <div className={styles.formSectionTall}>
              <p className={styles.lockerTitle}>ЯЧЕЙКА #{displayedCellNumber}</p>

              {renderPhoneSummary()}

              <Button
                variant="compact"
                className={styles.secondaryAction}
                disabled={isSubmitting}
                onClick={handleEndRent}
              >
                Завершить аренду
              </Button>

              {state.formMessage && <p className={styles.statusMessage}>{state.formMessage}</p>}
            </div>
          )}

          {state.step === "payment" && (
            <div className={styles.formSectionTall}>
              <p className={styles.lockerTitle}>ЯЧЕЙКА #{displayedCellNumber}</p>

              {renderPhoneSummary()}

              <p className={styles.paymentStatus}>{paymentStatusMessage}</p>
            </div>
          )}
        </div>

        <div className={styles.centerDivider} />

        <div className={styles.rightColumn}>
          {state.step === "select-size" && (
            <div className={styles.rightSection}>
              <h2 className={styles.rightTitle}>ЕСТЬ КОД?</h2>

              <div className={styles.codeRow}>
                <Input
                  variant="compact"
                  className={styles.codeInput}
                  placeholder="Введите код"
                  value={state.codeInput}
                  disabled={isSubmitting}
                  onChange={(event) => dispatch({ type: "update-code", value: event.target.value })}
                />

                <Button
                  variant="compact"
                  className={styles.arrowButton}
                  disabled={isSubmitting}
                  aria-label="Проверить код доступа"
                  onClick={handleCodeSubmit}
                >
                  <svg viewBox="0 0 24 24" aria-hidden="true" className={styles.arrowIcon}>
                    <path d="M5 12H19" />
                    <path d="M13 6L19 12L13 18" />
                  </svg>
                </Button>
              </div>

              {state.formMessage && <p className={styles.statusMessage}>{state.formMessage}</p>}
            </div>
          )}

          {state.step === "phone-entry" && (
            <div className={styles.rightSection}>
              <h2 className={styles.rightTitle}>
                ЯЧЕЙКА ОТКРОЕТСЯ
                <br />
                ПОСЛЕ ВВОДА
                <br />
                НОМЕРА ТЕЛЕФОНА
              </h2>

              <p className={styles.helperTextWide}>
                Запомните или запишите код доступа, он понадобится для открытия ячейки камеры
                хранения.
              </p>
            </div>
          )}

          {state.step === "access-code" && (
            <div className={styles.rightSection}>
              <p className={styles.label}>КОД ДОСТУПА: {state.issuedCode}</p>
              <p className={styles.helperTextWide}>
                Запомните или запишите этот код, он понадобится для открытия ячейки камеры хранения.
              </p>
            </div>
          )}

          {state.step === "active-rent" && (
            <div className={styles.rightSection}>
              <p className={styles.label}>КОД ДОСТУПА:</p>
              <p className={styles.codeValue}>{state.issuedCode}</p>

              <p className={styles.helperTextWide}>
                Запомните или запишите этот код, он понадобится для открытия ячейки камеры хранения.
              </p>
            </div>
          )}

          {state.step === "payment" && (
            <div className={styles.rightSection}>
              <p className={styles.amountTitle}>К ОПЛАТЕ: {paymentAmountText}</p>

              <div className={styles.qrPlaceholder} aria-label="Моковый QR-код">
                QR
              </div>

              {state.paymentQrPayload && (
                <p className={styles.helperText}>ID платежа: {state.paymentId}</p>
              )}

              <p className={styles.helperTextWide}>После оплаты ячейка откроется автоматически.</p>
            </div>
          )}
        </div>
      </section>
    </main>
  );
}

export default Locker;
