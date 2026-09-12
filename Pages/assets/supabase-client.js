import { createClient } from "@supabase/supabase-js";

const runtimeConfig =
  window.__PTSIP_SUPABASE__ &&
  typeof window.__PTSIP_SUPABASE__ === "object"
    ? window.__PTSIP_SUPABASE__
    : {};

function isConfigured(value) {
  return typeof value === "string" && value.trim().length > 0;
}

const configured =
  isConfigured(runtimeConfig.url) &&
  isConfigured(runtimeConfig.publishableKey);

export const supabaseClientState = Object.freeze({
  configured,
  registryQueryEnabled: false,
  authEnabled: false
});

export const supabase = configured
  ? createClient(
      runtimeConfig.url,
      runtimeConfig.publishableKey,
      {
        db: {
          schema: "public"
        },
        auth: {
          autoRefreshToken: false,
          persistSession: false,
          detectSessionInUrl: false
        }
      }
    )
  : null;
