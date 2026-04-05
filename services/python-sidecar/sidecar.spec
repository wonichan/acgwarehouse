# -*- mode: python ; coding: utf-8 -*-

from pathlib import Path

from PyInstaller.utils.hooks import collect_submodules


project_root = Path.cwd()
sidecar_root = project_root / 'services' / 'python-sidecar'
hiddenimports = [
    'fastapi',
    'uvicorn',
    'PIL',
    'imagehash',
    *collect_submodules('routers'),
    *collect_submodules('compute'),
    *collect_submodules('models'),
]

block_cipher = None


a = Analysis(
    [str(sidecar_root / 'main.py')],
    pathex=[str(sidecar_root)],
    binaries=[],
    datas=[],
    hiddenimports=hiddenimports,
    hookspath=[],
    hooksconfig={},
    runtime_hooks=[],
    excludes=[],
    win_no_prefer_redirects=False,
    win_private_assemblies=False,
    noarchive=False,
)
pyz = PYZ(a.pure, a.zipped_data, cipher=block_cipher)
exe = EXE(
    pyz,
    a.scripts,
    [],
    exclude_binaries=True,
    name='acgwarehouse-sidecar',
    debug=False,
    bootloader_ignore_signals=False,
    strip=False,
    upx=True,
    console=False,
)
coll = COLLECT(
    exe,
    a.binaries,
    a.zipfiles,
    a.datas,
    strip=False,
    upx=True,
    upx_exclude=[],
    name='python-sidecar',
)
