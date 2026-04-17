import { useEffect, useMemo, useReducer, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import Button from "../../components/Button/Button";
import Input from "../../components/Input/Input";
import Logo from "../../components/Logo/Logo";
import styles from "./Locker.module.css";
import {
  LOCKER_MOCK_DATA,
  type LockerMockScenario,
  type LockerSize,
} from "./mock";

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
  availableCellsBySize: Record<LockerSize, number[]>;
  currentCellNumber: number | null;
  dimensions: Dimensions;
  codeInput: string;
  phoneInput: string;
  confirmedPhone: string;
  issuedCode: string;
  formMessage: string | null;
};

type Action =
  | { type: "select-size"; size: LockerSize; cellNumber: number | null }
  | { type: "update-dimension"; field: keyof Dimensions; value: string }
  | { type: "update-code"; value: string }
  | { type: "update-phone"; value: string }
  | { type: "set-step"; step: FlowStep }
  | { type: "set-message"; message: string | null }
  | { type: "booking-created"; phone: string; code: string; cellNumber: number }
  | { type: "payment-started"; phone: string; code: string; cellNumber: number }
  | { type: "reset"; state: FlowState };

const SIZE_OPTIONS: LockerSize[] = ["s", "m", "l", "xl"];

const SIZE_LABELS: Record<LockerSize, string> = {
  s: "S",
  m: "M",
  l: "L",
  xl: "XL",
};

const cloneCellsBySize = (cells: Record<LockerSize, number[]>) => ({
  s: [...cells.s],
  m: [...cells.m],
  l: [...cells.l],
  xl: [...cells.xl],
});

const createLockerScenario = (
  lockerIdRaw: string | undefined,
): LockerMockScenario => {
  const parsedLockerId = Number(lockerIdRaw);

  if (Number.isFinite(parsedLockerId)) {
    const byId = LOCKER_MOCK_DATA.find((item) => item.id === parsedLockerId);

    if (byId) {
      return byId;
    }
  }

  return LOCKER_MOCK_DATA[0];
};

const createInitialFlowState = (scenario: LockerMockScenario): FlowState => ({
  step: "select-size",
  selectedSize: null,
  availableCellsBySize: cloneCellsBySize(scenario.availableCellsBySize),
  currentCellNumber: null,
  dimensions: {
    length: "",
    width: "",
    height: "",
  },
  codeInput: "",
  phoneInput: "",
  confirmedPhone: scenario.activeRentalPhone,
  issuedCode: scenario.existingAccessCode,
  formMessage: null,
});

const generateAccessCode = () =>
  Math.random().toString(36).slice(2, 8).toUpperCase();

const isPhoneValid = (phone: string) => {
  const digits = phone.replace(/\D/g, "");
  if (digits.length !== 11) {
    return false;
  }

  return digits.startsWith("7") || digits.startsWith("8");
};

const normalizePhone = (phone: string) => {
  const digits = phone.replace(/\D/g, "");

  if (digits.length !== 11) {
    return phone;
  }

  if (digits.startsWith("8")) {
    return `+7${digits.slice(1)}`;
  }

  if (digits.startsWith("7")) {
    return `+${digits}`;
  }

  return phone;
};

const detectLockerSizeByDimensions = (dimensions: Dimensions): LockerSize | null => {
  const numericValues = [dimensions.length, dimensions.width, dimensions.height]
    .map((value) => Number(value.trim().replace(",", ".")));

  if (numericValues.some((value) => !Number.isFinite(value) || value <= 0)) {
    return null;
  }

  const maxDimension = Math.max(...numericValues);

  if (maxDimension <= 40) {
    return "s";
  }

  if (maxDimension <= 60) {
    return "m";
  }

  if (maxDimension <= 90) {
    return "l";
  }

  return "xl";
};

