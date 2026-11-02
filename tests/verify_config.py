"""Strict TOML assertions shared by both PowerShell editions (Python 3.11+)."""
import json
import pathlib
import sys
import tomllib

config_path = pathlib.Path(sys.argv[1])
expected = json.loads(pathlib.Path(sys.argv[2]).read_text(encoding="utf-8-sig"))
raw = config_path.read_bytes()
config = tomllib.loads(raw.decode("utf-8-sig"))
for dotted_path, value in expected["values"].items():
    current = config
    for component in dotted_path.split("."):
        current = current[component]
    assert current == value, f"{dotted_path}: expected {value!r}, got {current!r}"
for dotted_path in expected.get("absent", []):
    components = dotted_path.split(".")
    current = config
    for component in components[:-1]:
        current = current.get(component, {})
    assert components[-1] not in current, f"Unexpected setting: {dotted_path}"
for forbidden in expected.get("forbidden", []):
    assert forbidden.encode() not in raw, "Credential found in configuration"
print("TOML verified")
