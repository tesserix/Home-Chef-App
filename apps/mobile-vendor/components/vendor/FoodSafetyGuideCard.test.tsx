import React from "react";
import { jest, it, expect, afterEach } from "@jest/globals";
import { act, create } from "react-test-renderer";
import { Linking } from "react-native";
import { FoodSafetyGuideCard } from "./FoodSafetyGuideCard";

const mockToast = jest.fn();
jest.mock("react-native", () => ({
  Linking: { openURL: jest.fn() },
  Pressable: "Pressable",
  Text: "Text",
  View: "View",
  StyleSheet: { create: (styles: unknown) => styles },
}));
jest.mock("@homechef/mobile-shared/ui", () => ({
  useToast: () => ({ show: mockToast }),
}));
(
  globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

let screen: ReturnType<typeof create>;
afterEach(async () => {
  if (screen) await act(async () => screen.unmount());
  jest.clearAllMocks();
});

it.each(["AU", "NZ"] as const)(
  "opens the %s registration guide with upload instructions",
  async (country) => {
    await act(async () => {
      screen = create(<FoodSafetyGuideCard country={country} />);
    });
    const toggle = screen.root.findByProps({
      accessibilityLabel: "Food safety registration guide",
    });
    expect(toggle.props.accessibilityState.expanded).toBe(false);
    await act(async () => {
      toggle.props.onPress();
    });
    const text = JSON.stringify(screen.toJSON());
    expect(text).toContain(
      country === "AU" ? "local council or food regulator" : "My Food Rules",
    );
    expect(text).toContain("Upload your registration");
    expect(text).toContain("training certificate");
    expect(text).not.toContain("FSSAI");
  },
);

it("keeps the Indian FSSAI flow separate", async () => {
  await act(async () => {
    screen = create(<FoodSafetyGuideCard country="IN" />);
  });
  expect(screen.toJSON()).toBeNull();
});

it("opens the official NZ requirements and reports a browser failure", async () => {
  jest.mocked(Linking.openURL).mockRejectedValueOnce(new Error("Unavailable"));
  await act(async () => {
    screen = create(<FoodSafetyGuideCard country="NZ" />);
  });
  await act(async () => {
    screen.root
      .findByProps({ accessibilityLabel: "Food safety registration guide" })
      .props.onPress();
  });
  await act(async () => {
    await screen.root
      .findByProps({ accessibilityLabel: "Open MPI My Food Rules" })
      .props.onPress();
  });
  expect(Linking.openURL).toHaveBeenCalledWith(
    "https://www.mpi.govt.nz/food-business/food-safety-rules",
  );
  expect(mockToast).toHaveBeenCalledWith(
    expect.objectContaining({ tone: "error" }),
  );
});
