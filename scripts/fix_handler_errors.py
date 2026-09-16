"""Bulk-replace string-concat http.Error pattern in handlers with json.WriteError."""
import os
ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
FILES = [
    "internal/agecategories/handler.go",
    "internal/distances/handler.go",
    "internal/events/handler.go",
    "internal/organizers/handler.go",
    "internal/runnerkits/handler.go",
]
PATTERN = 'http.Error(w, `{"error":"`+err.Error()+`"}`, '
STATUSES = [
    "http.StatusBadRequest",
    "http.StatusInternalServerError",
    "http.StatusNotFound",
    "http.StatusUnauthorized",
    "http.StatusForbidden",
]
for rel in FILES:
    fp = os.path.join(ROOT, rel)
    with open(fp, "r", encoding="utf-8") as f:
        s = f.read()
    before = s.count(PATTERN)
    for st in STATUSES:
        s = s.replace(PATTERN + st + ")",
                      "json.WriteError(w, " + st + ", err.Error())")
    after = s.count(PATTERN)
    with open(fp, "w", encoding="utf-8") as f:
        f.write(s)
    print(rel, "before:", before, "after:", after)
