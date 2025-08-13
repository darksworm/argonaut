#!/usr/bin/env node
import { execSync, exec } from 'node:child_process';
import { copyFileSync, chmodSync, existsSync, mkdirSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { promisify } from 'node:util';

const execAsync = promisify(exec);
const __dirname = dirname(fileURLToPath(import.meta.url));
const rootDir = join(__dirname, '..');
const distDir = join(rootDir, 'dist');
const seaDir = join(distDir, 'sea');

// Ensure directories exist
if (!existsSync(seaDir)) {
    mkdirSync(seaDir, { recursive: true });
}

console.log('🚀 Building Single Executable Application...');

try {
    // Step 1: Build CommonJS bundle
    console.log('📦 Building CommonJS bundle...');
    execSync('npm run build:sea', { stdio: 'inherit', cwd: rootDir });
    
    // Step 2: Generate SEA preparation blob
    console.log('🔧 Generating SEA preparation blob...');
    execSync('npm run sea:prep', { stdio: 'inherit', cwd: rootDir });
    
    // Step 3: Get Node.js executable path
    const nodePath = process.execPath;
    console.log(`📍 Using Node.js from: ${nodePath}`);
    
    // Step 4: Copy Node.js executable for each platform
    const platforms = [
        { name: 'argonaut-macos', source: nodePath }
    ];
    
    for (const platform of platforms) {
        console.log(`🔨 Creating ${platform.name}...`);
        const targetPath = join(seaDir, platform.name);
        
        // Copy Node.js binary
        copyFileSync(platform.source, targetPath);
        
        // Remove signature on macOS (required for injection)
        if (process.platform === 'darwin') {
            try {
                execSync(`codesign --remove-signature "${targetPath}"`, { stdio: 'pipe' });
                console.log('  ✅ Removed code signature');
            } catch (error) {
                console.log('  ⚠️  Could not remove signature (might not be signed)');
            }
        }
        
        // Inject the SEA blob using postject
        const blobPath = join(distDir, 'sea-prep.blob');
        execSync(`npx postject "${targetPath}" NODE_SEA_BLOB "${blobPath}" \\
            --sentinel-fuse NODE_SEA_FUSE_fce680ab2cc467b6e072b8b5df1996b2 \\
            --macho-segment-name NODE_SEA`, 
            { stdio: 'inherit', cwd: rootDir });
        
        // Make executable
        chmodSync(targetPath, 0o755);
        
        console.log(`  ✅ Created ${platform.name}`);
    }
    
    console.log('🎉 Single executable applications built successfully!');
    console.log(`📁 Executables available in: ${seaDir}`);
    console.log('');
    console.log('Usage:');
    console.log(`  ./dist/sea/argonaut-macos`);
    
} catch (error) {
    console.error('❌ SEA build failed:', error.message);
    process.exit(1);
}