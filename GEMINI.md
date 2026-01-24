# GEMINI.md

This file provides context and guidance for Gemini when working with the Monera Digital codebase.

## Project Overview

**Monera Digital** is an institutional-grade digital asset platform focused on static finance and lending solutions.
It is a **full-stack application** utilizing a TypeScript frontend and a **Golang backend**.

**Primary Stack:**
*   **Frontend:** React 18, TypeScript, Vite, Tailwind CSS, Shadcn/Radix UI.
*   **Backend:** **Golang (Go)** - Mandatory for all interfaces, database access, and operations.
*   **Database:** PostgreSQL (Neon).
*   **Caching/State:** Redis (Upstash) for rate limiting and session management.
*   **Testing:** Vitest (Unit/Integration), Playwright (E2E).

## Key Directory Structure

*   **`src/`**: Frontend source code and shared business logic.
    *   `src/components/`: React components (UI primitives in `ui/`, feature components elsewhere).
    *   `src/pages/`: Route components (Login, Register, Dashboard views).
    *   `src/lib/`: Frontend service layer and utilities.
    *   `src/i18n/`: Internationalization (English/Chinese).
*   **`internal/` & `cmd/`**: **Primary Backend Code**. All API handlers, business logic, and database operations reside here.
*   **`docs/`**: Extensive project documentation (Architecture, PRDs, Security).

## Development Workflow

### Commands
*   **Start Dev Server:** `npm run dev` (Vite, defaults to port 8080).
*   **Build:** `npm run build` (Production build to `dist/`).
*   **Test:** `npm test` (Runs Vitest).
*   **Lint:** `npm run lint`.

### Architecture & Patterns

1.  **Service Layer Pattern (`src/lib/`)**:
    *   Business logic is centralized here, *not* in the API handlers or React components.
    *   Services are dependency-injected (accept `db`, `redis` instances).
    *   Example: `auth-service.ts` handles the actual login logic, used by `api/auth/login.ts`.

2.  **Authentication**:
    *   JWT-based. Tokens are stored in `localStorage` (known limitation/choice).
    *   Flow: Login -> JWT -> LocalStorage -> Attached to headers in API calls.
    *   Includes 2FA (TOTP) support.

3.  **Database (Drizzle)**:
    *   Schema defined in `src/db/schema.ts`.
    *   Uses PostgreSQL native enums for status fields.

4.  **UI/UX**:
    *   **Naming Convention:** "Lending" features are exposed to the user as **"Fixed Deposit"** (Sidebar, Hero buttons), though internal code variables often still use `lending`.
    *   **Theme:** Shadcn UI + Tailwind.

## Important Context (Recent Changes)

*   **Fixed Deposit vs. Lending:** As of Jan 2026, the UI term "Lending" has been renamed to "Fixed Deposit" to clarify the product offering. The internal routes (`/dashboard/lending`) and API paths (`api/lending`) remain unchanged.
*   **Mandatory Go Backend:** The project has shifted to a strict **Golang backend** architecture. All new features and bug fixes regarding backend logic, database interactions, and API interfaces **must** be implemented in Go (residing in `internal/` and `cmd/`). Node.js/Vercel functions should only be used for frontend-specific proxying if absolutely necessary, but core logic belongs in Go.

## New Feature Development Rules

**Mandatory rules for developing new features:**

1.  **Technology Stack**
    - **Frontend**: TypeScript
    - **Backend**: Golang (Go) - **MUST** be used for all backend interfaces, database access, and operations.

2.  **Design Principles**
    - **KISS**: Keep code clean and simple.
    - **Architecture**: High Cohesion, Low Coupling. Use streamlined design patterns.

3.  **Testing**
    - **Requirement**: All new functional code must be tested.
    - **Coverage**: Maintain **100% test coverage**.

4.  **Isolation**
    - Changes must **not** affect unrelated functions.

5.  **Proposal Process**
    - Use **openspec** to generate proposals for new features.

## Bug Fixing Rules

**Mandatory rules for fixing bugs:**

1.  **Technology Stack**
    - **Frontend**: TypeScript
    - **Backend**: Golang (Go) - **MUST** be used for all backend interfaces, database access, and operations.

2.  **Design Principles**
    - **KISS**: Keep code clean and simple.
    - **Architecture**: High Cohesion, Low Coupling. Use concise design patterns.

3.  **Testing**
    - **Methodology**: Test-Driven Development (TDD) - **write tests first**.
    - **Requirement**: All new functional code must be tested.
    - **Coverage**: Maintain **100% test coverage**.
    - **Regression**: Perform regression testing after fixes.

4.  **Isolation**
    - Changes must **not** affect unrelated functions.

5.  **Proposal Process**
    - Use **openspec** to generate new bug proposals.