const lockerReducer = (state: FlowState, action: Action): FlowState => {
  switch (action.type) {
    case "select-size": {
      return {
        ...state,
        selectedSize: action.size,
        currentCellNumber: action.cellNumber,
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
      const nextAvailableCellsBySize = cloneCellsBySize(state.availableCellsBySize);

      if (state.selectedSize && state.currentCellNumber !== null) {
        nextAvailableCellsBySize[state.selectedSize] =
          nextAvailableCellsBySize[state.selectedSize].filter(
            (cellNumber) => cellNumber !== state.currentCellNumber,
          );
      }

      return {
        ...state,
        step: "access-code",
        availableCellsBySize: nextAvailableCellsBySize,
        currentCellNumber: action.cellNumber,
        confirmedPhone: action.phone,
        phoneInput: action.phone,
        issuedCode: action.code,
        formMessage: null,
      };
    }
    case "payment-started": {
      return {
        ...state,
        step: "payment",
        currentCellNumber: action.cellNumber,
        confirmedPhone: action.phone,
        issuedCode: action.code,
        formMessage: null,
      };
    }
    case "reset": {
      return action.state;
    }
    default: {
      return state;
    }
  }
};

function Locker() {
  const navigate = useNavigate();
  const { lockerId } = useParams<{ lockerId: string }>();

  const scenario = useMemo(() => createLockerScenario(lockerId), [lockerId]);
  const initialFlowState = useMemo(
    () => createInitialFlowState(scenario),
    [scenario],
  );

  const [state, dispatch] = useReducer(lockerReducer, initialFlowState);
  const [dimensionsMessage, setDimensionsMessage] = useState<string | null>(null);

  useEffect(() => {
    dispatch({ type: "reset", state: initialFlowState });
  }, [initialFlowState]);

  useEffect(() => {
    if (state.step !== "payment") {
      return;
    }

    const timeoutId = window.setTimeout(() => {
      dispatch({ type: "set-step", step: "active-rent" });
    }, 5000);

    return () => {
      window.clearTimeout(timeoutId);
    };
  }, [state.step]);

  const moveToNextBySize = (size: LockerSize) => {
    const availableCells = state.availableCellsBySize[size];
    const nextCellNumber = availableCells[0] ?? null;

    dispatch({ type: "select-size", size, cellNumber: nextCellNumber });

    if (nextCellNumber === null) {
      dispatch({ type: "set-step", step: "no-cells" });
      return;
    }

    dispatch({ type: "set-step", step: "phone-entry" });
  };

  const handleSizeSelect = (size: LockerSize) => {
    setDimensionsMessage(null);
    moveToNextBySize(size);
  };

  const handleContinueByDimensions = () => {
    const detectedSize = detectLockerSizeByDimensions(state.dimensions);

    if (!detectedSize) {
      setDimensionsMessage("Введите корректные габариты багажа");
      return;
    }

    setDimensionsMessage(null);
    moveToNextBySize(detectedSize);
  };

  const handleCodeSubmit = () => {
    const enteredCode = state.codeInput.trim().toUpperCase();

    if (!enteredCode) {
      dispatch({ type: "set-message", message: "Введите код доступа" });
      return;
    }

    if (enteredCode !== scenario.existingAccessCode) {
      dispatch({ type: "set-message", message: "Код не найден" });
      return;
    }

    dispatch({
      type: "payment-started",
      phone: scenario.activeRentalPhone,
      code: enteredCode,
      cellNumber: scenario.activeCellNumber,
    });
  };

  const handlePhoneSubmit = () => {
    const phoneValue = state.phoneInput.trim();

    if (!isPhoneValid(phoneValue)) {
      dispatch({
        type: "set-message",
        message: "Введите корректный номер телефона",
      });
      return;
    }

    if (state.currentCellNumber === null) {
      dispatch({
        type: "set-message",
        message: "Не удалось определить номер ячейки, выберите размер заново",
      });
      return;
    }

    dispatch({
      type: "booking-created",
      phone: normalizePhone(phoneValue),
      code: generateAccessCode(),
      cellNumber: state.currentCellNumber,
    });
  };

  const handleExit = () => {
    dispatch({ type: "reset", state: initialFlowState });
    navigate(`/locker/${scenario.id}`, { replace: true });
  };

  const handleEndRent = () => {
    dispatch({ type: "reset", state: initialFlowState });
  };

  const displayedCellNumber = state.currentCellNumber ?? scenario.activeCellNumber;

  const renderPhoneSummary = () => (
    <div className={styles.phoneSummary}>
      <p className={styles.label}>НОМЕР ТЕЛЕФОНА: {state.confirmedPhone}</p>
      <p className={styles.helperText}>
        Если это не ваш номер телефона, обратитесь в поддержку.
      </p>
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
            onClick={() => dispatch({ type: "reset", state: initialFlowState })}
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
                onClick={handleContinueByDimensions}
              >
                Далее
              </Button>

              {dimensionsMessage && (
                <p className={styles.statusMessage}>{dimensionsMessage}</p>
              )}

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
                  onChange={(event) =>
                    dispatch({ type: "update-phone", value: event.target.value })
                  }
                />

                <Button
                  variant="compact"
                  className={styles.primaryAction}
                  onClick={handlePhoneSubmit}
                >
                  Открыть ячейку
                </Button>
                              {state.formMessage && (
                <p className={styles.statusMessage}>{state.formMessage}</p>
              )}
              </div>

              <Button
                variant="compact"
                className={styles.secondaryAction}
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
                onClick={handleExit}
              >
                Выйти
              </Button>
            </div>
          )}

          {state.step === "active-rent" && (
            <div className={styles.formSectionTall}>
              <p className={styles.lockerTitle}>ЯЧЕЙКА #{displayedCellNumber}</p>

              {renderPhoneSummary()}

              <Button
                variant="compact"
                className={styles.secondaryAction}
                onClick={handleEndRent}
              >
                Завершить аренду
              </Button>
            </div>
          )}

          {state.step === "payment" && (
            <div className={styles.formSectionTall}>
              <p className={styles.lockerTitle}>ЯЧЕЙКА #{displayedCellNumber}</p>

              {renderPhoneSummary()}

              <p className={styles.paymentStatus}>Проверяем оплату...</p>
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
                  onChange={(event) =>
                    dispatch({ type: "update-code", value: event.target.value })
                  }
                />

                <Button
                  variant="compact"
                  className={styles.arrowButton}
                  aria-label="Проверить код доступа"
                  onClick={handleCodeSubmit}
                >
                  <svg
                    viewBox="0 0 24 24"
                    aria-hidden="true"
                    className={styles.arrowIcon}
                  >
                    <path d="M5 12H19" />
                    <path d="M13 6L19 12L13 18" />
                  </svg>
                </Button>
              </div>

              {state.formMessage && (
                <p className={styles.statusMessage}>{state.formMessage}</p>
              )}
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
                Запомните или запишите код доступа, он понадобится для открытия
                ячейки камеры хранения.
              </p>
            </div>
          )}

          {state.step === "access-code" && (
            <div className={styles.rightSection}>
              <p className={styles.label}>КОД ДОСТУПА: {state.issuedCode}</p>
              <p className={styles.helperTextWide}>
                Запомните или запишите этот код, он понадобится для открытия
                ячейки камеры хранения.
              </p>
            </div>
          )}

          {state.step === "active-rent" && (
            <div className={styles.rightSection}>
              <p className={styles.label}>КОД ДОСТУПА:</p>
              <p className={styles.codeValue}>{state.issuedCode}</p>

              <p className={styles.helperTextWide}>
                Запомните или запишите этот код, он понадобится для открытия
                ячейки камеры хранения.
              </p>
            </div>
          )}

          {state.step === "payment" && (
            <div className={styles.rightSection}>
              <p className={styles.amountTitle}>К ОПЛАТЕ: {scenario.paymentAmount} Р</p>

              <div className={styles.qrPlaceholder} aria-label="Моковый QR-код">
                QR
              </div>

              <p className={styles.helperTextWide}>
                После оплаты подождите 5 секунд, ячейка откроется автоматически.
              </p>
            </div>
          )}
        </div>
      </section>
    </main>
  );
}

export default Locker;