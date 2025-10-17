import type { ServerEvent } from '@agate/shared';

type EventCallback = (event: ServerEvent) => void;

/**
 * EventBus for pub/sub messaging
 * Transport-agnostic (no longer SSE-specific)
 */
export class EventBus {
  private subscribers: Map<string, EventCallback> = new Map();

  /**
   * Subscribe to all events
   */
  subscribe(id: string, callback: EventCallback): void {
    this.subscribers.set(id, callback);
    console.log(`Subscriber added: ${id} (total: ${this.subscribers.size})`);
  }

  /**
   * Unsubscribe from events
   */
  unsubscribe(id: string): void {
    const removed = this.subscribers.delete(id);
    if (removed) {
      console.log(`Subscriber removed: ${id} (total: ${this.subscribers.size})`);
    }
  }

  /**
   * Publish an event to all subscribers
   */
  publish(event: ServerEvent): void {
    if (this.subscribers.size === 0) {
      return;
    }

    this.subscribers.forEach((callback) => {
      try {
        callback(event);
      } catch (error) {
        console.error('Error in event subscriber:', error);
      }
    });
  }

  /**
   * Get number of active subscribers
   */
  getSubscriberCount(): number {
    return this.subscribers.size;
  }

  /**
   * Cleanup all subscribers
   */
  destroy(): void {
    this.subscribers.clear();
  }
}
