import { defineConfig } from "orval";

export default defineConfig({
  client: {
    input: "../../packages/contracts/openapi.yaml",
    output: {
      mode: "tags-split",
      target: "lib/api/gen/endpoints",
      schemas: "lib/api/gen/models",
      client: "react-query",
      httpClient: "fetch",
      clean: true,
      override: {
        mutator: {
          path: "lib/api/fetcher.ts",
          name: "apiFetch",
        },
        query: {
          useQuery: true,
          useSuspenseQuery: false,
          signal: true,
        },
        fetch: {
          includeHttpResponseReturnType: false,
        },
      },
    },
  },
  zod: {
    input: "../../packages/contracts/openapi.yaml",
    output: {
      mode: "tags-split",
      target: "lib/api/gen/zod",
      client: "zod",
      fileExtension: ".zod.ts",
      clean: true,
    },
  },
});
