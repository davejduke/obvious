import tomllib
import sys

with open("engine/pyproject.toml", "rb") as f:
    data = tomllib.load(f)

assert "project" in data, "Missing [project] table"
assert data["project"]["name"] == "aiauditor-engine", f"Wrong name: {data['project']['name']}"

deps = data["project"]["dependencies"]
required_deps = ["pydantic", "fastapi", "uvicorn", "sqlalchemy", "asyncpg", "redis"]
for dep in required_deps:
    found = any(dep in d for d in deps)
    assert found, f"Missing dependency: {dep}"
    print(f"OK dep: {dep}")

print("pyproject.toml is valid")
