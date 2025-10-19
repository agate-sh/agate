#!/usr/bin/env node
import { render } from "@opentui/react";
import { App } from "./App.js";
import { logger } from "./logger.js";

logger.info('Starting Agate client');

render(<App />);
