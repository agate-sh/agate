import { client } from "@agate/sdk";
import { AGATE_SERVER_PORT } from "@agate/shared";

/**
 * Configure SDK client on module load
 */
client.setConfig({
  baseUrl: `http://localhost:${AGATE_SERVER_PORT}`,
});

// Re-export everything from SDK for convenience
export * from "@agate/sdk";
