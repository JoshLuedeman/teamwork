/** GitHub repository.created webhook payload. */
export interface WebhookPayload {
  action: string;
  repository: {
    name: string;
    owner: {
      login: string;
    };
    full_name: string;
    fork: boolean;
    default_branch: string;
  };
  installation: {
    id: number;
  };
}

/** Git tree entry for the GitHub Trees API. */
export interface TreeEntry {
  path: string;
  mode: string;
  type: string;
  sha: string;
  content?: string;
}

/** Worker environment bindings. */
export interface Env {
  GITHUB_APP_ID: string;
  GITHUB_APP_PRIVATE_KEY: string;
  GITHUB_WEBHOOK_SECRET: string;
  SOURCE_REPO_OWNER: string;
  SOURCE_REPO_NAME: string;
  SOURCE_REF: string;
}
