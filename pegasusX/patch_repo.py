import re

with open("apps/backend-go/warehouse/service_test.go", "r") as f:
    content = f.read()

content = re.sub(
    r"events          \[\]outbox\.Event",
    r"events          []outbox.Event\n\tlocks           map[string]DispatchLock",
    content
)

content = re.sub(
    r"func \(r \*warehouseRepoSpy\) GetLocks\(ctx context\.Context, warehouseID string\) \(map\[string\]DispatchLock, error\) \{\n\treturn nil, nil\n\}",
    r"""func (r *warehouseRepoSpy) GetLocks(ctx context.Context, warehouseID string) (map[string]DispatchLock, error) {
	if r.locks == nil {
		r.locks = make(map[string]DispatchLock)
	}
	return r.locks, nil
}""",
    content
)

content = re.sub(
    r"func \(r \*warehouseRepoSpy\) UpsertLock\(ctx context\.Context, warehouseID string, lock DispatchLock, emit func\(outbox\.TxnBuffer\) error\) error \{\n\tr\.applyCalls\+\+\n\tif emit \!\= nil \{\n\t\tbuf :\= \&warehouseTxnBufferSpy\{\}\n\t\tif err :\= emit\(buf\); err \!\= nil \{\n\t\t\treturn err\n\t\t\}\n\t\tr\.events \= append\(r\.events, buf\.events\.\.\.\)\n\t\}\n\treturn nil\n\}",
    r"""func (r *warehouseRepoSpy) UpsertLock(ctx context.Context, warehouseID string, lock DispatchLock, emit func(outbox.TxnBuffer) error) error {
	r.applyCalls++
	if r.locks == nil {
		r.locks = make(map[string]DispatchLock)
	}
	r.locks[lock.LockID] = lock
	if emit != nil {
		buf := &warehouseTxnBufferSpy{}
		if err := emit(buf); err != nil {
			return err
		}
		r.events = append(r.events, buf.events...)
	}
	return nil
}""",
    content
)

content = re.sub(
    r"func \(r \*warehouseRepoSpy\) DeleteLock\(ctx context\.Context, warehouseID, lockID string, emit func\(outbox\.TxnBuffer\) error\) error \{\n\tr\.applyCalls\+\+\n\tif emit \!\= nil \{\n\t\tbuf :\= \&warehouseTxnBufferSpy\{\}\n\t\tif err :\= emit\(buf\); err \!\= nil \{\n\t\t\treturn err\n\t\t\}\n\t\tr\.events \= append\(r\.events, buf\.events\.\.\.\)\n\t\}\n\treturn nil\n\}",
    r"""func (r *warehouseRepoSpy) DeleteLock(ctx context.Context, warehouseID, lockID string, emit func(outbox.TxnBuffer) error) error {
	r.applyCalls++
	if r.locks != nil {
		delete(r.locks, lockID)
	}
	if emit != nil {
		buf := &warehouseTxnBufferSpy{}
		if err := emit(buf); err != nil {
			return err
		}
		r.events = append(r.events, buf.events...)
	}
	return nil
}""",
    content
)

with open("apps/backend-go/warehouse/service_test.go", "w") as f:
    f.write(content)
