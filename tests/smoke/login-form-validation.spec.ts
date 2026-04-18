import { test, expect } from "@playwright/test";
import path from "path";
import fs from "fs";

const SCENARIO = "login-form-validation";
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

test.describe("Scenario: Login Form Client-Side Validation", () => {
  test("should show inline errors on empty submit", async ({ page }) => {
    // Step 1: Navigate to /login
    await page.goto("/login");
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-01-navigate.png"),
      fullPage: true,
    });

    // Step 2: Click "Sign in" without filling any fields
    // react-hook-form with zodResolver fires validation on submit
    await page.getByRole("button", { name: "Sign in" }).click();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-02-empty-submit.png"),
      fullPage: true,
    });

    // Step 3: Assert email validation error appears
    // loginSchema: z.string().email("Enter a valid email address")
    await expect(
      page.getByText("Enter a valid email address")
    ).toBeVisible();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-03-email-error.png"),
      fullPage: true,
    });

    // Step 4: Assert password validation error appears
    // loginSchema: z.string().min(1, "Password is required")
    await expect(page.getByText("Password is required")).toBeVisible();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-04-password-error.png"),
      fullPage: true,
    });
  });
});
