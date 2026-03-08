import { TreeEntry } from "./types";
import { isFrameworkFile } from "./files";

const GITHUB_API = "https://api.github.com";

/** Standard headers for GitHub API requests. */
function headers(token: string): Record<string, string> {
  return {
    Authorization: `Bearer ${token}`,
    Accept: "application/vnd.github.v3+json",
    "User-Agent": "teamwork-installer",
  };
}

/**
 * Fetch the recursive tree from the source repository, filtered to
 * framework files only.
 */
export async function fetchSourceTree(
  token: string,
  owner: string,
  repo: string,
  ref: string,
): Promise<TreeEntry[]> {
  const response = await fetch(
    `${GITHUB_API}/repos/${owner}/${repo}/git/trees/${ref}?recursive=1`,
    { headers: headers(token) },
  );

  if (!response.ok) {
    throw new Error(
      `Failed to fetch source tree: ${response.status} ${response.statusText}`,
    );
  }

  const data = (await response.json()) as { tree: TreeEntry[] };
  return data.tree.filter(
    (entry) => entry.type === "blob" && isFrameworkFile(entry.path),
  );
}

/** Get blob content (base64-encoded) by SHA. */
export async function getBlob(
  token: string,
  owner: string,
  repo: string,
  sha: string,
): Promise<string> {
  const response = await fetch(
    `${GITHUB_API}/repos/${owner}/${repo}/git/blobs/${sha}`,
    { headers: headers(token) },
  );

  if (!response.ok) {
    throw new Error(
      `Failed to get blob: ${response.status} ${response.statusText}`,
    );
  }

  const data = (await response.json()) as { content: string };
  return data.content;
}

/** Create a blob in the target repository and return its SHA. */
export async function createBlob(
  token: string,
  owner: string,
  repo: string,
  content: string,
  encoding: string,
): Promise<string> {
  const response = await fetch(
    `${GITHUB_API}/repos/${owner}/${repo}/git/blobs`,
    {
      method: "POST",
      headers: { ...headers(token), "Content-Type": "application/json" },
      body: JSON.stringify({ content, encoding }),
    },
  );

  if (!response.ok) {
    throw new Error(
      `Failed to create blob: ${response.status} ${response.statusText}`,
    );
  }

  const data = (await response.json()) as { sha: string };
  return data.sha;
}

/** Create a tree in the target repository and return its SHA. */
export async function createTree(
  token: string,
  owner: string,
  repo: string,
  entries: TreeEntry[],
  baseTree?: string,
): Promise<string> {
  const body: Record<string, unknown> = {
    tree: entries.map((e) => ({
      path: e.path,
      mode: e.mode,
      type: e.type,
      sha: e.sha,
    })),
  };
  if (baseTree) {
    body.base_tree = baseTree;
  }

  const response = await fetch(
    `${GITHUB_API}/repos/${owner}/${repo}/git/trees`,
    {
      method: "POST",
      headers: { ...headers(token), "Content-Type": "application/json" },
      body: JSON.stringify(body),
    },
  );

  if (!response.ok) {
    throw new Error(
      `Failed to create tree: ${response.status} ${response.statusText}`,
    );
  }

  const data = (await response.json()) as { sha: string };
  return data.sha;
}

/** Create a commit in the target repository and return its SHA. */
export async function createCommit(
  token: string,
  owner: string,
  repo: string,
  message: string,
  treeSha: string,
  parents: string[],
): Promise<string> {
  const response = await fetch(
    `${GITHUB_API}/repos/${owner}/${repo}/git/commits`,
    {
      method: "POST",
      headers: { ...headers(token), "Content-Type": "application/json" },
      body: JSON.stringify({ message, tree: treeSha, parents }),
    },
  );

  if (!response.ok) {
    throw new Error(
      `Failed to create commit: ${response.status} ${response.statusText}`,
    );
  }

  const data = (await response.json()) as { sha: string };
  return data.sha;
}

/** Update an existing ref to point to a new SHA. */
export async function updateRef(
  token: string,
  owner: string,
  repo: string,
  ref: string,
  sha: string,
): Promise<void> {
  const response = await fetch(
    `${GITHUB_API}/repos/${owner}/${repo}/git/refs/${ref}`,
    {
      method: "PATCH",
      headers: { ...headers(token), "Content-Type": "application/json" },
      body: JSON.stringify({ sha }),
    },
  );

  if (!response.ok) {
    throw new Error(
      `Failed to update ref: ${response.status} ${response.statusText}`,
    );
  }
}

/** Create a new ref pointing to a SHA. */
export async function createRef(
  token: string,
  owner: string,
  repo: string,
  ref: string,
  sha: string,
): Promise<void> {
  const response = await fetch(
    `${GITHUB_API}/repos/${owner}/${repo}/git/refs`,
    {
      method: "POST",
      headers: { ...headers(token), "Content-Type": "application/json" },
      body: JSON.stringify({ ref: `refs/${ref}`, sha }),
    },
  );

  if (!response.ok) {
    throw new Error(
      `Failed to create ref: ${response.status} ${response.statusText}`,
    );
  }
}

/** Check whether a file exists in a repository via the Contents API. */
export async function checkFileExists(
  token: string,
  owner: string,
  repo: string,
  path: string,
): Promise<boolean> {
  const response = await fetch(
    `${GITHUB_API}/repos/${owner}/${repo}/contents/${path}`,
    {
      method: "HEAD",
      headers: headers(token),
    },
  );

  return response.ok;
}
