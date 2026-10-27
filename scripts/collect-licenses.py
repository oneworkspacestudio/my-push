"""Preserve dependency licence texts from the pinned public Go modules."""
import json,os,subprocess
from pathlib import Path
root=Path(__file__).resolve().parents[1]
raw=subprocess.check_output([os.environ.get('GO','go'),'list','-m','-json','all'],cwd=root,text=True)
decoder=json.JSONDecoder();modules=[]
while raw.strip():
 raw=raw.lstrip();m,end=decoder.raw_decode(raw);raw=raw[end:];modules.append(m)
parts=['Third-party licence texts for the pinned Go dependency graph.\n']
for m in modules:
 if m.get('Main'):continue
 directory=Path(m.get('Dir',''))
 if not m.get('Dir'):continue
 selected=[]
 for p in directory.iterdir():
  if p.is_file() and p.name.upper().startswith(('LICENSE','LICENCE','COPYING','NOTICE')):selected.append(p)
 for p in sorted(selected):
  parts.append('\n===== '+m['Path']+' '+m.get('Version','')+' / '+p.name+' =====\n'+p.read_text(encoding='utf-8',errors='replace')+'\n')
goroot=Path(subprocess.check_output([os.environ.get('GO','go'),'env','GOROOT'],text=True).strip())
parts.append('\n===== Go standard library / LICENSE =====\n'+(goroot/'LICENSE').read_text(encoding='utf-8'))
(root/'THIRD-PARTY-LICENSES.txt').write_text(''.join(parts),encoding='utf-8')
print('Preserved licence texts from',len(modules)-1,'pinned modules and the Go standard library.')
