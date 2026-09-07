"""Hermetic seed-script regression tests: python3 -m unittest discover -s argocd/fixtures -p '*_test.py'."""
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

SCRIPT = Path(__file__).with_name('seed-sync-fixtures.sh').resolve()


class SeedFixturesTest(unittest.TestCase):
    def run_seed(self, port='9418', stuck=False, probe_fails=False):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            bin_dir = root / 'bin'
            bin_dir.mkdir()
            repo = root / 'argonaut-sync-fixtures-repo'
            repo.mkdir()
            sentinel = repo / 'sentinel'
            sentinel.touch()
            stubs = {
                'argocd': '#!/bin/bash\necho "$*" >> "$CALLS"\nif [[ "$1 $2" == "app get" ]]; then [[ "$STUCK" == 1 ]]; else exit 0; fi\n',
                'bash': '#!/bin/sh\n[ "$PROBE_FAILS" != 1 ]\n',
                'sleep': '#!/bin/sh\nexit 0\n',
                'kubectl': '#!/bin/sh\necho "kubectl $*" >> "$CALLS"\ncat >/dev/null\n',
            }
            for name, contents in stubs.items():
                path = bin_dir / name
                path.write_text(contents)
                path.chmod(0o755)
            calls = root / 'calls'
            env = dict(os.environ, PATH=f'{bin_dir}:' + os.environ['PATH'],
                       GIT_DAEMON_BASE_PATH=tmp, GIT_HOST='127.0.0.1',
                       GIT_DAEMON_PORT=port, STUCK=str(int(stuck)),
                       PROBE_FAILS=str(int(probe_fails)), CALLS=str(calls))
            result = subprocess.run(['/bin/bash', str(SCRIPT)], env=env, capture_output=True, text=True, timeout=10)
            return result, sentinel.exists(), calls.read_text() if calls.exists() else ''

    def test_invalid_ports_stop_before_reset(self):
        for port in ['0', '65536', '99999999999999999999', '9418; echo injected', 'abc']:
            with self.subTest(port=port):
                result, preserved, calls = self.run_seed(port=port)
                self.assertNotEqual(result.returncode, 0)
                self.assertTrue(preserved, result.stdout)
                self.assertNotIn('app delete', calls)

    def test_deletion_timeout_preserves_repo_and_stops_sync(self):
        result, preserved, calls = self.run_seed(stuck=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(preserved, result.stdout)
        self.assertIn('prune-demo', result.stderr)
        self.assertNotIn('app sync', calls)
        self.assertNotIn('kubectl apply', calls)

    def test_disappeared_apps_allow_reset_and_sync(self):
        result, preserved, calls = self.run_seed()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(preserved)
        self.assertIn('app sync prune-demo', calls)
        self.assertIn('app sync prune-confirm-demo', calls)

    def test_failed_probe_stops_before_reset(self):
        result, preserved, _ = self.run_seed(probe_fails=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(preserved)
        self.assertIn('No git daemon on :9418', result.stderr)
