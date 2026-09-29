import { describe, it, expect } from "@jest/globals";

import { fssaiDashboardNotice, type FssaiRequest } from "./fssai";

const request: FssaiRequest = {
  id: "test-request",
  status: "issued",
  termYears: 1,
  kitchenName: "Test kitchen",
  applicantName: "Test chef",
  feeAmount: 100,
  feeTax: 18,
  feeTotal: 118,
  currency: "INR",
  registrationNo: "TEST-REGISTRATION",
  documents: [],
  needsDocuments: false,
  canPay: false,
  createdAt: "2026-09-29",
};

describe("FSSAI dashboard notice", () => {
  it("hides completed registrations without requiring dismissal", () => {
    expect(fssaiDashboardNotice(request)).toBeNull();
  });

  it("keeps requests needing vendor action visible", () => {
    expect(
      fssaiDashboardNotice({ ...request, status: "more_info_required" })?.tone,
    ).toBe("urgent");
  });
});
