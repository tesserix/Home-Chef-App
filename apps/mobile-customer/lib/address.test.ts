import { addressSchema } from "./address";
import { expect, it } from "@jest/globals";

const address = {
  label: "Home",
  addressLine1: "90 Swanston Street",
  city: "Melbourne",
  state: "Victoria",
  country: "AU",
  pincode: "3000",
};

it.each([
  ["AU", "3000"],
  ["NZ", "0620"],
  ["IN", "560001"],
])("accepts a %s postcode and preserves its country", (country, pincode) => {
  expect(addressSchema.parse({ ...address, country, pincode })).toMatchObject({
    country,
    pincode,
  });
});

it.each([
  ["AU", "560001"],
  ["NZ", "123"],
  ["IN", "3000"],
  ["AU", "abcd"],
  ["US", "123456"],
  ["", "123456"],
])("rejects country %s with postcode %s", (country, pincode) => {
  expect(
    addressSchema.safeParse({ ...address, country, pincode }).success,
  ).toBe(false);
});
