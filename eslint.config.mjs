import nextCoreWebVitals from "eslint-config-next/core-web-vitals";
import tseslint from "typescript-eslint";

const tsFiles = ["frontend/**/*.{ts,tsx}", "packages/**/*.ts"];
const nextFiles = ["frontend/**/*.{js,jsx,mjs,ts,tsx,mts,cts}"];
const scopedNextConfig = nextCoreWebVitals.map((config) => {
  if ("ignores" in config) {
    return config;
  }

  return {
    ...config,
    files: nextFiles,
    settings: {
      ...config.settings,
      next: {
        ...config.settings?.next,
        rootDir: "frontend/",
      },
    },
  };
});

export default [
  {
    ignores: [
      "**/node_modules/**",
      "**/.next/**",
      "**/dist/**",
      "**/coverage/**",
      "**/*.tsbuildinfo",
    ],
  },
  ...tseslint.configs.recommended.map((config) => ({
    ...config,
    files: tsFiles,
  })),
  ...scopedNextConfig,
];
