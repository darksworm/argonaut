# Single Executable Application (SEA) Support

This branch implements experimental support for Node.js Single Executable Applications (SEA) for Argonaut.

## What's Implemented

### ✅ Build Infrastructure
- **Dual Build System**: Both ESM and CommonJS builds work independently
- **SEA Configuration**: `sea-config.json` for Node.js SEA preparation
- **Build Scripts**: Complete pipeline from source to executable
- **Dependencies**: All required tools (`postject`, bundlers, etc.)

### ✅ File Structure
```
├── rollup.config.js         # Original ESM build (unchanged)
├── rollup.config.sea.js     # New CommonJS build for SEA
├── tsconfig.json            # Original TypeScript config
├── tsconfig.sea.json        # SEA-specific TypeScript config
├── sea-config.json          # SEA preparation configuration
└── scripts/build-sea.js     # Executable creation script
```

### ✅ NPM Scripts
```bash
npm run build          # Original ESM build (still works)
npm run build:sea      # Build CommonJS bundle for SEA
npm run sea:prep       # Generate SEA preparation blob
npm run sea:build      # Build + prepare (combined)
npm run sea:executable # Create actual executable files
```

## Current Limitation ⚠️

The SEA build currently **fails** due to a fundamental compatibility issue:

**Problem**: Some dependencies (particularly `yoga-layout` used by Ink) contain **top-level await**, which is incompatible with CommonJS format required by SEA.

**Error**: `Module format "cjs" does not support top-level await`

## Potential Solutions

### Option 1: Wait for Dependency Updates
- Wait for Ink/React ecosystem to become SEA-compatible
- Monitor Node.js SEA evolution for ESM support

### Option 2: Dependency Replacement
- Replace Ink with a SEA-compatible TUI library
- Major refactoring required

### Option 3: Alternative Distribution
- Use `pkg` or similar bundlers instead of native SEA
- Different trade-offs but might work today

### Option 4: Hybrid Approach
- Distribute regular Node.js app alongside SEA attempts
- SEA as experimental/future option

## Testing the Current State

The infrastructure is ready to go. Once the dependency compatibility issues are resolved:

```bash
# This should work in the future:
npm run sea:executable
./dist/sea/argonaut-macos
```

## Node.js SEA Status

As of 2024, Node.js SEA is still experimental (stability 1.1) and has limitations:
- CommonJS requirement conflicts with modern ESM ecosystem
- Limited cross-platform support
- Native module handling complexity

## Recommendation

Keep this branch as **experimental infrastructure** ready for when:
1. Node.js SEA gains ESM support, OR
2. Dependencies become SEA-compatible, OR  
3. Alternative solutions mature

The regular ESM build continues to work perfectly and should remain the primary distribution method.