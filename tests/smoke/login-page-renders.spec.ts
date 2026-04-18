import { test, expect } from "@playwright/test";
import path from "path";
import fs from "fs";

const SCENARIO = "login-page-renders";
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

test.describe("Scenario: Login Page Renders", () => {
  test("should render all login form elements", async ({ page }) => {
    // Step 1: Navigate to /login
    await page.goto("/login");
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-01-navigate.png"),
      fullPage: true,
    });

    // Step 2: Assert heading "Sign in" is visible
    // LoginForm uses CardTitle with "Sign in"
    await expect(
      page.getByRole("heading", { name: "Sign in" })
    ).toBeVisible();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-02-heading.png"),
      fullPage: true,
    });

    // Step 3: Assert email input is visible
    await expect(page.getByLabel("Email")).toBeVisible();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-03-email-input.png"),
      fullPage: true,
    });

    // Step 4: Assert password input is visible
    await expect(page.getByLabel("Password")).toBeVisible();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-04-password-input.png"),
      fullPage: true,
    });

    // Step 5: Assert submit button is visible
    await expect(
      page.getByRole("button", { name: "Sign in" })
    ).toBeVisible();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-05-submit-button.png"),
      fullPage: true,
    });

    // Step 6: Assert "Sign up" link is visible and points to /signup
    const signUpLink = page.getByRole("link", { name: "Sign up" });
    await expect(signUpLink).toBeVisible();
    await expect(signUpLink).toHaveAttribute("href", "/signup");
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-06-signup-link.png"),
      fullPage: true,
    });
  });
});
