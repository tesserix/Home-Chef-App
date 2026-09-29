import { describe, it, expect } from "@jest/globals";

import { refundDestinationLine } from "./refund-destination";

describe("refundDestinationLine", () => {
  // The live case that prompted this: order …14404483 cancelled pre-acceptance.
  // ₹377.07 refunded — ₹132.22 to the card, ₹244.42 wallet + ₹0.43 loyalty.
  // The screen said "₹377 refunded to your card".
  it("splits a mixed refund between the original payment method and the wallet", () => {
    expect(refundDestinationLine(37707, 244.42, 0.43, "original")).toBe(
      "₹377.07 refunded — ₹132.22 to your original payment method, ₹244.85 to your wallet.",
    );
  });

  it("names the original payment method when no credit funded the order", () => {
    expect(refundDestinationLine(37707, 0, 0, "original")).toBe(
      "₹377.07 refunded to your original payment method.",
    );
  });

  it("names only the wallet when credit funded the whole refund", () => {
    expect(refundDestinationLine(4107, 41.07, 0, "original")).toBe(
      "₹41.07 refunded to your wallet.",
    );
  });

  it("honours a wallet-destination order for the gateway slice", () => {
    expect(refundDestinationLine(10000, 0, 0, "wallet")).toBe(
      "₹100 refunded to your wallet.",
    );
  });

  it("does not round paise away", () => {
    // The defect this replaces used (paise / 100).toFixed(0): ₹377.07 -> "₹377".
    expect(refundDestinationLine(37707, 0, 0, "original")).toContain("₹377.07");
    // A whole amount still renders without decimals, per lib/format.
    expect(refundDestinationLine(40000, 0, 0, "original")).toContain("₹400");
  });

  it("treats missing rail figures as no credit rather than NaN", () => {
    expect(refundDestinationLine(34347, undefined, undefined, "original")).toBe(
      "₹343.47 refunded to your original payment method.",
    );
  });

  it("never reports a negative or over-total card slice when the rails exceed the refund", () => {
    const line = refundDestinationLine(4107, 999, 0, "original");
    expect(line).toBe("₹41.07 refunded to your wallet.");
    expect(line).not.toContain("-");
  });
});
