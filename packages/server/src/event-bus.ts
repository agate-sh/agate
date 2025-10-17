import type { Response } from 'express';
import type { ServerEvent } from '@agate/shared';

type Subscriber = {
  id: string;
  res: Response;
};

/**
 * EventBus for Server-Sent Events
 * Supports multiple concurrent subscribers
 */
export class EventBus {
  private subscribers: Map<string, Subscriber> = new Map();
  private keepaliveInterval: NodeJS.Timeout | null = null;
  private readonly KEEPALIVE_INTERVAL_MS = 30000; // 30 seconds

  constructor() {
    this.startKeepalive();
  }

  /**
   * Subscribe a client to the event stream
   */
  subscribe(id: string, res: Response): void {
    // Set SSE headers
    res.setHeader('Content-Type', 'text/event-stream');
    res.setHeader('Cache-Control', 'no-cache');
    res.setHeader('Connection', 'keep-alive');
    res.setHeader('X-Accel-Buffering', 'no'); // Disable nginx buffering

    // Handle client disconnect
    res.on('close', () => {
      this.unsubscribe(id);
    });

    this.subscribers.set(id, { id, res });
    console.log(`Client subscribed: ${id} (total: ${this.subscribers.size})`);

    // Send initial keepalive
    this.sendToClient(id, {
      type: 'keepalive',
      timestamp: new Date().toISOString(),
    });
  }

  /**
   * Unsubscribe a client from the event stream
   */
  unsubscribe(id: string): void {
    const removed = this.subscribers.delete(id);
    if (removed) {
      console.log(`Client unsubscribed: ${id} (total: ${this.subscribers.size})`);
    }
  }

  /**
   * Publish an event to all subscribers
   */
  publish(event: ServerEvent): void {
    if (this.subscribers.size === 0) {
      return;
    }

    const data = JSON.stringify(event);
    this.subscribers.forEach((subscriber) => {
      this.sendRaw(subscriber.res, event.type, data);
    });
  }

  /**
   * Send an event to a specific client
   */
  private sendToClient(clientId: string, event: ServerEvent): void {
    const subscriber = this.subscribers.get(clientId);
    if (subscriber) {
      const data = JSON.stringify(event);
      this.sendRaw(subscriber.res, event.type, data);
    }
  }

  /**
   * Send raw SSE formatted data
   */
  private sendRaw(res: Response, event: string, data: string): void {
    res.write(`event: ${event}\n`);
    res.write(`data: ${data}\n\n`);
  }

  /**
   * Start keepalive ping to prevent connection timeouts
   */
  private startKeepalive(): void {
    this.keepaliveInterval = setInterval(() => {
      if (this.subscribers.size === 0) {
        return;
      }

      const keepaliveEvent: ServerEvent = {
        type: 'keepalive',
        timestamp: new Date().toISOString(),
      };
      this.publish(keepaliveEvent);
    }, this.KEEPALIVE_INTERVAL_MS);
  }

  /**
   * Stop keepalive ping and disconnect all clients
   */
  destroy(): void {
    if (this.keepaliveInterval) {
      clearInterval(this.keepaliveInterval);
      this.keepaliveInterval = null;
    }
    this.subscribers.clear();
  }

  /**
   * Get number of active subscribers
   */
  getSubscriberCount(): number {
    return this.subscribers.size;
  }
}
