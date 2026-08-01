from dataclasses import dataclass
from enum import Enum
from typing import Any

class LabelType(Enum):
    Bool = 0
    Int = 1
    String = 2

@dataclass(frozen=True)
class Label:
    name: str
    label_type: LabelType

@dataclass
class Field:
    label: Label
    value: Any

def labels_match(ll1: list[Label], ll2: list[Label]) -> bool:
    if len(ll1) != len(ll2):
        return False
    count1 = {}
    count2 = {}
    for l in ll1:
        key = (l.name, l.label_type)
        count1[key] = count1.get(key, 0) + 1
    for l in ll2:
        key = (l.name, l.label_type)
        count2[key] = count2.get(key, 0) + 1
    return count1 == count2

def types_match(label: Label, field: Field) -> bool:
    if isinstance(field.value, bool):
        return label.label_type == LabelType.Bool
    elif isinstance(field.value, int):
        return label.label_type == LabelType.Int
    elif isinstance(field.value, str):
        return label.label_type == LabelType.String
    else:
        return False

def fields_match(fl1: list[Field], fl2: list[Field]) -> bool:
    if len(fl1) != len(fl2):
        return False
    count1 = {}
    count2 = {}
    for f in fl1:
        key = (f.label.name, f.label.label_type, type(f.value))
        count1[key] = count1.get(key, 0) + 1
    for f in fl2:
        key = (f.label.name, f.label.label_type, type(f.value))
        count2[key] = count2.get(key, 0) + 1
    return count1 == count2
