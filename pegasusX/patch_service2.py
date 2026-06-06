import re

with open("apps/backend-go/warehouse/service.go", "r") as f:
    content = f.read()

# ACQUIRED
content = re.sub(
    r"if err := s\.repo\.Apply\(r\.Context\(\), func\(\) error \{\n\t\t\ts\.mu\.Lock\(\)\n\t\t\tdefer s\.mu\.Unlock\(\)\n\t\t\ts\.locks\[lock\.LockID\] = lock\n\t\t\treturn nil\n\t\t\}, func\(txn outbox\.TxnBuffer\) error \{\n\t\t\treturn outbox\.EmitJSON\(r\.Context\(\), txn, events\.AggregateWarehouse, lock\.LockID, events\.TopicMain, eventPayload\)\n\t\t\}\); err != nil \{",
    r"""if err := s.repo.UpsertLock(r.Context(), warehouseID, lock, func(txn outbox.TxnBuffer) error {
			return outbox.EmitJSON(r.Context(), txn, events.AggregateWarehouse, lock.LockID, events.TopicMain, eventPayload)
		}); err != nil {""",
    content
)

# DELETE
content = re.sub(
    r"if err := s\.repo\.Apply\(r\.Context\(\), func\(\) error \{\n\t\t\ts\.mu\.Lock\(\)\n\t\t\tdefer s\.mu\.Unlock\(\)\n\t\t\treleasedLock, exists := s\.locks\[lockID\]\n\t\t\tif !exists \{\n\t\t\t\treturn errDispatchLockNotFound\n\t\t\t\}\n\t\t\treleased = releasedLock\n\t\t\tdelete\(s\.locks, lockID\)\n\t\t\treturn nil\n\t\t\}, func\(txn outbox\.TxnBuffer\) error \{",
    r"""// fetch to check exists
		lockMap, err := s.repo.GetLocks(r.Context(), warehouseID)
		releasedLock, exists := lockMap[lockID]
		if !exists || err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "lock_not_found"})
			return
		}
		released = releasedLock
		if err := s.repo.DeleteLock(r.Context(), warehouseID, lockID, func(txn outbox.TxnBuffer) error {""",
    content
)

with open("apps/backend-go/warehouse/service.go", "w") as f:
    f.write(content)

