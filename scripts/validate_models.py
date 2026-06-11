import ast
import sys

with open("engine/shared/models.py") as f:
    src = f.read()

tree = ast.parse(src)
classes = [n.name for n in ast.walk(tree) if isinstance(n, ast.ClassDef)]
required = ["Persona", "EngagementStatus", "FindingSeverity", "NIS2Article",
            "ControlModel", "EvidenceModel", "QualityScore", "FindingModel"]
for r in required:
    assert r in classes, f"Missing class: {r}"
    print(f"OK class: {r}")

# Check all 7 personas are in Persona enum
personas = ["internal_auditor", "cae", "audit_committee", "auditee_ciso",
            "cosourced_provider", "ptwg_member", "beta_tester"]
for p in personas:
    assert p in src, f"Missing persona: {p}"
print("All 7 personas present")

# Check all 10 NIS2 articles
articles = ["21a", "21b", "21c", "21d", "21e", "21f", "21g", "21h", "21i", "21j"]
for a in articles:
    assert a in src, f"Missing NIS2 article: {a}"
print("All 10 NIS2 articles present")

print("engine/shared/models.py is valid")
