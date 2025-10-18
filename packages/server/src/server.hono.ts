import { Hono } from 'hono';
import { cors } from 'hono/cors';
import { describeRoute, resolver, openAPIRouteHandler, generateSpecs } from 'hono-openapi';
import z from 'zod';
import { EventBus } from './event-bus.js';
import { StateManager } from './state/manager.js';
import { gitRouter } from './routes/git.hono.js';
import { sessionRouter } from './routes/session.hono.js';

type Env = {
  Variables: {
    eventBus: EventBus;
    stateManager: StateManager;
  };
};

/**
 * Create and configure the Hono application
 * This replaces the Express server with Hono + hono-openapi
 */
export function createHonoServer(eventBus: EventBus, stateManager: StateManager) {
  const app = new Hono<Env>();

  // Middleware to inject dependencies into context
  app.use('*', async (c, next) => {
    c.set('eventBus', eventBus);
    c.set('stateManager', stateManager);
    await next();
  });

  // CORS middleware
  app.use('*', cors());

  // Error handling
  app.onError((err, c) => {
    console.error('Error:', err);
    return c.json(
      {
        error: {
          message: err.message || 'Internal server error',
          ...(process.env.NODE_ENV === 'development' && { stack: err.stack }),
        },
      },
      500
    );
  });

  // OpenAPI documentation endpoint (Swagger UI)
  app.get(
    '/doc',
    openAPIRouteHandler(app, {
      documentation: {
        info: {
          title: 'Agate Server API',
          version: '1.0.0',
          description: 'Agate terminal multiplexer API',
        },
        openapi: '3.1.0',
      },
    })
  );

  // OpenAPI spec as JSON
  app.get(
    '/openapi.json',
    describeRoute({
      description: 'Get OpenAPI specification',
      operationId: 'openapi.spec',
      responses: {
        200: {
          description: 'OpenAPI specification',
          content: {
            'application/json': {
              schema: resolver(z.any()),
            },
          },
        },
      },
    }),
    async (c) => {
      const spec = await generateOpenAPISpec(app);
      return c.json(spec);
    }
  );

  // Health check endpoint
  app.get(
    '/health',
    describeRoute({
      description: 'Health check endpoint',
      operationId: 'health.check',
      responses: {
        200: {
          description: 'Server is healthy',
          content: {
            'application/json': {
              schema: resolver(
                z.object({
                  status: z.string(),
                  timestamp: z.string(),
                })
              ),
            },
          },
        },
      },
    }),
    (c) => {
      return c.json({ status: 'ok', timestamp: new Date().toISOString() });
    }
  );

  // Mount API routes
  app.route('/git', gitRouter);
  app.route('/session', sessionRouter);

  return app;
}

/**
 * Generate OpenAPI spec from Hono routes
 */
export async function generateOpenAPISpec(app: Hono): Promise<any> {
  const spec = await generateSpecs(app, {
    documentation: {
      info: {
        title: 'Agate Server API',
        version: '1.0.0',
        description: 'Agate terminal multiplexer API - auto-generated from route definitions',
      },
      openapi: '3.1.0',
    },
  });
  return spec;
}
