import { useEffect, useState } from "react";
import Logo from "../../components/Logo/Logo";
import styles from "./lockerList.module.css";

type LockerSizes = {
  s: number;
  m: number;
  l: number;
  xl: number;
};

type LockerItem = {
  id: number;
  street: string;
  freeCells: LockerSizes;
};

type LockerListResponse = {
  ok: boolean;
  data: LockerItem[];
  meta: {
    total: number;
  };
};

const splitStreetText = (street: string) => {
  const [firstWord = "", ...restWords] = street.trim().split(/\s+/);

  return {
    firstWord,
    restText: restWords.join(" "),
  };
};

function LockerList() {
  const [lockers, setLockers] = useState<LockerItem[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const controller = new AbortController();

    const fetchLockers = async (showLoader: boolean) => {
      try {
        if (showLoader) {
          setIsLoading(true);
        }

        const response = await fetch(new URL("./mock.json", import.meta.url), {
          signal: controller.signal,
        });

        if (!response.ok) {
          throw new Error(`Request failed: ${response.status}`);
        }

        const payload = (await response.json()) as LockerListResponse;

        if (!payload.ok) {
          throw new Error("Mock API returned an error status");
        }

        setLockers(payload.data);
        setError(null);
      } catch (requestError) {
        if ((requestError as Error).name === "AbortError") {
          return;
        }

        if (showLoader) {
          setError("Не удалось загрузить список камер хранения");
        }
      } finally {
        if (showLoader) {
          setIsLoading(false);
        }
      }
    };

    void fetchLockers(true);

    const intervalId = window.setInterval(() => {
      void fetchLockers(false);
    }, 20000);

    return () => {
      controller.abort();
      window.clearInterval(intervalId);
    };
  }, []);

  return (
    <main className={styles.page}>
      <header className={styles.header}>
        <Logo />
      </header>

      {isLoading && <p className={styles.message}>Загрузка камер хранения...</p>}

      {error && <p className={styles.message}>{error}</p>}

      {!isLoading && !error && (
        <section className={styles.list} aria-label="Список камер хранения">
          {lockers.map((locker) => {
            const { firstWord, restText } = splitStreetText(locker.street);

            return (
              <article key={locker.id} className={styles.listItem}>
                <h2 className={styles.street}>
                  <span className={styles.streetFirstWord}>{firstWord}</span>
                  {restText && (
                    <span className={styles.streetRestWords}>{restText}</span>
                  )}
                </h2>

                <div className={styles.sizes}>
                  <span className={styles.sizeItem}>
                    <span className={styles.sizeLabel}>S:</span>
                    <span className={styles.sizeValue}>{locker.freeCells.s}</span>
                  </span>
                  <span className={styles.sizeItem}>
                    <span className={styles.sizeLabel}>M:</span>
                    <span className={styles.sizeValue}>{locker.freeCells.m}</span>
                  </span>
                  <span className={styles.sizeItem}>
                    <span className={styles.sizeLabel}>L:</span>
                    <span className={styles.sizeValue}>{locker.freeCells.l}</span>
                  </span>
                  <span className={styles.sizeItem}>
                    <span className={styles.sizeLabel}>XL:</span>
                    <span className={styles.sizeValue}>{locker.freeCells.xl}</span>
                  </span>
                </div>
              </article>
            );
          })}
        </section>
      )}

      <footer className={styles.footer}>LOCK&apos;IT © all rights reserved</footer>
    </main>
  );
}

export default LockerList;