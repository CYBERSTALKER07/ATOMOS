import re

with open("apps/backend-go/warehouse/service.go", "r") as f:
    content = f.read()

# remove fields
content = re.sub(r"\tinventory\s+map\[string\]InventoryRow\n", "", content)
content = re.sub(r"\tlocks\s+map\[string\]DispatchLock\n", "", content)

# remove init
content = re.sub(r"\t\tinventory:\s+make\(map\[string\]InventoryRow\),\n", "", content)
content = re.sub(r"\t\tlocks:\s+make\(map\[string\]DispatchLock\),\n", "", content)

# replace s.locks usages in HandleLocks
content = re.sub(
    r"locks := make\(\[\]map\[string\]any, 0, len\(s\.locks\)\)\n\tfor _, v := range s\.locks \{",
    r"""lockMap, err := s.repo.GetLocks(r.Context(), "wh-1")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_fetch_locks"})
		return
	}
	locks := make([]map[string]any, 0, len(lockMap))
	for _, v := range lockMap {""",
    content
)

content = re.sub(
    r"s\.mu\.Lock\(\)\n\t\t\ts\.locks\[lock\.LockID\] = lock\n\t\t\ts\.mu\.Unlock\(\)",
    r"""err := s.repo.UpsertLock(r.Context(), "wh-1", lock, nil)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_upsert_lock"})
				return
			}""",
    content
)

content = re.sub(
    r"s\.mu\.Lock\(\)\n\t\t\treleasedLock, exists := s\.locks\[lockID\]\n\t\t\tif !exists \{\n\t\t\t\ts\.mu\.Unlock\(\)[\s\S]+?s\.mu\.Unlock\(\)",
    r"""// fetch to check exists
			lockMap, err := s.repo.GetLocks(r.Context(), "wh-1")
			releasedLock, exists := lockMap[lockID]
			if !exists || err != nil {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "lock_not_found"})
				return
			}
			err = s.repo.DeleteLock(r.Context(), "wh-1", lockID, nil)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_delete_lock"})
				return
			}""",
    content
)


with open("apps/backend-go/warehouse/service.go", "w") as f:
    f.write(content)

