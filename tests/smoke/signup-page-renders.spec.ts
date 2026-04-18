import { test, expect } from "@playwright/test";
import path from "path";
import fs from "fs";

const SCENARIO = "signup-page-renders";
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

test.describe("Scenario: Signup Page Renders", () => {
  test("should render all signup form fields", async ({ page }) => {
    // Step 1: Navigate to /signup
    await page.goto("/signup");
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-01-navigate.png"),
      fullPage: true,
    });

    // Step 2: Assert heading "Create account"
    // SignupForm uses CardTitle with "Create account"
    await expect(
      page.getByRole("heading", { name: "Create account" })
    ).toBeVisible();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-02-heading.png"),
      fullPage: true,
    });

    // Step 3: Assert "Full name" input
    await expect(page.getByLabel("Full name")).toBeVisible();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-03-name-input.png"),
      fullPage: true,
    });

    // Step 4: Assert "Email" input
    await expect(page.getByLabel("Email")).toBeVisible();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-04-email-input.png"),
      fullPage: true,
    });

    // Step 5: Assert "Password" input
    await expect(page.getByLabel("Password")).toBeVisible();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-05-password-input.png"),
      fullPage: true,
    });

    // Step 6: Assert "Confirm password" input
    await expect(page.getByLabel("Confirm password")).toBeVisible();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-06-confirm-password-input.png"),
      fullPage: true,
    });

    // Step 7: Assert submit button "Create account"
    await expect(
      page.getByRole("button", { name: "Create account" })
    ).toBeVisible();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-07-submit-button.png"),
      fullPage: true,
    });

    // Step 8: Assert "Sign in" link back to /login
    const signInLink = page.getByRole("link", { name: "Sign in" });
    await expect(signInLink).toBeVisible();
    await expect(signInLink).toHaveAttribute("href", "/login");
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-08-signin-link.png"),
      fullPage: true,
    });
  });
});
