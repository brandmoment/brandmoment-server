# Smoke Test Scenarios

Base URL: http://localhost:3000
App: BrandMoment Dashboard (apps/dashboard — Next.js 15)

---

## Scenario: login-page-renders
Tags: auth, rendering
Spec file: tests/smoke/login-page-renders.spec.ts

Steps:
1. Navigate to /login
2. Assert heading "Sign in" is visible
3. Assert email input (label "Email") is visible
4. Assert password input (label "Password") is visible
5. Assert submit button "Sign in" is visible
6. Assert link "Sign up" pointing to /signup is visible

Expected: Login form renders completely without JS errors or blank screen.

---

## Scenario: signup-page-renders
Tags: auth, rendering
Spec file: tests/smoke/signup-page-renders.spec.ts

Steps:
1. Navigate to /signup
2. Assert heading "Create account" is visible
3. Assert input labeled "Full name" is visible
4. Assert input labeled "Email" is visible
5. Assert input labeled "Password" is visible
6. Assert input labeled "Confirm password" is visible
7. Assert submit button "Create account" is visible
8. Assert link "Sign in" pointing to /login is visible

Expected: Signup form renders all four fields and the submit button.

---

## Scenario: unauthenticated-redirect
Tags: auth, middleware
Spec file: tests/smoke/unauthenticated-redirect.spec.ts

Steps:
1. Clear all cookies / ensure no session
2. Navigate to / (dashboard root)
3. Assert final URL contains /login
4. Assert "Sign in" heading is visible on the resulting page

Expected: Next.js middleware redirects unauthenticated visitors from / to /login.
Covers middleware.ts logic: absence of better-auth.session_token cookie triggers redirect.

---

## Scenario: protected-apps-route-redirect
Tags: auth, middleware
Spec file: tests/smoke/protected-apps-route-redirect.spec.ts

Steps:
1. Clear all cookies / ensure no session
2. Navigate to /apps
3. Assert final URL contains /login
4. Assert redirect param in URL contains /apps

Expected: /apps (a protected dashboard route) redirects to /login?redirect=%2Fapps.

---

## Scenario: accept-invite-stub
Tags: auth, rendering
Spec file: tests/smoke/accept-invite-stub.spec.ts

Steps:
1. Navigate to /accept-invite/test-token-abc123
2. Assert heading "Invite Acceptance" is visible
3. Assert text "Invite acceptance is being set up" is visible
4. Assert the token value "test-token-abc123" appears on the page (inside <code> element)

Expected: Accept-invite stub page renders with the token from the URL path parameter.
This is a public route (no redirect to /login).

---

## Scenario: onboarding-page-renders
Tags: auth, rendering, wizard
Spec file: tests/smoke/onboarding-page-renders.spec.ts

Steps:
1. Navigate to /onboarding (no session cookie — may redirect; this tests the redirect, not the form)
2. Assert final URL contains /login (onboarding is protected)

Expected: /onboarding is a protected route; unauthenticated user is redirected to /login.
Note: Full wizard rendering test requires an authenticated session (covered in authenticated suite).

---

## Scenario: login-form-validation
Tags: auth, form-validation
Spec file: tests/smoke/login-form-validation.spec.ts

Steps:
1. Navigate to /login
2. Click "Sign in" button without filling any fields
3. Assert validation error "Enter a valid email address" appears
4. Assert validation error "Password is required" appears

Expected: Client-side zod/react-hook-form validation fires on empty submit, showing inline error messages.

---

## Scenario: signup-form-validation
Tags: auth, form-validation
Spec file: tests/smoke/signup-form-validation.spec.ts

Steps:
1. Navigate to /signup
2. Click "Create account" button without filling any fields
3. Assert validation error "Name is required" appears
4. Assert validation error "Enter a valid email address" appears

Expected: Signup form shows inline validation errors on empty submit without making API calls.

---

## Scenario: login-to-signup-navigation
Tags: auth, navigation
Spec file: tests/smoke/login-to-signup-navigation.spec.ts

Steps:
1. Navigate to /login
2. Click "Sign up" link
3. Assert URL is /signup
4. Assert heading "Create account" is visible

Expected: Navigation link from login to signup works correctly.

---

## Scenario: signup-to-login-navigation
Tags: auth, navigation
Spec file: tests/smoke/signup-to-login-navigation.spec.ts

Steps:
1. Navigate to /signup
2. Click "Sign in" link
3. Assert URL is /login
4. Assert heading "Sign in" is visible

Expected: Navigation link from signup back to login works correctly.
