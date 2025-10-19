import { useKeyboard } from "@opentui/react";
import { AGATE_SERVER_PORT } from "@agate/shared";
import { theme } from "../../utils/theme.js";
import { useAgentsState } from "../../hooks/useAgentsState.js";
import { useWebSocket } from "../../hooks/useWebSocket.js";
import * as api from "../../api.js";
import type { DialogContext } from "../../hooks/useDialog.js";
import { NewAgentDialog } from "../NewAgentDialog.js";
import type { AgentListItem } from "./types.js";
import { EmptyState } from "./EmptyState.js";
import { RepoHeader } from "./RepoHeader.js";
import { SectionHeader } from "./SectionHeader.js";
import { SessionItem } from "./SessionItem.js";
import { EmptyMessage } from "./EmptyMessage.js";

interface ItemRendererProps {
  item: AgentListItem;
  isSelected: boolean;
  isHovered: boolean;
  expandedRepos: Set<string>;
  currentRepo: string | null;
}

function ItemRenderer({
  item,
  isSelected,
  isHovered,
  expandedRepos,
  currentRepo,
}: ItemRendererProps) {
  switch (item.type) {
    case "repo_header":
      return (
        <RepoHeader
          repoName={item.repoName}
          isExpanded={expandedRepos.has(item.repoName)}
          isHovered={isHovered}
          isCurrent={item.repoName === currentRepo}
        />
      );
    case "section_header":
      return <SectionHeader title={item.sectionTitle} isHovered={isHovered} />;
    case "main_session":
      return (
        <SessionItem
          worktree={item.worktree}
          isPinned={item.isPinned}
          isLinked={false}
          isSelected={isSelected}
          isHovered={isHovered}
        />
      );
    case "linked_session":
      return (
        <SessionItem
          worktree={item.worktree}
          isPinned={item.isPinned}
          isLinked={true}
          isSelected={isSelected}
          isHovered={isHovered}
        />
      );
    case "empty_message":
      return <EmptyMessage sectionType={item.sectionTitle} />;
    case "gap":
      return <box height={1} />;
    default:
      return null;
  }
}

interface AgentsPaneProps {
  dialog: DialogContext;
  isFocused: boolean;
}

export function AgentsPane({ dialog, isFocused }: AgentsPaneProps) {
  const state = useAgentsState();

  useWebSocket({
    url: `ws://localhost:${AGATE_SERVER_PORT}/ws`,
    onSessionCreated: (session) => {
      state.setSessions((prev) => [...prev, session]);
    },
    onSessionDeleted: (sessionId) => {
      state.setSessions((prev) => prev.filter((s) => s.id !== sessionId));
    },
    onSessionUpdated: (session) => {
      state.setSessions((prev) =>
        prev.map((s) => (s.id === session.id ? session : s)),
      );
    },
  });

  useKeyboard(async (key) => {
    // Only handle keyboard input when this pane is focused
    if (!isFocused) return;

    if (key.name === "up" || key.sequence === "k") {
      state.moveSelectionUp();
      return;
    }

    if (key.name === "down" || key.sequence === "j") {
      state.moveSelectionDown();
      return;
    }

    if (key.name === "return") {
      const item = state.items[state.selectedIndex];
      if (!item) return;

      if (item.type === "repo_header") {
        state.toggleRepo(item.repoName);
      } else if (
        item.type === "main_session" ||
        item.type === "linked_session"
      ) {
        state.setPinnedWorktree(item.worktree);
      }
      return;
    }

    if (key.sequence === "n") {
      const repoName = state.currentRepo || "Repository";
      dialog.push(
        <NewAgentDialog
          repoName={repoName}
          onSelect={async (branchName, agentName) => {
            dialog.pop();
            const response = await api.sessionCreate({
              body: {
                worktreePath: process.cwd(),
                branch: branchName,
                agentName,
              },
            });
            if (response.error) {
              console.error("Failed to create session:", response.error);
            }
          }}
          onClose={() => dialog.pop()}
        />
      );
      return;
    }

    if (key.sequence === "d") {
      const item = state.items[state.selectedIndex];
      if (item?.type === "linked_session") {
        const response = await api.sessionDelete({
          path: { id: item.worktree.id },
        });
        if (response.error) {
          console.error("Failed to delete session:", response.error);
        }
      }
    }
  });

  if (state.items.length === 0) {
    return (
      <box
        width="30%"
        minWidth={24}
        height="100%"
        borderStyle="single"
        borderColor={isFocused ? theme.agate : theme.borderDefault}
        flexDirection="column"
      >
        <EmptyState />
      </box>
    );
  }

  return (
    <box
      width="30%"
      minWidth={24}
      height="100%"
      borderStyle="single"
      borderColor={isFocused ? theme.agate : theme.borderDefault}
      title="Agents"
      flexDirection="column"
    >
      {state.items.map((item, idx) => (
        <box key={idx}>
          <ItemRenderer
            item={item}
            isSelected={idx === state.selectedIndex}
            isHovered={idx === state.selectedIndex}
            expandedRepos={state.expandedRepos}
            currentRepo={state.currentRepo}
          />
        </box>
      ))}
    </box>
  );
}
