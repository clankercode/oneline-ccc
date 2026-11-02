#!/usr/bin/env python3
"""Prepare focused real local fixtures, then run bounded evidence/process faults."""
import json
import pathlib
import subprocess
import sys
import tempfile

from capture import CACHE, HERE, ROOT
from resource_budget import budget_environment, run_supervised, verification_slot

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
        ('more-first','more','baseline','baseline','40x12'),
        ('more-last','more','baseline','last-item','40x12'),
        ('pending','dashboard','parity','pending-focus','40x12'),
        ('filter-match','dashboard','parity','filter-match','40x12'),
        ('filter-unicode','dashboard','unicode','filter-unicode','40x12'),
    ]:
        path = pathlib.Path(temp) / name
        subprocess.run([sys.executable,str(HERE/'capture.py'),'capture','--scenario',scenario,'--state',state,
                        '--variant',variant,'--sizes',size,'--output',str(path)],cwd=ROOT,check=True)
        scenes[name] = str(path/'manifest.json')
    env = dict(budget_environment(scratch), A2_TEST_MANIFEST=str(output/'manifest.json'),
               A2_SCENE_MANIFESTS=json.dumps(scenes), A2B_BASELINE_MANIFEST=str(output/'manifest.json'),
               A2B_UNICODE_BASELINE_MANIFEST=scenes['unicode'], A2B_SCENE_MANIFESTS=json.dumps({key:scenes[key] for key in ['more-first','more-last','pending','filter-match','filter-unicode']}), PYTHONDONTWRITEBYTECODE='1')
    with verification_slot(ROOT):
        run_supervised([sys.executable,'-m','unittest','discover','-s',str(HERE),'-p','test_*.py','-v'], cwd=ROOT,env=env,check=True)
        run_supervised(['node','--test','--test-concurrency='+env['GOMAXPROCS'],str(HERE/'render.test.mjs'),str(HERE/'semantic.test.mjs'),str(HERE/'legacy.test.mjs'),str(HERE/'a2b.test.mjs'),str(HERE/'prepared.test.mjs')], cwd=ROOT,env=env,check=True)
