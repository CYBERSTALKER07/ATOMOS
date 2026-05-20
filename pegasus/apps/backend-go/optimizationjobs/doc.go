// Package optimizationjobs owns the durable ledger for externally queued
// optimization work. It stores request snapshots, bounded retry metadata, and
// status transitions, but it does not execute solver calls or apply results.
package optimizationjobs
