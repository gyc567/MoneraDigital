import { render, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import App from "./App";

describe("registration route", () => {
  beforeEach(() => {
    localStorage.clear();
    window.history.pushState({}, "", "/");
    vi.stubGlobal("matchMedia", vi.fn().mockReturnValue({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }));
  });

  it("redirects visitors from registration to login", async () => {
    window.history.pushState({}, "", "/register");

    render(<App />);

    await waitFor(() => {
      expect(window.location.pathname).toBe("/login");
    });
  });
});
