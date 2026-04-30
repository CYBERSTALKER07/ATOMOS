import { z } from "zod";

export const GeolocationSchema = z.object({
    latitude: z.number().min(-90).max(90),
    longitude: z.number().min(-180).max(180),
});

export const RetailerSignupSchema = z.object({
    name: z.string().min(2, "Retailer name must be at least 2 characters"),
    taxIdentificationNumber: z
        .string()
        .min(9, "TIN/STIR must be at least 9 characters"),
    shopLocation: GeolocationSchema,
});

export type RetailerSignup = z.infer<typeof RetailerSignupSchema>;
export type Geolocation = z.infer<typeof GeolocationSchema>;
