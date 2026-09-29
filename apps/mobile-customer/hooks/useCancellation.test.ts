import { describe, it, expect, jest } from "@jest/globals";

jest.mock("../lib/api", () => ({ api: {} }));

import { orderCancellable } from "./useCancellation";

describe("order cancellation eligibility", () => {
  it.each(["pending", "failed", "refunded", ""])(
    "does not offer a refund for %s payment",
    (paymentStatus) => {
      expect(orderCancellable("pending", paymentStatus)).toBe(false);
    },
  );

  it("offers cancellation for a paid order before delivery", () => {
    expect(orderCancellable("accepted", "completed")).toBe(true);
  });

  it("does not offer cancellation after delivery", () => {
    expect(orderCancellable("delivered", "completed")).toBe(false);
  });
});
