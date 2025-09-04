import { z } from 'zod';
import validate from 'express-zod-safe';

// Schema validators for common request bodies
export const EmailVerificationSchema = z.object({
  email: z.email(),
  code: z.string().length(6),
});

export const SignInSchema = z.object({
  email: z.email(),
});

// Type-safe validation middleware using express-zod-safe
export const validateSignIn = validate({
  body: SignInSchema,
});

export const validateEmailVerification = validate({
  body: EmailVerificationSchema,
});