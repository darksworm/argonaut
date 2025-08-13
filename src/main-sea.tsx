import {render} from "ink";
import React from "react";
import {App} from "./components/App";

(function setupAlternateScreen() {
    if (typeof process === 'undefined') return;
    const out = process.stdout as any;
    const isTTY = !!out && typeof out.isTTY === 'boolean' ? out.isTTY : false;
    if (!isTTY) return;

    let cleaned = false;
    const enable = () => {
        try {
            out.write("\u001B[?1049h");
        } catch {
        }
    };
    const disable = () => {
        if (cleaned) return;
        cleaned = true;
        try {
            out.write("\u001B[?1049l");
        } catch {
        }
    };

    enable();

    process.on('exit', disable);
    process.on('SIGINT', () => {
        disable();
        process.exit(130);
    });
    process.on('SIGTERM', () => {
        disable();
        process.exit(143);
    });
    process.on('SIGHUP', () => {
        disable();
        process.exit(129);
    });
    process.on('uncaughtException', (err) => {
        disable();
        console.error(err);
        process.exit(1);
    });
    process.on('unhandledRejection', (reason) => {
        disable();
        console.error(reason);
        process.exit(1);
    });
})();

// For SEA builds, wait for Yoga WASM to load properly
async function startApp() {
    // Import yoga-layout to ensure it's loaded
    const yoga = await import('yoga-layout');
    
    // Wait for the WASM to be ready
    let attempts = 0;
    while (attempts < 50) { // Max 5 seconds
        try {
            // Try to access a Yoga function to see if it's ready
            if (yoga.default && typeof yoga.default === 'object') {
                // Give it one more moment
                await new Promise(resolve => setTimeout(resolve, 50));
                break;
            }
        } catch (e) {
            // Still loading
        }
        attempts++;
        await new Promise(resolve => setTimeout(resolve, 100));
    }
    
    if (attempts >= 50) {
        console.error('Yoga WASM failed to initialize after 5 seconds');
        process.exit(1);
    }
    
    render(<App/>);
}

startApp().catch(error => {
    console.error('Failed to start app:', error);
    process.exit(1);
});