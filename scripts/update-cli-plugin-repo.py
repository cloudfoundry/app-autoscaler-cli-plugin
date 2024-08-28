#!/usr/bin/env python3

# This scripts takes the version number, platform and filename of a plugin file
# and updates the repo-index.yml in the cli-plugin-repo directory


import os
import hashlib
import yaml
import sys
import datetime
import argparse

def sha1sum(filename):
    with open(filename, 'rb', buffering=0) as f:
        return hashlib.file_digest(f, 'sha1').hexdigest()


parser = argparse.ArgumentParser(description='Update the cli-plugin-repo')
parser.add_argument('version', help='The version number')
parser.add_argument('platform', help='The platform (linux64, osx, win64)')
parser.add_argument('filename', help='The plugin file name')
args = parser.parse_args()

with open('cli-plugin-repo/repo-index.yml', 'r') as f:
    repo_index = yaml.safe_load(f)
    if repo_index is None:
        raise ValueError('Could not load repo-index.yml')

    for entry in repo_index['plugins']:
        if entry['name'] == 'app-autoscaler-plugin':
            entry['version'] = args.version
            entry['updated'] = datetime.datetime.now(datetime.UTC).strftime('%Y-%m-%dT%H:%M:%SZ')
            for binary in entry['binaries']:
                if binary['platform'] == args.platform:
                    binary['url'] = f"https://github.com/cloudfoundry/app-autoscaler-cli-plugin/releases/download/v{args.version}/{args.filename}"
                    binary['checksum'] = sha1sum(f'build/{args.filename}')

with open('cli-plugin-repo/repo-index.yml', 'w') as f:
    yaml.dump(repo_index, f)
