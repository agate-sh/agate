import { z } from 'zod';

/**
 * Zod schemas for Git-related API responses
 */

export const FileStatusSchema = z.object({
  path: z.string(),
  index: z.string(),
  working_dir: z.string(),
  additions: z.number().optional(),
  deletions: z.number().optional(),
});

export const GitStatusSchema = z.object({
  files: z.array(FileStatusSchema),
  staged: z.array(z.string()),
  modified: z.array(z.string()),
  not_added: z.array(z.string()),
  conflicted: z.array(z.string()),
  current: z.string().nullable(),
  tracking: z.string().nullable(),
  ahead: z.number(),
  behind: z.number(),
});

export const WorktreeSchema = z.object({
  path: z.string(),
  branch: z.string(),
  commit: z.string(),
  isMain: z.boolean(),
  bare: z.boolean(),
  detached: z.boolean(),
});

export const WorktreesResponseSchema = z.object({
  worktrees: z.array(WorktreeSchema),
});

export const RepoInfoSchema = z.object({
  repoRoot: z.string(),
  repoName: z.string(),
  currentBranch: z.string(),
});
