---
name: test-with-spanner
description: Run unit tests that require the Spanner emulator. Use this skill when the user wants to run tests in packages like satellite/metabase, satellite/metainfo, or any other tests that interact with Spanner. Automatically handles checking for and configuring the Spanner emulator environment.
allowed-tools: Bash(go *)
---

# Run Unit Tests with Spanner

You are helping run unit tests that require the Spanner emulator.

## Instructions

The Storj test framework automatically manages the Spanner emulator lifecycle using the `run:` prefix in the `STORJ_TEST_SPANNER` environment variable.
To run tests automatically `spanner_emulator` binary needs to be on PATH. Spanner can be also set for tests using `-spanner-test-db` flag.

### Running Tests

1. **If test name is provided in arguments**:
   - Find the package containing the test using Grep
   - Run the test with Spanner

2. **If only package path is provided**:
   - Run all tests in that package with Spanner

3. **Determine the Spanner connection method**:
   - First, check if `STORJ_TEST_SPANNER` is already set in the environment. If it is, do NOT pass `-spanner-test-db` — the test framework will pick it up automatically.
   - Only if `STORJ_TEST_SPANNER` is not set, pass `-spanner-test-db 'run:spanner_emulator'` to auto-manage the emulator.

4. **Command format**:
```bash
# When STORJ_TEST_SPANNER is already set in the environment:
go test -v ./package/path -run TestName

# When STORJ_TEST_SPANNER is NOT set:
go test -v ./package/path -run TestName -spanner-test-db 'run:spanner_emulator'
```

   The `run:` prefix tells the test framework to:
   - Automatically start the Spanner emulator before tests
   - Configure the connection for each test
   - Clean up and stop the emulator after tests complete

5. **Report test results**:
   - Show whether tests passed or failed
   - List all subtests that ran
   - If tests failed, offer to help investigate the failures

## Common test paths

Some common test paths in the Storj codebase:
- `./satellite/metabase` - Metabase tests
- `./satellite/metainfo` - Metainfo API tests
- `./satellite/satellitedb` - Database tests

## Example Usage

```bash
# Run a specific test
go test -v ./satellite/metainfo -run TestEndpoint_Object_No_StorageNodes -spanner-test-db 'run:spanner_emulator'

# Run all tests in a package
go test -v ./satellite/metabase -spanner-test-db 'run:spanner_emulator'

# Run tests with timeout
go test -v -timeout 10m ./satellite/metabase -run TestLoop -spanner-test-db 'run:spanner_emulator'
```

## Notes

- Each test gets its own emulator instance that's automatically managed
- No manual cleanup is required - the framework handles emulator lifecycle
- The `run:` prefix is the recommended approach used in Storj's CI/CD (see Jenkinsfile.verify and Jenkinsfile.public)


# Universal Agent & Engineering Guidelines
When developing, designing, or planning, always ensure to account for:
- Gaps, edge cases, and comprehensive feature validation.
- Best practices and optimized integration for Kafka, Redis, Backend, Optimizers, AI, and UI.
- Real-time concepts including WebSockets, webhooks, and their native app equivalents.
- Thorough business logic for features, understanding how the role, app, and ecosystem work together, and engagements with other roles and features.
- Best practices for backend, frontend, and infrastructure libraries/packages. Always prefer existing, high-quality open-source libraries and packages that best suit our features before creating our own.
- Optimal UI infrastructure and UX patterns (e.g., optimal screen positioning for drivers during an active route), applying the same high standards to backend and cloud architecture.
- ALWAYS search the web to find open-source code, libraries, packages, math, algorithms, approaches, and best practices for anything we are doing. If none exist, then create our own.
- Always search the web to get the correct logic, and incorporate edge cases, business logic for features, operations (ops), workflow, data consistency, finance, and AI into everything we do.
