/**
 * Generate a random branch name in format: adjective-noun
 * Matches the Go implementation's GenerateRandomBranchName
 */
export function generateRandomBranchName(): string {
  const adjectives = [
    "quick",
    "bright",
    "swift",
    "clever",
    "bold",
    "neat",
    "clean",
    "smooth",
    "sharp",
    "cool",
  ];
  const nouns = [
    "fix",
    "update",
    "patch",
    "change",
    "work",
    "task",
    "feature",
    "test",
    "demo",
    "trial",
  ];

  const adj = adjectives[Math.floor(Math.random() * adjectives.length)];
  const noun = nouns[Math.floor(Math.random() * nouns.length)];

  return `${adj}-${noun}`;
}

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
