import { test, expect } from "@playwright/test";
import path from "path";
import fs from "fs";

const SCENARIO = "accept-invite-stub";
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

test.describe("Scenario: Accept Invite Stub Page", () => {
  test("should render the invite stub with token from URL", async ({ page }) => {
    const testToken = "test-token-abc123";

    // Step 1: Navigate to /accept-invite/<token>
    // This is a public route: PUBLIC_PREFIXES includes "/accept-invite/"
    await page.goto(`/accept-invite/${testToken}`);
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-01-navigate.png"),
      fullPage: true,
    });

    // Step 2: Assert NOT redirected to /login (public route)
    await expect(page).not.toHaveURL(/\/login/);
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-02-not-redirected.png"),
      fullPage: true,
    });

    // Step 3: Assert heading "Invite Acceptance" is visible
    await expect(
      page.getByRole("heading", { name: "Invite Acceptance" })
    ).toBeVisible();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-03-heading.png"),
      fullPage: true,
    });

    // Step 4: Assert stub message text
    await expect(
      page.getByText("Invite acceptance is being set up")
    ).toBeVisible();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-04-stub-message.png"),
      fullPage: true,
    });

    // Step 5: Assert the token value is displayed on the page
    // AcceptInvitePage renders <code>{token}</code>
    await expect(page.locator("code").getByText(testToken)).toBeVisible();
    await page.screenshot({
      path: path.join(RESULTS_DIR, "step-05-token-displayed.png"),
      fullPage: true,
    });
  });
});
