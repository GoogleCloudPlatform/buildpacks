# Complete refactored code here
import os
from gcpbuildpack import opt_in_always, env

def detect_fn(ctx):
    return opt_in_always()

def build_fn(ctx):
    for key, value in os.environ.items():
        if not key.startswith(env.LABEL_PREFIX):
            continue
        label_key = key[len(env.LABEL_PREFIX):]
        ctx.AddLabel(label_key, value)
