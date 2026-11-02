#!/usr/bin/env python3
"""Prepare focused real local fixtures, then run bounded evidence/process faults."""
import json
import os
import pathlib
import subprocess
import sys
import tempfile

from capture import CACHE, HERE, ROOT

scratch = CACHE / 'tmp'
scratch.mkdir(parents=True, exist_ok=True)
with tempfile.TemporaryDirectory(prefix='evidence-faults-', dir=scratch) as temp:
    output = pathlib.Path(temp) / 'capture'
    subprocess.run([sys.executable, str(HERE/'capture.py'), 'capture', '--scenario', 'dashboard', '--state', 'parity', '--output', str(output)], cwd=ROOT, check=True)
    scenes = {}
    for name,scenario,state,variant,size in [
        ('unicode','dashboard','unicode','baseline','120x40'),
        ('help-first','help','baseline','baseline','40x12'),
        ('help-last','help','baseline','last-page','40x12'),
        ('budget-invalid','edit-budget','baseline','invalid','100x30'),
        ('compact-invalid','edit-compaction','baseline','invalid','100x30'),
        ('endpoint-invalid','edit-endpoint','baseline','invalid','100x30'),
        ('filter-empty','dashboard','baseline','no-results','80x24'),
        ('palette-empty','palette','baseline','no-results','80x24'),
        ('review-last','review','baseline','last-page','40x12'),
        ('review-ack','review','baseline','acknowledged','40x12'),
    ]:
        path = pathlib.Path(temp) / name
        subprocess.run([sys.executable,str(HERE/'capture.py'),'capture','--scenario',scenario,'--state',state,
                        '--variant',variant,'--sizes',size,'--output',str(path)],cwd=ROOT,check=True)
        scenes[name] = str(path/'manifest.json')
    env = dict(os.environ, A2_TEST_MANIFEST=str(output/'manifest.json'),
               A2_SCENE_MANIFESTS=json.dumps(scenes), PYTHONDONTWRITEBYTECODE='1')
    subprocess.run([sys.executable,'-m','unittest','discover','-s',str(HERE),'-p','test_*.py','-v'], cwd=ROOT,env=env,check=True)
    subprocess.run(['node','--test',str(HERE/'render.test.mjs'),str(HERE/'semantic.test.mjs'),str(HERE/'legacy.test.mjs')], cwd=ROOT,env=env,check=True)
