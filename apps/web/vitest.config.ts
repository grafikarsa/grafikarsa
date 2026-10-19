import { defineConfig } from "vitest/config";
import path from "node:path";

// Vitest: unit test murni untuk logika klien (validator payload block,
// pemetaan deep-link notifikasi, dsb). Test komponen/e2e menyusul terpisah.
export default defineConfig({
  test: {
    include: ["**/*.test.ts", "**/*.test.tsx"],
    exclude: ["node_modules", ".next"],
    environment: "node",
    passWithNoTests: false,
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "."),
    },
  },
});
