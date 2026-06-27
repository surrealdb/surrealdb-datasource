// Jest setup provided by Grafana scaffolding
import './.config/jest-setup';

// @grafana/ui transitively imports react-dom/server, whose scheduler relies on
// MessageChannel. jsdom does not implement it, so polyfill it from Node.
import { MessageChannel } from 'worker_threads';

if (typeof global.MessageChannel === 'undefined') {
  global.MessageChannel = MessageChannel;
}
