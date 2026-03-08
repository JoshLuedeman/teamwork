/** Prefixes and exact paths that identify Teamwork framework files. */
export const FRAMEWORK_PREFIXES = [
  ".github/agents/",
  ".github/skills/",
  ".github/instructions/",
  ".github/copilot-instructions.md",
  ".github/ISSUE_TEMPLATE/",
  ".github/PULL_REQUEST_TEMPLATE.md",
  "docs/",
  ".editorconfig",
  ".pre-commit-config.yaml",
  "Makefile",
];

/** Starter template files created in every new repository. */
export const STARTER_TEMPLATES: Record<string, string> = {
  "MEMORY.md":
    "# Project Memory\n\nThis file captures project learnings that persist across agent sessions.\n",
  "CHANGELOG.md":
    "# Changelog\n\nAll notable changes to this project will be documented in this file.\n\nThe format is based on [Keep a Changelog](https://keepachangelog.com/).\n",
  "README.md": "# Project\n",
};

/**
 * Check whether a file path belongs to the Teamwork framework.
 *
 * Directory prefixes match any path starting with that prefix.
 * Non-directory entries require an exact match.
 */
export function isFrameworkFile(path: string): boolean {
  return FRAMEWORK_PREFIXES.some((prefix) =>
    prefix.endsWith("/") ? path.startsWith(prefix) : path === prefix,
  );
}
