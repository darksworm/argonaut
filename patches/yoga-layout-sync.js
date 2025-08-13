/**
 * Synchronous wrapper for yoga-layout to avoid top-level await
 * This patches the yoga-layout index.js to work with CommonJS/SEA builds
 */

// @ts-ignore untyped from Emscripten
import loadYoga from '../binaries/yoga-wasm-base64-esm.js';
import wrapAssembly from "./wrapAssembly.js";

let yogaInstance = null;
let yogaPromise = null;

// Initialize yoga asynchronously but store the result
function initYoga() {
    if (!yogaPromise) {
        yogaPromise = loadYoga().then(yoga => {
            yogaInstance = wrapAssembly(yoga);
            return yogaInstance;
        });
    }
    return yogaPromise;
}

// Synchronous getter that throws if not initialized
function getYoga() {
    if (!yogaInstance) {
        throw new Error('Yoga WASM not yet initialized. Call initYoga() first and wait for it to resolve.');
    }
    return yogaInstance;
}

// For compatibility, export both sync and async interfaces
const Yoga = {
    ...getYoga, // This will fail if called too early, but SEA doesn't mind the structure
    init: initYoga,
    getInstance: getYoga
};

// Try to initialize immediately in a way that doesn't block module loading
initYoga().catch(() => {
    // Ignore initialization errors during module import
});

export default Yoga;
export * from "./generated/YGEnums.js";