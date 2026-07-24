import json
from typing import Dict, Any
from .counter import Counter
from .floatdp import FloatDP
from threading import Lock

class BuilderMetrics:
    _instance = None
    _lock = Lock()
    
    def __init__(self):
        self.counters: Dict[str, Counter] = {}
        self.float_dps: Dict[str, FloatDP] = {}

    @classmethod
    def get_instance(cls) -> 'BuilderMetrics':
        with cls._lock:
            if not cls._instance:
                cls._instance = BuilderMetrics()
        return cls._instance

    def reset(self):
        with self._lock:
            self.counters.clear()
            self.float_dps.clear()

    def get_counter(self, metric_id: str) -> Counter:
        if metric_id not in self.counters:
            self.counters[metric_id] = Counter()
        return self.counters[metric_id]

    def for_each_counter(self, callback):
        for metric_id, counter in self.counters.items():
            callback(metric_id, counter)

    def get_float_dp(self, metric_id: str) -> FloatDP:
        if metric_id not in self.float_dps:
            self.float_dps[metric_id] = FloatDP()
        return self.float_dps[metric_id]

    def for_each_float_dp(self, callback):
        for metric_id, float_dp in self.float_dps.items():
            callback(metric_id, float_dp)

    def to_dict(self) -> Dict[str, Any]:
        return {
            'counters': {k: v.value for k, v in self.counters.items()},
            'float_dps': {k: v.value for k, v in self.float_dps.items()}
        }

    def from_dict(self, data: Dict[str, Any]):
        self.counters = {}
        self.float_dps = {}
        if 'counters' in data:
            self.counters = {k: Counter(v) for k, v in data['counters'].items()}
        if 'float_dps' in data:
            self.float_dps = {k: FloatDP(v) for k, v in data['float_dps'].items()}

    def __repr__(self):
        return f"BuilderMetrics(counters={len(self.counters)}, float_dps={len(self.float_dps)})"

# Global instance
BM_INSTANCE = BuilderMetrics()
