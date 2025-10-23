import { generateBranchName } from "@agate/shared";

/**
 * Generate a random branch name in format: adjective-noun
 * Delegates to the shared implementation so the client matches Go + CLI behaviour.
 */
export const generateRandomBranchName = generateBranchName;

/**
 * Validate a Git branch name
 * Matches the Go implementation's ValidateBranchName
 */
export function validateBranchName(name: string): string | null {
  if (name === "") {
    return "branch name cannot be empty";
  }
  if (name.startsWith("-") || name.endsWith(".")) {
    return "invalid branch name format";
  }
  if (name.includes("..") || name.includes(" ")) {
    return "branch name contains invalid characters";
  }
  if (name.includes("~") || name.includes("^") || name.includes(":")) {
    return "branch name contains invalid characters";
  }
  return null;
}
