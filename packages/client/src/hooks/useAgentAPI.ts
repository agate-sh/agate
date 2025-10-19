import { useCallback } from 'react';
import { sessionCreate, sessionDelete } from '@agate/sdk';

/**
 * Hook for agent API operations
 * Wraps SDK calls for session management
 */
export function useAgentAPI(baseUrl: string = 'http://localhost:3000') {
  const createSession = useCallback(
    async (worktreePath: string, branch: string, agentName: string) => {
      try {
        const response = await sessionCreate({
          body: {
            worktreePath,
            branch,
            agentName,
          },
        });

        if (response.error) {
          throw new Error(`Failed to create session: ${response.error}`);
        }

        return response.data;
      } catch (error) {
        console.error('Error creating session:', error);
        throw error;
      }
    },
    [baseUrl]
  );

  const deleteSession = useCallback(
    async (sessionId: string) => {
      try {
        const response = await sessionDelete({
          path: {
            id: sessionId,
          },
        });

        if (response.error) {
          throw new Error(`Failed to delete session: ${response.error}`);
        }

        return response.data;
      } catch (error) {
        console.error('Error deleting session:', error);
        throw error;
      }
    },
    [baseUrl]
  );

  return {
    createSession,
    deleteSession,
  };
}
