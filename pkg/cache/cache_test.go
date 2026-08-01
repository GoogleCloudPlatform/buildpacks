import hashlib
from typing import Any, List, Callable

class Context:
    def __init__(self, buildpack_id: str, buildpack_version: str):
        self.buildpack_id = buildpack_id
        self.buildpack_version = buildpack_version
    
    def set_metadata(self, layer: 'Layer', key: str, value: str) -> None:
        layer.metadata[key] = value
    
    def get_metadata(self, layer: 'Layer', key: str) -> Any:
        return layer.metadata.get(key)
    
    def debugf(self, format_str: str, *args) -> None:
        print(format_str % args)
    
    def cache_hit(self, layer_name: str) -> None:
        self.debugf("Cache hit for layer: %s", layer_name)
    
    def cache_miss(self, layer_name: str) -> None:
        self.debugf("Cache miss for layer: %s", layer_name)

class Layer:
    def __init__(self, name: str = None):
        self.metadata = {}
        self._name = name
    
    @property
    def name(self) -> str:
        return self._name if self._name is not None else ''

def WithStrings(*strings: str) -> Callable[[], List[str]]:
    def option() -> List[str]:
        return list(strings)
    return option

def WithFiles(*files: str) -> Callable[[], List[str]]:
    def option() -> List[str]:
        contents = []
        for f in files:
            with open(f, 'r') as fp:
                contents.append(fp.read())
        return contents
    return option

def hash(ctx: Context, *opts: Callable[[], List[str]]) -> str:
    h = hashlib.sha256()
    h.update(ctx.buildpack_id.encode('utf-8'))
    h.update(ctx.buildpack_version.encode('utf-8'))
    
    for opt in opts:
        strings = opt()
        for s in strings:
            h.update(s.encode('utf-8'))
    
    return h.hexdigest()

def Add(ctx: Context, layer: Layer, key: str, value: str) -> None:
    ctx.set_metadata(layer, key, value)

def HashAndCheck(
    ctx: Context,
    layer: Layer,
    key: str,
    *opts: Callable[[], List[str]]
) -> (str, bool):
    current_hash = hash(ctx, *opts)
    prev_hash = ctx.get_metadata(layer, key)
    
    ctx.debugf("Current dependency hash: %r", current_hash)
    ctx.debugf("  Cache dependency hash: %r", prev_hash)
    
    if not prev_hash:
        ctx.debugf("No cache metadata found from a previous build for key: %r, skipping cache.", key)
    
    cached = current_hash == prev_hash
    if cached:
        ctx.cache_hit(layer.name)
    else:
        ctx.cache_miss(layer.name)
    
    return (current_hash, cached)
