import { describe, it, expect } from "vitest";
import { isFrameworkFile, FRAMEWORK_PREFIXES, STARTER_TEMPLATES } from "../src/files";

describe("isFrameworkFile", () => {
  describe("matches framework directory prefixes", () => {
    it("matches .github/agents/ files", () => {
      expect(isFrameworkFile(".github/agents/coder.agent.md")).toBe(true);
    });

    it("matches .github/skills/ nested files", () => {
      expect(isFrameworkFile(".github/skills/feature/SKILL.md")).toBe(true);
    });

    it("matches .github/instructions/ files", () => {
      expect(isFrameworkFile(".github/instructions/coding.md")).toBe(true);
    });

    it("matches .github/ISSUE_TEMPLATE/ files", () => {
      expect(isFrameworkFile(".github/ISSUE_TEMPLATE/bug_report.md")).toBe(true);
    });

    it("matches docs/ files", () => {
      expect(isFrameworkFile("docs/conventions.md")).toBe(true);
    });

    it("matches docs/ nested files", () => {
      expect(isFrameworkFile("docs/decisions/001-architecture.md")).toBe(true);
    });
  });

  describe("matches exact framework file paths", () => {
    it("matches Makefile", () => {
      expect(isFrameworkFile("Makefile")).toBe(true);
    });

    it("matches .editorconfig", () => {
      expect(isFrameworkFile(".editorconfig")).toBe(true);
    });

    it("matches .pre-commit-config.yaml", () => {
      expect(isFrameworkFile(".pre-commit-config.yaml")).toBe(true);
    });

    it("matches .github/copilot-instructions.md", () => {
      expect(isFrameworkFile(".github/copilot-instructions.md")).toBe(true);
    });

    it("matches .github/PULL_REQUEST_TEMPLATE.md", () => {
      expect(isFrameworkFile(".github/PULL_REQUEST_TEMPLATE.md")).toBe(true);
    });
  });

  describe("rejects non-framework files", () => {
    it("rejects internal Go files", () => {
      expect(isFrameworkFile("internal/installer/installer.go")).toBe(false);
    });

    it("rejects cmd Go files", () => {
      expect(isFrameworkFile("cmd/teamwork/main.go")).toBe(false);
    });

    it("rejects go.mod", () => {
      expect(isFrameworkFile("go.mod")).toBe(false);
    });

    it("rejects go.sum", () => {
      expect(isFrameworkFile("go.sum")).toBe(false);
    });

    it("rejects Dockerfile", () => {
      expect(isFrameworkFile("Dockerfile")).toBe(false);
    });

    it("rejects root-level README.md", () => {
      // README.md is a starter template, not a framework file
      expect(isFrameworkFile("README.md")).toBe(false);
    });

    it("rejects random files at root", () => {
      expect(isFrameworkFile("package.json")).toBe(false);
    });

    it("rejects Makefile-like paths that aren't exact", () => {
      expect(isFrameworkFile("src/Makefile")).toBe(false);
    });
  });
});

describe("FRAMEWORK_PREFIXES", () => {
  it("is a non-empty array", () => {
    expect(FRAMEWORK_PREFIXES.length).toBeGreaterThan(0);
  });
});

describe("STARTER_TEMPLATES", () => {
  it("contains MEMORY.md", () => {
    expect(STARTER_TEMPLATES["MEMORY.md"]).toBeDefined();
  });

  it("contains CHANGELOG.md", () => {
    expect(STARTER_TEMPLATES["CHANGELOG.md"]).toBeDefined();
  });

  it("contains README.md", () => {
    expect(STARTER_TEMPLATES["README.md"]).toBeDefined();
  });

  it("MEMORY.md starts with expected heading", () => {
    expect(STARTER_TEMPLATES["MEMORY.md"]).toContain("# Project Memory");
  });

  it("CHANGELOG.md references Keep a Changelog", () => {
    expect(STARTER_TEMPLATES["CHANGELOG.md"]).toContain("Keep a Changelog");
  });
});
