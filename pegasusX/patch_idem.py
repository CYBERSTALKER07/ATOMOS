import re

with open('apps/backend-go/idempotency/middleware.go', 'r') as f:
    content = f.read()

old_block = """			recorder := &responseRecorder{
				ResponseWriter: w,
				status:         http.StatusOK,
				body:           &bytes.Buffer{},
			}

			next.ServeHTTP(recorder, r)

			if recorder.status >= 200 && recorder.status < 300 {
				_ = store.Save(r.Context(), key, Record{
					BodyHash:   hashHex,
					StatusCode: recorder.status,
					Response:   recorder.body.Bytes(),
					StoredAt:   time.Now(),
				}, 24*time.Hour)
			} else {
				_ = store.Release(r.Context(), key)
			}"""

new_block = """			recorder := &responseRecorder{
				ResponseWriter: w,
				status:         http.StatusOK,
				body:           &bytes.Buffer{},
			}

			func() {
				defer func() {
					if rec := recover(); rec != nil {
						_ = store.Release(r.Context(), key)
						panic(rec)
					}
				}()
				next.ServeHTTP(recorder, r)
			}()

			if recorder.status >= 200 && recorder.status < 300 {
				_ = store.Save(r.Context(), key, Record{
					BodyHash:   hashHex,
					StatusCode: recorder.status,
					Response:   recorder.body.Bytes(),
					StoredAt:   time.Now(),
				}, 24*time.Hour)
			} else {
				_ = store.Release(r.Context(), key)
			}"""

if old_block in content:
    content = content.replace(old_block, new_block)
    with open('apps/backend-go/idempotency/middleware.go', 'w') as f:
        f.write(content)
    print("Patched middleware.go successfully.")
else:
    print("Could not find the target block to replace.")

