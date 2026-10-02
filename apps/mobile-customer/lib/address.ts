import { z } from "zod";

export const addressSchema = z
  .object({
    label: z.string().min(1),
    addressLine1: z.string().min(5, "Address must be at least 5 characters"),
    addressLine2: z.string().optional(),
    city: z.string().min(1, "City is required"),
    state: z.string().min(1, "State is required"),
    country: z.enum(["IN", "AU", "NZ"]),
    pincode: z.string(),
    isDefault: z.boolean().optional(),
  })
  .superRefine((address, ctx) => {
    const digits = address.country === "IN" ? 6 : 4;
    if (!(digits === 6 ? /^\d{6}$/ : /^\d{4}$/).test(address.pincode)) {
      ctx.addIssue({
        code: "custom",
        path: ["pincode"],
        message: `Enter a valid ${digits}-digit postcode`,
      });
    }
  });
