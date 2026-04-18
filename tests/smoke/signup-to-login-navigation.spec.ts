import { test, expect } from "@playwright/test";
import path from "path";
import fs from "fs";

const SCENARIO = "signup-to-login-navigation";
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

test.describe("Scenario: Signup to Login Navigation", () => {
  test("should navigate from signup back to login via Sign in link", async ({
    page,
  }) => {
    // Step 1: Navigate to /signup
    await page.goto("/signup");
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-01-signup-page.png"),
      fullPage: true,
    });

    // Step 2: Click "Sign in" link
    await page.getByRole("link", { name: "Sign in" }).click();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-02-click-signin.png"),
      fullPage: true,
    });

    // Step 3: Assert URL is /login
    await expect(page).toHaveURL(/\/login/);
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-03-url-login.png"),
      fullPage: true,
    });

    // Step 4: Assert login heading is visible
    await expect(
      page.getByRole("heading", { name: "Sign in" })
    ).toBeVisible();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-04-login-heading.png"),
      fullPage: true,
    });
  });
});
