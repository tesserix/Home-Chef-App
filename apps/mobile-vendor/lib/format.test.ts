import { describe, it, expect } from "@jest/globals";
import { currencySymbol, formatMoney, moneyValue } from "./format";

describe("vendor transaction currency", () => {
  it.each([
    ["INR", "₹1,234.56"],
    ["AUD", "$1,234.56"],
    ["NZD", "$1,234.56"],
  ])("renders %s in its home-market style without conversion", (currency, expected) => {
    expect(formatMoney(1234.56, currency)).toBe(expected);
  });
  it("keeps the legacy India default", () => {
    expect(formatMoney(20)).toBe("₹20");
    expect(moneyValue(20.15)).toBe("20.15");
  });
  it("labels unknown currencies by code rather than guessing a symbol", () => {
    expect(formatMoney(5, "usd")).toBe("USD 5");
  });
  it.each([
    ["INR", "₹"],
    ["aud", "$"],
    ["NZD", "$"],
    [undefined, "₹"],
  ])("gives %s an input-prefix symbol", (currency, expected) => {
    expect(currencySymbol(currency)).toBe(expected);
  });
});
