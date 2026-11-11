import importlib.util
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parent.parent
spec = importlib.util.spec_from_file_location('packager', ROOT / 'scripts/package-docker-release.py')
packager = importlib.util.module_from_spec(spec)
spec.loader.exec_module(packager)

class ReleaseComposeTests(unittest.TestCase):
    def test_download_bundles_cannot_build_or_pull_unpublished_app_images(self):
        for filename in ('docker-compose.prod.yml', 'docker-compose.external.yml'):
            with self.subTest(filename=filename):
                value = packager.release_compose((ROOT / filename).read_text(), '0.1.0-alpha.1')
                self.assertNotIn('build:', value)
                self.assertNotIn(':local', value)
                self.assertEqual(value.count('pull_policy: never'), 2)
                self.assertIn('media-library-manager:0.1.0-alpha.1', value)
                self.assertIn('media-library-manager-setup:0.1.0-alpha.1', value)
                self.assertIn('service_completed_successfully', value)

if __name__ == '__main__':
    unittest.main()
