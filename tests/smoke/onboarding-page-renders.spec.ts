import { test, expect } from "@playwright/test";
import path from "path";
import fs from "fs";

const SCENARIO = "onboarding-page-renders";
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

test.describe("Scenario: Onboarding Page — Auth Guard", () => {
  test("should redirect /onboarding to /login when unauthenticated", async ({
    page,
    context,
  }) => {
    // Step 1: Clear cookies — no session
    await context.clearCookies();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-01-cookies-cleared.png"),
      fullPage: true,
    });

    // Step 2: Navigate to /onboarding
    // /onboarding is NOT in PUBLIC_ROUTES, so middleware redirects unauthenticated users
    await page.goto("/onboarding");
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-02-navigate-onboarding.png"),
      fullPage: true,
    });

    // Step 3: Assert redirected to /login
    await expect(page).toHaveURL(/\/login/);
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-03-redirected-to-login.png"),
      fullPage: true,
    });

    // Step 4: Assert "Sign in" heading is shown
    await expect(
      page.getByRole("heading", { name: "Sign in" })
    ).toBeVisible();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-04-login-heading.png"),
      fullPage: true,
    });
  });
});
