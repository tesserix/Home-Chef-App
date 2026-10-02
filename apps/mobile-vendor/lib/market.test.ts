import { describe, it, expect } from "@jest/globals";

import { getMarket, isValidPostcode, MARKET_CODES } from "./market";

describe("kitchen markets", () => {
  it("serves India, Australia and New Zealand", () => {
    expect(MARKET_CODES).toEqual(["IN", "AU", "NZ"]);
  });

  it("falls back to India for a draft saved before the country existed", () => {
    expect(getMarket(undefined).code).toBe("IN");
    expect(getMarket("xx").code).toBe("IN");
  });

  it("checks postcodes per country", () => {
    expect(isValidPostcode("560001", "IN")).toBe(true);
    expect(isValidPostcode("3000", "IN")).toBe(false);
    expect(isValidPostcode("3000", "AU")).toBe(true);
    expect(isValidPostcode("1010", "NZ")).toBe(true);
    expect(isValidPostcode("560001", "NZ")).toBe(false);
  });

  it("asks AU and NZ kitchens for the council food registration, not FSSAI", () => {
    expect(getMarket("IN").licenceDocType).toBe("fssai_license");
    expect(getMarket("AU").licenceDocType).toBe("food_safety_cert");
    expect(getMarket("NZ").licenceDocType).toBe("food_safety_cert");
  });

  it("pays AU and NZ kitchens through Stripe", () => {
    expect(getMarket("IN").payoutRail).toBe("bank");
    expect(getMarket("AU").payoutRail).toBe("stripe");
    expect(getMarket("AU").currency).toBe("AUD");
    expect(getMarket("NZ").currency).toBe("NZD");
  });

  it("lists every Australian state and territory and New Zealand region", () => {
    expect(getMarket("AU").regions).toHaveLength(8);
    expect(getMarket("NZ").regions).toHaveLength(16);
    expect(getMarket("IN").regions).toBeNull();
  });
});
