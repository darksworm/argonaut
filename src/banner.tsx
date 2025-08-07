import React from 'react';
import {Box, Text} from 'ink';
import chalk from 'chalk';

interface ArgoNautBannerProps {
    server?: string | null;
    clusterScope?: string;
    namespaceScope?: string;
    projectScope?: string;
    termCols?: number;
    termRows?: number;
    apiVersion?: string;
    argonautVersion?: string;
}

const ArgoNautBanner: React.FC<ArgoNautBannerProps> = ({
                                                           server,
                                                           clusterScope,
                                                           namespaceScope,
                                                           projectScope,
                                                           termCols = 80,
                                                           termRows = 24,
                                                           apiVersion,
                                                           argonautVersion,
                                                       }) => {
    const isNarrow = termCols <= 100;   // stack vertically

    const Context = ({paddingBottom, paddingTop}) => (
        <Box
            flexDirection="column"
            paddingRight={2}
            paddingBottom={paddingBottom || 0}
            paddingTop={paddingTop || 0}
            alignSelf={isNarrow ? undefined : 'flex-end'}
        >
            {server && (
                <>
                    <Text><Text bold>Context:</Text> <Text color="cyan">{server || '—'}</Text></Text>
                    {clusterScope && clusterScope !== '—' && <Text><Text bold>Cluster:</Text> {clusterScope}</Text>}
                    {namespaceScope && namespaceScope !== '—' && <Text><Text bold>Namespace:</Text> {namespaceScope}</Text>}
                    {projectScope && projectScope !== '—' && <Text><Text bold>Project:</Text> {projectScope}</Text>}
                    {!isNarrow && apiVersion && <Text><Text bold>ArgoCD:</Text> <Text color="green">{apiVersion}</Text></Text>}
                </>
            )}
        </Box>
    );

    // Text-only for tiny terminals
    if (isNarrow) {
        return (
            <Box flexDirection="column" paddingTop={1}>
                <Box>
                    <Text backgroundColor="cyan" color="white" bold>{' '}Argonaut{' '} {argonautVersion && `${argonautVersion}`}</Text>
                </Box>
                <Context paddingBottom={1} paddingTop={1}></Context>
            </Box>
        );
    }

    // ASCII logo (wrap=truncate so it never line-wraps on narrow widths)
    const Logo = ({align, paddingBottom}:{align:'center'|'flex-end', paddingBottom: number}) => (
        <Box flexDirection="column"
             paddingBottom={paddingBottom || 0}
             alignItems={align}>
            <Text>
                {chalk.cyan('    _____')}&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;{chalk.whiteBright(' __   ')}
            </Text>
            <Text>
                {chalk.cyan('  /  _  \\_______  ____   ____') + chalk.whiteBright('   ____ _____   __ ___/  |_ ')}
            </Text>
            <Text>
                {chalk.cyan(' /  /_\\  \\_  __ \\/ ___\\ /  _ \\ ') + chalk.whiteBright('/    \\\\__  \\ |  |  \\   __\\')}
            </Text>
            <Text>
                {chalk.cyan('/    |    \\  | \\/ /_/  >  <_> )  ')+chalk.whiteBright(' |  \\/ __ \\|  |  /|  |  ')}
            </Text>
            <Text>
                {chalk.cyan('\\____|__  /__|  \\___  / \\____/')+chalk.whiteBright('|___|  (____  /____/ |__|  ')}
            </Text>
            <Text>
                {chalk.cyan('        \\/     /_____/             ')+chalk.whiteBright(`\\/     \\/     ${chalk.dim(argonautVersion)}  `)}
            </Text>
        </Box>
    );

    // Wide: side-by-side, bottom-aligned
    return (
        <Box justifyContent="space-between" alignItems="flex-end">
            <Context />
            <Logo paddingBottom={0} align="flex-end" />
        </Box>
    );
};

export default ArgoNautBanner;
