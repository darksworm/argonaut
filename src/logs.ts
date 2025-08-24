#!/usr/bin/env bun
import { tailLogs } from './services/log-tailer';

async function main() {
  const args = process.argv.slice(2);
  const sessionArg = args.find(arg => arg.startsWith('--session='));
  const session = sessionArg ? sessionArg.split('=')[1] : 'latest';
  
  console.log(`📋 Tailing logs for session: ${session === 'latest' ? 'latest' : session}`);
  console.log('Press \'q\' or Ctrl+C to stop tailing...\n');
  
  const result = await tailLogs({ session });
  
  if (result.isErr()) {
    console.error(`❌ Failed to tail logs: ${result.error.message}`);
    process.exit(1);
  }
  
  // Setup input handling for 'q' to quit
  process.stdin.setRawMode?.(true);
  process.stdin.resume();
  process.stdin.setEncoding('utf8');
  
  process.stdin.on('data', (key: string) => {
    if (key === 'q' || key === 'Q' || key === '\u0003') { // 'q', 'Q', or Ctrl+C
      console.log('\n👋 Stopping log tail...');
      process.exit(0);
    }
  });
  
  // Keep the process alive while tailing
  process.on('SIGINT', () => {
    console.log('\n👋 Stopping log tail...');
    process.exit(0);
  });
}

main().catch(console.error);