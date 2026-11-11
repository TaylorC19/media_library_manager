#!/usr/bin/env python3
"""Build public Docker download bundles without registry credentials."""
import argparse
import hashlib
import pathlib
import re
import shutil
import subprocess
import tarfile
import tempfile

ROOT = pathlib.Path(__file__).resolve().parent.parent

def release_compose(source, version):
    # Source files retain build support for maintainers. Download bundles contain
    # only version-pinned, locally loaded app/setup images and no build sections.
    text = re.sub(r'^    build:\n(?:      .*\n)+', '', source, flags=re.MULTILINE)
    text = text.replace('media-library-manager:local', 'media-library-manager:' + version)
    text = text.replace('media-library-manager-setup:local', 'media-library-manager-setup:' + version)
    text = re.sub(r'^(    image: media-library-manager[^\n]+)$', r'\1\n    pull_policy: never', text, flags=re.MULTILINE)
    if re.search(r'^\s*build:', text, re.MULTILINE):
        raise RuntimeError('release Compose unexpectedly contains a build instruction')
    return text

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('version')
    args = parser.parse_args()
    version = args.version
    if not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9._-]*', version):
        parser.error('invalid version')
    revision = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()
    output = ROOT / 'dist' / 'release' / version
    output.mkdir(parents=True, exist_ok=True)
    subprocess.run([str(ROOT / 'scripts/package-setup.sh'), version], cwd=ROOT, check=True)
    for archive in (ROOT / 'dist/setup').glob('mlm-setup_' + version + '_*.tar.gz'):
        shutil.copyfile(archive, output / archive.name)
    for arch in ('amd64', 'arm64'):
        images = []
        for target, name in [('app', 'media-library-manager'), ('setup', 'media-library-manager-setup')]:
            tag = name + ':' + version
            subprocess.run(['docker', 'buildx', 'build', '--platform', 'linux/' + arch,
                            '--target', target, '--build-arg', 'VERSION=' + version,
                            '--build-arg', 'REVISION=' + revision, '--load', '-t', tag, '.'],
                           cwd=ROOT, check=True)
            images.append(tag)
        with tempfile.TemporaryDirectory(prefix='mlm-release-bundle-') as scratch:
            stage = pathlib.Path(scratch) / ('media-library-manager_' + version)
            stage.mkdir()
            subprocess.run(['docker', 'image', 'save', '-o', str(stage / 'images.tar'), *images], check=True)
            (stage / 'compose.yml').write_text(release_compose((ROOT / 'docker-compose.prod.yml').read_text(), version))
            (stage / 'compose.external.yml').write_text(release_compose((ROOT / 'docker-compose.external.yml').read_text(), version))
            (stage / 'scripts').mkdir()
            shutil.copyfile(ROOT / 'scripts/mongo-entrypoint.sh', stage / 'scripts/mongo-entrypoint.sh')
            shutil.copyfile(ROOT / '.env.prod.example', stage / '.env.prod.example')
            shutil.copyfile(ROOT / '.env.setup.example', stage / '.env.setup.example')
            guide = (ROOT / 'docs/release-install.md').read_text().replace('@VERSION@', version)
            (stage / 'README.md').write_text(guide)
            with tarfile.open(output / ('media-library-manager_' + version + '_linux_' + arch + '.tar.gz'), 'w:gz') as archive:
                archive.add(stage, arcname=stage.name)
    with (output / 'SHA256SUMS').open('w') as manifest:
        for path in sorted(output.glob('*.tar.gz')):
            with path.open('rb') as data:
                digest = hashlib.file_digest(data, 'sha256').hexdigest()
            manifest.write(digest + '  ' + path.name + '\n')
    print('Release assets prepared in', output)

if __name__ == '__main__':
    main()
