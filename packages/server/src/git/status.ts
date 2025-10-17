import { SimpleGit, StatusResult } from 'simple-git';

export interface FileStatus {
  path: string;
  index: string;
  working_dir: string;
  additions?: number;
  deletions?: number;
}

export interface GitStatus {
  files: FileStatus[];
  staged: string[];
  modified: string[];
  not_added: string[];
  conflicted: string[];
  current: string | null;
  tracking: string | null;
  ahead: number;
  behind: number;
}

/**
 * Gets detailed file status including additions/deletions per file.
 */
export async function getFileStatus(git: SimpleGit): Promise<GitStatus> {
  const status: StatusResult = await git.status();

  const files: FileStatus[] = [];
  const staged: string[] = [];
  const modified: string[] = [];
  const not_added: string[] = [];
  const conflicted: string[] = [];

  // Process each file in the status
  for (const file of status.files) {
    const fileStatus: FileStatus = {
      path: file.path,
      index: file.index,
      working_dir: file.working_dir,
    };

    // Calculate additions and deletions for this file
    try {
      const diffSummary = await git.diffSummary(['HEAD', '--', file.path]);
      if (diffSummary.files.length > 0) {
        const firstFile = diffSummary.files[0];
        if (firstFile && 'insertions' in firstFile) {
          fileStatus.additions = firstFile.insertions;
          fileStatus.deletions = firstFile.deletions;
        }
      }
    } catch (error) {
      // File might be untracked, diff against /dev/null
      try {
        const diffSummary = await git.diffSummary(['--', file.path]);
        if (diffSummary.files.length > 0) {
          const firstFile = diffSummary.files[0];
          if (firstFile && 'insertions' in firstFile) {
            fileStatus.additions = firstFile.insertions;
            fileStatus.deletions = firstFile.deletions;
          }
        }
      } catch {
        // If diff fails, set to 0
        fileStatus.additions = 0;
        fileStatus.deletions = 0;
      }
    }

    files.push(fileStatus);

    // Categorize files
    if (file.index !== ' ' && file.index !== '?') {
      staged.push(file.path);
    }
    if (file.working_dir === 'M') {
      modified.push(file.path);
    }
    if (file.index === '?' && file.working_dir === '?') {
      not_added.push(file.path);
    }
    if (file.index === 'U' || file.working_dir === 'U') {
      conflicted.push(file.path);
    }
  }

  return {
    files,
    staged,
    modified,
    not_added,
    conflicted,
    current: status.current ?? null,
    tracking: status.tracking ?? null,
    ahead: status.ahead,
    behind: status.behind,
  };
}
