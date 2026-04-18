import { test, expect } from "@playwright/test";
import path from "path";
import fs from "fs";

const SCENARIO = "login-to-signup-navigation";
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

test.describe("Scenario: Login to Signup Navigation", () => {
  test("should navigate from login to signup via Sign up link", async ({
    page,
  }) => {
    // Step 1: Navigate to /login
    await page.goto("/login");
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-01-login-page.png"),
      fullPage: true,
    });

    // Step 2: Click "Sign up" link
    await page.getByRole("link", { name: "Sign up" }).click();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-02-click-signup.png"),
      fullPage: true,
    });

    // Step 3: Assert URL is /signup
    await expect(page).toHaveURL(/\/signup/);
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-03-url-signup.png"),
      fullPage: true,
    });

    // Step 4: Assert signup heading is visible
    await expect(
      page.getByRole("heading", { name: "Create account" })
    ).toBeVisible();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-04-signup-heading.png"),
      fullPage: true,
    });
  });
});
