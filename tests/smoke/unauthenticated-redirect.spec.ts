import { test, expect } from "@playwright/test";
import path from "path";
import fs from "fs";

const SCENARIO = "unauthenticated-redirect";
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

test.describe("Scenario: Unauthenticated Redirect", () => {
  test("should redirect / to /login when no session cookie is present", async ({
    page,
    context,
  }) => {
    // Step 1: Clear all cookies — ensure no session
    await context.clearCookies();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-01-cookies-cleared.png"),
      fullPage: true,
    });

    // Step 2: Navigate to / (dashboard root)
    // middleware.ts checks for better-auth.session_token cookie;
    // absent cookie triggers redirect to /login?redirect=%2F
    await page.goto("/");
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-02-navigate-root.png"),
      fullPage: true,
    });

    // Step 3: Assert final URL contains /login
    await expect(page).toHaveURL(/\/login/);
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-03-url-contains-login.png"),
      fullPage: true,
    });

    // Step 4: Assert the login page heading is visible (page actually rendered)
    await expect(
      page.getByRole("heading", { name: "Sign in" })
    ).toBeVisible();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-04-login-heading.png"),
      fullPage: true,
    });
  });
});
