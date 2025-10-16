/**
 * WebSocket Context - React context for sharing WebSocket connection across components
 * Provides global WebSocket state and functionality
 */

import { createContext } from 'react';
import {
  WebSocketContextValue,
} from '../../types/websocket';

export const WebSocketContext = createContext<WebSocketContextValue | null>(null);