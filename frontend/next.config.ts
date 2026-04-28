import type { NextConfig } from "next";
import path from "node:path";

const nextConfig: NextConfig = {
  transpilePackages: [
    "@airpath/shared-types",
    "@airpath/geo",
    "@airpath/map-rendering",
    "@deck.gl/layers",
    "@deck.gl/react",
  ],
  turbopack: {
    root: path.resolve(process.cwd(), ".."),
  },
};

export default nextConfig;
