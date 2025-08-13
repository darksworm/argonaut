/**
 * Rollup plugin to patch top-level await issues for SEA compatibility
 * - Patches yoga-layout's WASM loading
 * - Patches Ink's devtools import
 */

export function patchYogaTopLevelAwait() {
  return {
    name: 'patch-yoga-toplevel-await',
    transform(code, id) {
      let patchedCode = code;
      let wasPatched = false;

      // Patch yoga-layout's top-level await
      if (id.includes('node_modules/yoga-layout/dist/src/index.js')) {
        console.log('🔧 Patching yoga-layout top-level await...');
        
        // Replace the problematic line
        patchedCode = patchedCode.replace(
          /const Yoga = wrapAssembly\(await loadYoga\(\)\);/g,
          `
// Patched for SEA compatibility - remove top-level await
let Yoga = null;
let yogaPromise = null;

function ensureYogaLoaded() {
  if (!yogaPromise) {
    yogaPromise = loadYoga().then(yoga => {
      Yoga = wrapAssembly(yoga);
      return Yoga;
    });
  }
  return yogaPromise;
}

// Initialize immediately but don't block module loading
ensureYogaLoaded().catch(console.error);

// Create a proxy that initializes on first access
const YogaProxy = new Proxy({}, {
  get(target, prop) {
    if (!Yoga) {
      throw new Error('Yoga WASM not yet loaded. This should not happen in normal usage.');
    }
    return Yoga[prop];
  }
});

// For modules that expect Yoga to be available immediately, 
// we'll need to handle this differently, but this gets us past the parsing stage
const YogaExport = YogaProxy;
          `
        );
        
        // Also need to update the export
        patchedCode = patchedCode.replace(
          /export default Yoga;/g,
          'export default YogaExport;'
        );
        
        if (patchedCode !== code) {
          console.log('✅ Successfully patched yoga-layout top-level await');
          wasPatched = true;
        }
      }
      
      // Patch Ink's devtools top-level await
      if (id.includes('node_modules/ink/build/reconciler.js')) {
        console.log('🔧 Patching Ink devtools top-level await...');
        
        // Replace the await import with a conditional import that doesn't block
        patchedCode = patchedCode.replace(
          /if \(process\.env\['DEV'\] === 'true'\) \{\s*try \{\s*await import\('\.\/devtools\.js'\);/g,
          `if (process.env['DEV'] === 'true') {
    try {
        // Patched for SEA compatibility - make devtools import non-blocking
        import('./devtools.js').catch(() => {
            // Ignore devtools loading errors in SEA builds
        });`
        );
        
        if (patchedCode !== code) {
          console.log('✅ Successfully patched Ink devtools top-level await');
          wasPatched = true;
        }
      }
      
      return wasPatched ? {
        code: patchedCode,
        map: null
      } : null;
    }
  };
}