import { useKeyboard } from "@opentui/react";
import { theme } from "../../utils/theme.js";
import { useAgentsState } from "../../hooks/useAgentsState.js";
import { useWebSocket } from "../../hooks/useWebSocket.js";
import { useAgentAPI } from "../../hooks/useAgentAPI.js";
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
}

export function AgentsPane({ dialog }: AgentsPaneProps) {
  const state = useAgentsState();
  const api = useAgentAPI();

  useWebSocket({
    url: "ws://localhost:3000/ws",
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

  useKeyboard((key) => {
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
          onSelect={(branchName, agentName) => {
            dialog.pop();
            api
              .createSession(process.cwd(), branchName, agentName)
              .catch((error) => {
                console.error("Failed to create session:", error);
              });
          }}
          onClose={() => dialog.pop()}
        />
      );
      return;
    }

    if (key.sequence === "d") {
      const item = state.items[state.selectedIndex];
      if (item?.type === "linked_session") {
        api.deleteSession(item.worktree.id).catch((error) => {
          console.error("Failed to delete session:", error);
        });
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
        borderColor={theme.borderDefault}
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
      borderColor={theme.borderDefault}
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
