import { test, expect } from "@playwright/test";
import path from "path";
import fs from "fs";

const SCENARIO = "signup-form-validation";
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

test.describe("Scenario: Signup Form Client-Side Validation", () => {
  test("should show inline errors on empty submit", async ({ page }) => {
    // Step 1: Navigate to /signup
    await page.goto("/signup");
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-01-navigate.png"),
      fullPage: true,
    });

    // Step 2: Click "Create account" without filling any fields
    await page.getByRole("button", { name: "Create account" }).click();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-02-empty-submit.png"),
      fullPage: true,
    });

    // Step 3: Assert name validation error
    // signupSchema: z.string().min(1, "Name is required")
    await expect(page.getByText("Name is required")).toBeVisible();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-03-name-error.png"),
      fullPage: true,
    });

    // Step 4: Assert email validation error
    // signupSchema: z.string().email("Enter a valid email address")
    await expect(
      page.getByText("Enter a valid email address")
    ).toBeVisible();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-04-email-error.png"),
      fullPage: true,
    });
  });
});
