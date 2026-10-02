import { jest, it, expect } from "@jest/globals";
import { useOrderDetail } from "./useOrderDetail";
import { api } from "../lib/api";
import { useQuery } from "@tanstack/react-query";

jest.mock("../lib/api", () => ({ api: { get: jest.fn() } }));
jest.mock("@tanstack/react-query", () => ({ useQuery: jest.fn() }));

it.each(["INR", "AUD", "NZD"])(
  "preserves %s from the order API",
  async (currency) => {
    jest.mocked(api.get).mockResolvedValue({
      data: {
        id: "order",
        orderNumber: "HC-test",
        status: "pending",
        currency,
        createdAt: "2026-10-01T00:00:00Z",
        subtotal: 20.15,
        total: 23.15,
      },
    });
    useOrderDetail("order");
    const options = jest.mocked(useQuery).mock.calls.at(-1)![0];
    const result = await (
      options.queryFn as () => Promise<{
        currency: string;
        pricing: { total: number };
      }>
    )();
    expect(result.currency).toBe(currency);
    expect(result.pricing.total).toBe(23.15);
  },
);
