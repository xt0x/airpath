import { describe, expect, it } from "vitest";

import frontendPackage from "../../../package.json";

describe("frontend package boundaries", () => {
  it("declares workspace packages that feature code imports directly", () => {
    expect(frontendPackage.dependencies).toMatchObject({
      "@airpath/map-rendering": "workspace:*",
      "@airpath/shared-types": "workspace:*",
    });
  });
});
