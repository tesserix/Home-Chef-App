import { describe, it, expect } from "@jest/globals";
import { formatMoney, moneyValue } from "./format";

describe("vendor transaction currency", () => {
  it.each([
    ["INR", "₹1,234.56"],
    ["AUD", "AUD 1,234.56"],
    ["NZD", "NZD 1,234.56"],
  ])("preserves %s and cents without conversion", (currency, expected) => {
    expect(formatMoney(1234.56, currency)).toBe(expected);
  });
  it("keeps the legacy India default", () => {
    expect(formatMoney(20)).toBe("₹20");
    expect(moneyValue(20.15)).toBe("20.15");
  });
});
