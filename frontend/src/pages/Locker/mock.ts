export type LockerSize = "s" | "m" | "l" | "xl";

export type LockerMockScenario = {
  id: number;
  availableCellsBySize: Record<LockerSize, number[]>;
  existingAccessCode: string;
  activeRentalPhone: string;
  activeCellNumber: number;
  paymentAmount: number;
};

export const LOCKER_MOCK_DATA: LockerMockScenario[] = [
  {
    id: 123,
    availableCellsBySize: {
      s: [101, 102, 103, 104],
      m: [201, 202, 203, 204],
      l: [301, 302],
      xl: [],
    },
    existingAccessCode: "1A2B3C",
    activeRentalPhone: "+79999999999",
    activeCellNumber: 214,
    paymentAmount: 900,
  },
  {
    id: 101,
    availableCellsBySize: {
      s: [11, 12, 13],
      m: [21, 22],
      l: [31],
      xl: [],
    },
    existingAccessCode: "7D4F8Q",
    activeRentalPhone: "+79991112233",
    activeCellNumber: 22,
    paymentAmount: 600,
  },
  {
    id: 112,
    availableCellsBySize: {
      s: [111, 112, 113, 114, 115],
      m: [121, 122, 123],
      l: [131, 132, 133],
      xl: [141],
    },
    existingAccessCode: "9K2M1P",
    activeRentalPhone: "+79995556677",
    activeCellNumber: 123,
    paymentAmount: 1200,
  },
];
