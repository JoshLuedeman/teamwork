import { Env, WebhookPayload, TreeEntry } from "./types";
import { verifySignature } from "./verify";
import { createJWT, getInstallationToken } from "./auth";
import {
  fetchSourceTree,
  getBlob,
  createBlob,
  createTree,
  createCommit,
  updateRef,
  createRef,
  checkFileExists,
} from "./github";
import { STARTER_TEMPLATES } from "./files";

const COMMIT_MESSAGE =
  "Initialize Teamwork framework\n\nAuto-installed by teamwork-installer GitHub App";

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    // 1. Only accept POST
    if (request.method !== "POST") {
      return new Response("Method not allowed", { status: 405 });
    }

    // 2. Read raw body
    const body = await request.text();

    // 3. Verify webhook signature
    const signature = request.headers.get("X-Hub-Signature-256") ?? "";
    const valid = await verifySignature(body, signature, env.GITHUB_WEBHOOK_SECRET);
    if (!valid) {
      return new Response("Invalid signature", { status: 401 });
    }

    // 4. Parse JSON body
    let payload: WebhookPayload;
    try {
      payload = JSON.parse(body) as WebhookPayload;
    } catch {
      return new Response("Invalid JSON", { status: 400 });
    }

    // 5. Check event type
    const event = request.headers.get("X-GitHub-Event") ?? "";
    if (event !== "repository") {
      return new Response("Event ignored", { status: 200 });
    }

    // 6. Check action
    if (payload.action !== "created") {
      return new Response("Action ignored", { status: 200 });
    }

    // 7. Skip forks
    if (payload.repository.fork) {
      return new Response("Skipped (fork)", { status: 200 });
    }

    try {
      const { repository, installation } = payload;
      const owner = repository.owner.login;
      const repo = repository.name;
      const defaultBranch = repository.default_branch || "main";

      // 8. Get installation token
      const jwt = await createJWT(env.GITHUB_APP_ID, env.GITHUB_APP_PRIVATE_KEY);
      const token = await getInstallationToken(jwt, installation.id);

      // 9. Check for .teamwork-skip marker file
      const shouldSkip = await checkFileExists(token, owner, repo, ".teamwork-skip");
      if (shouldSkip) {
        return new Response("Skipped (.teamwork-skip exists)", { status: 200 });
      }

      // 10. Fetch source tree from framework repo
      const sourceTree = await fetchSourceTree(
        token,
        env.SOURCE_REPO_OWNER,
        env.SOURCE_REPO_NAME,
        env.SOURCE_REF,
      );

      // 11. Copy framework file blobs to target repo
      const treeEntries: TreeEntry[] = [];

      for (const entry of sourceTree) {
        const blobContent = await getBlob(
          token,
          env.SOURCE_REPO_OWNER,
          env.SOURCE_REPO_NAME,
          entry.sha,
        );
        const newSha = await createBlob(token, owner, repo, blobContent, "base64");
        treeEntries.push({
          path: entry.path,
          mode: entry.mode,
          type: "blob",
          sha: newSha,
        });
      }

      // 12. Create starter template blobs
      for (const [path, content] of Object.entries(STARTER_TEMPLATES)) {
        const sha = await createBlob(token, owner, repo, content, "utf-8");
        treeEntries.push({
          path,
          mode: "100644",
          type: "blob",
          sha,
        });
      }

      // 13. Determine if repo is empty and get base tree if it exists
      let parentSha: string | undefined;
      let baseTree: string | undefined;
      let isEmptyRepo = false;

      try {
        const refResponse = await fetch(
          `https://api.github.com/repos/${owner}/${repo}/git/refs/heads/${defaultBranch}`,
          {
            headers: {
              Authorization: `Bearer ${token}`,
              Accept: "application/vnd.github.v3+json",
              "User-Agent": "teamwork-installer",
            },
          },
        );

        if (refResponse.ok) {
          const refData = (await refResponse.json()) as {
            object: { sha: string };
          };
          parentSha = refData.object.sha;

          // Get commit to find the base tree
          const commitResponse = await fetch(
            `https://api.github.com/repos/${owner}/${repo}/git/commits/${parentSha}`,
            {
              headers: {
                Authorization: `Bearer ${token}`,
                Accept: "application/vnd.github.v3+json",
                "User-Agent": "teamwork-installer",
              },
            },
          );

          if (commitResponse.ok) {
            const commitData = (await commitResponse.json()) as {
              tree: { sha: string };
            };
            baseTree = commitData.tree.sha;
          }
        } else {
          isEmptyRepo = true;
        }
      } catch {
        isEmptyRepo = true;
      }

      // 14. Create tree and commit
      const treeSha = await createTree(
        token,
        owner,
        repo,
        treeEntries,
        baseTree,
      );

      const parents = parentSha ? [parentSha] : [];
      const commitSha = await createCommit(
        token,
        owner,
        repo,
        COMMIT_MESSAGE,
        treeSha,
        parents,
      );

      // 15. Create or update ref
      if (isEmptyRepo) {
        await createRef(
          token,
          owner,
          repo,
          `heads/${defaultBranch}`,
          commitSha,
        );
      } else {
        await updateRef(
          token,
          owner,
          repo,
          `heads/${defaultBranch}`,
          commitSha,
        );
      }

      // 16. Success
      console.log(`Installed Teamwork framework in ${repository.full_name}`);
      return new Response(
        `Installed Teamwork framework in ${repository.full_name}`,
        { status: 200 },
      );
    } catch (error) {
      console.error("Failed to install Teamwork framework:", error);
      return new Response("Internal server error", { status: 500 });
    }
  },
};
