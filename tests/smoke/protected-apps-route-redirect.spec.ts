import { test, expect } from "@playwright/test";
import path from "path";
import fs from "fs";

const SCENARIO = "protected-apps-route-redirect";
const RUN_ID = new Date().toISOString().replace(/[:.]/g, "-").slice(0, 19);
const RESULTS_DIR = path.join(
  __dirname,
  "results",
  SCENARIO,
  RUN_ID
);

test.beforeAll(() => {
  fs.mkdirSync(RESULTS_DIR, { recursive: true });
});

test.describe("Scenario: Protected /apps Route Redirect", () => {
  test("should redirect /apps to /login with redirect param when unauthenticated", async ({
    page,
    context,
  }) => {
    // Step 1: Clear cookies — no session
    await context.clearCookies();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-01-cookies-cleared.png"),
      fullPage: true,
    });

    // Step 2: Navigate to /apps
    await page.goto("/apps");
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-02-navigate-apps.png"),
      fullPage: true,
    });

    // Step 3: Assert redirected to /login
    await expect(page).toHaveURL(/\/login/);
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-03-redirected-to-login.png"),
      fullPage: true,
    });

    // Step 4: Assert redirect query param encodes /apps
    // middleware sets loginUrl.searchParams.set("redirect", pathname)
    await expect(page).toHaveURL(/redirect/);
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-04-redirect-param.png"),
      fullPage: true,
    });
  });
});
