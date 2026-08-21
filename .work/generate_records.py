from pathlib import Path
import json

ROOT = Path('/Users/hutu/Desktop/go_projects/work_projects/013-distributed-search-service')
DATE = '2026-08-21'
REPO = 'distributed-search-service'

FILES = {
    1: ['internal/repository/error_get.go', 'internal/repository/error_create.go', 'internal/repository/error_update.go', 'internal/repository/error_list.go'],
    2: ['internal/storage/context_replay.go', 'internal/storage/context_append.go', 'internal/storage/context_recovery.go', 'internal/storage/context_copy.go'],
    3: ['internal/index/snapshot_view.go', 'internal/index/document_cache.go', 'internal/index/segment_catalog.go', 'internal/index/generation_stats.go'],
    4: ['internal/analyzer/token_filter.go', 'internal/analyzer/synonym_expand.go', 'internal/analyzer/stopword_filter.go', 'internal/analyzer/pipeline_snapshot.go'],
    5: ['internal/shard/result_producer.go', 'internal/shard/error_collector.go', 'internal/shard/wait_coordinator.go', 'internal/shard/stream_consumer.go'],
    6: ['internal/maintenance/lease_batch.go', 'internal/maintenance/operation_cleanup.go', 'internal/maintenance/rollback_guard.go', 'internal/maintenance/close_result.go'],
    7: ['internal/ingest/retry_transition.go', 'internal/ingest/transition_table.go', 'internal/ingest/pending_view.go', 'internal/ingest/state_audit.go'],
    8: ['internal/query/options.go', 'internal/query/policy.go', 'internal/query/facets.go', 'internal/query/validation_chain.go'],
    9: ['internal/transport/httpapi/cancel_bridge.go', 'internal/transport/httpapi/deadline_bridge.go', 'internal/transport/httpapi/context_slot.go', 'internal/transport/httpapi/trace_context.go'],
    10: ['internal/config/open_error.go', 'internal/config/scan_error.go', 'internal/config/validation_error.go', 'internal/config/classify_error.go'],
}

SYMBOLS = {
    1: ['WrapCollectionMissing', 'WrapCollectionConflict', 'WrapCollectionVersion', 'WrapCollectionPage'],
    2: ['ReplayWithContext', 'AppendWithContext', 'RecoverWithContext', 'CopyWithContext'],
    3: ['SnapshotView.Copy', 'DocumentCache.Snapshot', 'SegmentCatalog.List', 'GenerationStats.Values'],
    4: ['FilterTokens', 'ExpandSynonyms', 'RemoveStopwords', 'SnapshotPipeline'],
    5: ['ProduceResults', 'CollectShardError', 'StartShardWork', 'ConsumeStream'],
    6: ['ProcessLeases', 'RunWithCleanup', 'RunWithRollback', 'ResolveCloseResult'],
    7: ['RetryTransition', 'CanMove', 'PendingStates', 'AuditFinalState'],
    8: ['Options.AddLabel', 'PolicyEnabled', 'Facets.Add', 'ValidationChain.Add'],
    9: ['BridgeCancellation', 'BridgeDeadline', 'BridgeContextSlot', 'RequestContextForLog'],
    10: ['WrapOpenError', 'WrapScanError', 'WrapValidationError', 'ClassifyConfigError'],
}


def roots(n):
    name = f'{REPO}__{n:03d}'
    return ROOT / DATE / name / 'env', ROOT / '_gold' / name


def put_pair(n, rel, buggy, fixed):
    env, gold = roots(n)
    for base, text in ((env, buggy), (gold, fixed)):
        path = base / rel
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text.lstrip(), encoding='utf-8')


def put_same(n, rel, text):
    put_pair(n, rel, text, text)


def error_chain_file(name, sentinel, label):
    buggy = f'''package repository

import (
    "errors"
    "fmt"
)

var {sentinel} = errors.New("{label}")

func {name}(id string) error {{
    message := fmt.Sprintf("{label} %s: %v", id, {sentinel})
    detached := errors.New(message)
    return fmt.Errorf("repository boundary: %v", detached)
}}
'''
    fixed = f'''package repository

import (
    "errors"
    "fmt"
)

var {sentinel} = errors.New("{label}")

func {name}(id string) error {{
    if id == "" {{
        id = "<unknown>"
    }}
    wrapped := fmt.Errorf("{label} %s: %w", id, {sentinel})
    return fmt.Errorf("repository boundary: %w", wrapped)
}}
'''
    return buggy, fixed


def add_001():
    specs = [
        ('error_get.go', 'WrapCollectionMissing', 'ErrCollectionMissing', 'collection missing'),
        ('error_create.go', 'WrapCollectionConflict', 'ErrCollectionConflict', 'collection conflict'),
        ('error_update.go', 'WrapCollectionVersion', 'ErrCollectionVersion', 'collection version mismatch'),
        ('error_list.go', 'WrapCollectionPage', 'ErrCollectionPage', 'collection page invalid'),
    ]
    for rel, fn, sentinel, label in specs:
        buggy, fixed = error_chain_file(fn, sentinel, label)
        put_pair(1, f'internal/repository/{rel}', buggy, fixed)
    put_same(1, 'internal/repository/error_chain_test.go', r'''
package repository

import (
    "errors"
    "testing"
)

func TestRepositoryErrorChainGet(t *testing.T) {
    if !errors.Is(WrapCollectionMissing("c-1"), ErrCollectionMissing) { t.Fatal("missing sentinel detached") }
}
func TestRepositoryErrorChainCreate(t *testing.T) {
    if !errors.Is(WrapCollectionConflict("c-2"), ErrCollectionConflict) { t.Fatal("conflict sentinel detached") }
}
func TestRepositoryErrorChainUpdate(t *testing.T) {
    if !errors.Is(WrapCollectionVersion("c-3"), ErrCollectionVersion) { t.Fatal("version sentinel detached") }
}
func TestRepositoryErrorChainList(t *testing.T) {
    if !errors.Is(WrapCollectionPage("page"), ErrCollectionPage) { t.Fatal("page sentinel detached") }
}
''')


def context_file(fn):
    buggy = f'''package storage

import "context"

func {fn}(ctx context.Context, steps int, apply func(int) error) error {{
    runCtx := context.Background()
    for i := 0; i < steps; i++ {{
        select {{
        case <-runCtx.Done():
            return runCtx.Err()
        default:
        }}
        if err := apply(i); err != nil {{ return err }}
    }}
    return nil
}}
'''
    fixed = f'''package storage

import "context"

func {fn}(ctx context.Context, steps int, apply func(int) error) error {{
    for i := 0; i < steps; i++ {{
        select {{
        case <-ctx.Done():
            return ctx.Err()
        default:
        }}
        if err := apply(i); err != nil {{ return err }}
        if err := ctx.Err(); err != nil {{ return err }}
    }}
    return ctx.Err()
}}
'''
    return buggy, fixed


def add_002():
    specs = [('context_replay.go', 'ReplayWithContext'), ('context_append.go', 'AppendWithContext'), ('context_recovery.go', 'RecoverWithContext'), ('context_copy.go', 'CopyWithContext')]
    for rel, fn in specs:
        buggy, fixed = context_file(fn)
        put_pair(2, f'internal/storage/{rel}', buggy, fixed)
    put_same(2, 'internal/storage/context_contract_test.go', r'''
package storage

import (
    "context"
    "errors"
    "testing"
)

func canceled(t *testing.T, run func(context.Context, int, func(int) error) error) {
    t.Helper()
    ctx, cancel := context.WithCancel(context.Background())
    cancel()
    calls := 0
    err := run(ctx, 4, func(int) error { calls++; return nil })
    if !errors.Is(err, context.Canceled) || calls != 0 { t.Fatalf("err=%v calls=%d", err, calls) }
}
func TestWALReplayHonorsCanceledContext(t *testing.T) { canceled(t, ReplayWithContext) }
func TestWALAppendHonorsDeadline(t *testing.T) { canceled(t, AppendWithContext) }
func TestRecoveryStopsApplyingAfterCancel(t *testing.T) { canceled(t, RecoverWithContext) }
func TestObjectCopyStopsAfterCancel(t *testing.T) { canceled(t, CopyWithContext) }
''')


def snapshot_file(type_name, value_type, method):
    buggy = f'''package index

import "sync"

type {type_name} struct {{
    mu sync.RWMutex
    values []{value_type}
}}

func (s *{type_name}) Store(value {value_type}) {{
    s.mu.Lock()
    defer s.mu.Unlock()
    s.values = append(s.values, value)
}}

func (s *{type_name}) {method}() []{value_type} {{
    return s.values
}}
'''
    fixed = f'''package index

import "sync"

type {type_name} struct {{
    mu sync.RWMutex
    values []{value_type}
}}

func (s *{type_name}) Store(value {value_type}) {{
    s.mu.Lock()
    defer s.mu.Unlock()
    s.values = append(s.values, value)
}}

func (s *{type_name}) {method}() []{value_type} {{
    s.mu.RLock()
    defer s.mu.RUnlock()
    out := make([]{value_type}, len(s.values))
    copy(out, s.values)
    return out
}}
'''
    return buggy, fixed


def add_003():
    specs = [
        ('snapshot_view.go', 'SnapshotView', 'string', 'Copy'),
        ('document_cache.go', 'DocumentCache', 'string', 'Snapshot'),
        ('segment_catalog.go', 'SegmentCatalog', 'int', 'List'),
        ('generation_stats.go', 'GenerationStats', 'uint64', 'Values'),
    ]
    for rel, typ, value, method in specs:
        buggy, fixed = snapshot_file(typ, value, method)
        put_pair(3, f'internal/index/{rel}', buggy, fixed)
    put_same(3, 'internal/index/snapshot_isolation_test.go', r'''
package index

import (
    "sync"
    "testing"
)

func exerciseStrings(t *testing.T, store func(string), snapshot func() []string) {
    t.Helper()
    store("base")
    held := snapshot()
    start := make(chan struct{})
    var wg sync.WaitGroup
    for i := 0; i < 4; i++ { wg.Add(1); go func() { defer wg.Done(); <-start; for j := 0; j < 200; j++ { store("next"); _ = snapshot() } }() }
    close(start); wg.Wait()
    held[0] = "mutated"
    if snapshot()[0] != "base" { t.Fatal("snapshot escaped internal storage") }
}
func TestIndexSnapshotIsolation(t *testing.T) { var s SnapshotView; exerciseStrings(t, s.Store, s.Copy) }
func TestIndexConcurrentSearchUpdate(t *testing.T) { var s DocumentCache; exerciseStrings(t, s.Store, s.Snapshot) }
func TestIndexConcurrentDeleteMerge(t *testing.T) {
    var s SegmentCatalog; s.Store(1); held := s.List(); start := make(chan struct{}); var wg sync.WaitGroup
    for i := 0; i < 4; i++ { wg.Add(1); go func(v int) { defer wg.Done(); <-start; for j := 0; j < 200; j++ { s.Store(v); _ = s.List() } }(i) }
    close(start); wg.Wait(); held[0] = 99; if s.List()[0] != 1 { t.Fatal("catalog snapshot escaped") }
}
func TestIndexStatsIsolation(t *testing.T) {
    var s GenerationStats; s.Store(1); held := s.Values(); start := make(chan struct{}); var wg sync.WaitGroup
    for i := 0; i < 4; i++ { wg.Add(1); go func(v uint64) { defer wg.Done(); <-start; for j := 0; j < 200; j++ { s.Store(v); _ = s.Values() } }(uint64(i)) }
    close(start); wg.Wait(); held[0] = 99; if s.Values()[0] != 1 { t.Fatal("stats snapshot escaped") }
}
''')


def slice_file(fn, typ):
    buggy = f'''package analyzer

func {fn}(in []{typ}, keep func({typ}) bool) []{typ} {{
    out := in[:0]
    for _, value := range in {{
        if keep(value) {{ out = append(out, value) }}
    }}
    return out
}}
'''
    fixed = f'''package analyzer

func {fn}(in []{typ}, keep func({typ}) bool) []{typ} {{
    out := make([]{typ}, 0, len(in))
    for _, value := range in {{
        if keep(value) {{
            out = append(out, value)
        }}
    }}
    return out
}}
'''
    return buggy, fixed


def add_004():
    for rel, fn in [('token_filter.go', 'FilterTokens'), ('synonym_expand.go', 'ExpandSynonyms'), ('stopword_filter.go', 'RemoveStopwords'), ('pipeline_snapshot.go', 'SnapshotPipeline')]:
        buggy, fixed = slice_file(fn, 'string')
        put_pair(4, f'internal/analyzer/{rel}', buggy, fixed)
    put_same(4, 'internal/analyzer/slice_isolation_test.go', r'''
package analyzer

import (
    "reflect"
    "testing"
)

func preserves(t *testing.T, fn func([]string, func(string) bool) []string) {
    t.Helper(); original := []string{"drop", "keep", "tail"}; before := append([]string(nil), original...)
    got := fn(original, func(v string) bool { return v != "drop" })
    if !reflect.DeepEqual(original, before) { t.Fatalf("input mutated: %v", original) }
    got[0] = "changed"; if original[1] != "keep" { t.Fatalf("result aliases input: %v", original) }
}
func TestAnalyzerDoesNotMutateTokens(t *testing.T) { preserves(t, FilterTokens) }
func TestAnalyzerSynonymIsolation(t *testing.T) { preserves(t, ExpandSynonyms) }
func TestAnalyzerStopwordIsolation(t *testing.T) { preserves(t, RemoveStopwords) }
func TestRegistryReturnsDetachedPipeline(t *testing.T) { preserves(t, SnapshotPipeline) }
''')


def add_005():
    put_pair(5, 'internal/shard/result_producer.go', r'''
package shard

func ProduceResults(values []int, failAt int) <-chan int {
    out := make(chan int)
    go func() {
        for i, value := range values { if i == failAt { return }; out <- value }
        close(out)
    }()
    return out
}
''', r'''
package shard

func ProduceResults(values []int, failAt int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for i, value := range values { if i == failAt { return }; out <- value }
    }()
    return out
}
''')
    put_pair(5, 'internal/shard/error_collector.go', r'''
package shard

func CollectShardError(err error) <-chan error {
    out := make(chan error)
    go func() { out <- err; close(out) }()
    return out
}
''', r'''
package shard

func CollectShardError(err error) <-chan error {
    out := make(chan error, 1)
    out <- err
    close(out)
    return out
}
''')
    put_pair(5, 'internal/shard/wait_coordinator.go', r'''
package shard

import "sync"

func StartShardWork(gate <-chan struct{}, work func()) bool {
    var wg sync.WaitGroup
    go func() { <-gate; wg.Add(1); defer wg.Done(); work() }()
    wg.Wait()
    return true
}
''', r'''
package shard

import "sync"

func StartShardWork(gate <-chan struct{}, work func()) bool {
    var wg sync.WaitGroup
    wg.Add(1)
    go func() { defer wg.Done(); <-gate; work() }()
    wg.Wait()
    return true
}
''')
    put_pair(5, 'internal/shard/stream_consumer.go', r'''
package shard

import "context"

func ConsumeStream(ctx context.Context, input <-chan int) <-chan int {
    out := make(chan int)
    go func() { for { select { case value := <-input: out <- value; case <-ctx.Done(): return } } }()
    return out
}
''', r'''
package shard

import "context"

func ConsumeStream(ctx context.Context, input <-chan int) <-chan int {
    out := make(chan int)
    go func() { defer close(out); for { select { case value, ok := <-input: if !ok { return }; out <- value; case <-ctx.Done(): return } } }()
    return out
}
''')
    put_same(5, 'internal/shard/lifecycle_test.go', r'''
package shard

import (
    "context"
    "errors"
    "sync/atomic"
    "testing"
    "time"
)

func waitClosed(t *testing.T, ch <-chan int) { t.Helper(); done := make(chan struct{}); go func() { for range ch {}; close(done) }(); select { case <-done: case <-time.After(100*time.Millisecond): t.Fatal("channel did not close") } }
func TestCoordinatorClosesResultsOnShardError(t *testing.T) { waitClosed(t, ProduceResults([]int{1,2}, 1)) }
func TestCoordinatorDoesNotBlockErrorDelivery(t *testing.T) { select { case err := <-CollectShardError(errors.New("x")): if err == nil { t.Fatal("missing error") }; case <-time.After(100*time.Millisecond): t.Fatal("error delivery blocked") } }
func TestCoordinatorWaitsForAllShards(t *testing.T) { gate := make(chan struct{}); var ran atomic.Bool; done := make(chan struct{}); go func(){ StartShardWork(gate, func(){ ran.Store(true) }); close(done) }(); select { case <-done: t.Fatal("wait returned before shard started"); case <-time.After(20*time.Millisecond): }; close(gate); <-done; if !ran.Load() { t.Fatal("shard not run") } }
func TestMergeConsumerTerminates(t *testing.T) { ctx, cancel := context.WithCancel(context.Background()); input := make(chan int); out := ConsumeStream(ctx, input); cancel(); waitClosed(t, out) }
''')


def add_006():
    put_pair(6, 'internal/maintenance/lease_batch.go', r'''
package maintenance

func ProcessLeases(count int, acquire func() func()) {
    for i := 0; i < count; i++ { release := acquire(); defer release() }
}
''', r'''
package maintenance

func ProcessLeases(count int, acquire func() func()) {
    for i := 0; i < count; i++ { func() { release := acquire(); defer release() }() }
}
''')
    put_pair(6, 'internal/maintenance/operation_cleanup.go', r'''
package maintenance

func RunWithCleanup(operation, cleanup func() error) (err error) {
    defer func() { err = cleanup() }()
    return operation()
}
''', r'''
package maintenance

import "errors"

func RunWithCleanup(operation, cleanup func() error) (err error) {
    defer func() { err = errors.Join(err, cleanup()) }()
    return operation()
}
''')
    put_pair(6, 'internal/maintenance/rollback_guard.go', r'''
package maintenance

func RunWithRollback(operation, rollback func() error) error {
    if err := operation(); err != nil { return err }
    return nil
}
''', r'''
package maintenance

import "errors"

func RunWithRollback(operation, rollback func() error) (err error) {
    defer func() { if err != nil { err = errors.Join(err, rollback()) } }()
    return operation()
}
''')
    put_pair(6, 'internal/maintenance/close_result.go', r'''
package maintenance

func ResolveCloseResult(operation, closeResource func() error) (err error) {
    defer func() { err = closeResource() }()
    return operation()
}
''', r'''
package maintenance

import "errors"

func ResolveCloseResult(operation, closeResource func() error) (err error) {
    defer func() { err = errors.Join(err, closeResource()) }()
    return operation()
}
''')
    put_same(6, 'internal/maintenance/defer_contract_test.go', r'''
package maintenance

import (
    "errors"
    "testing"
)

func TestMaintenanceReleasesEachLease(t *testing.T) { active, peak := 0, 0; ProcessLeases(4, func() func(){ active++; if active > peak { peak = active }; return func(){active--} }); if peak != 1 || active != 0 { t.Fatalf("active=%d peak=%d", active, peak) } }
func TestMaintenancePreservesOperationError(t *testing.T) { op, clean := errors.New("operation"), errors.New("cleanup"); err := RunWithCleanup(func() error{return op}, func() error{return clean}); if !errors.Is(err, op) || !errors.Is(err, clean) { t.Fatalf("err=%v", err) } }
func TestMaintenanceRollsBackFailedTask(t *testing.T) { op, rb := errors.New("operation"), errors.New("rollback"); called := false; err := RunWithRollback(func() error{return op}, func() error{called=true; return rb}); if !called || !errors.Is(err, op) || !errors.Is(err, rb) { t.Fatalf("called=%v err=%v", called, err) } }
func TestMaintenanceJoinsCleanupError(t *testing.T) { op, closeErr := errors.New("operation"), errors.New("close"); err := ResolveCloseResult(func() error{return op}, func() error{return closeErr}); if !errors.Is(err, op) || !errors.Is(err, closeErr) { t.Fatalf("err=%v", err) } }
''')


def state_file(fn, body_buggy, body_fixed, extra=''):
    return f'''package ingest

{extra}
func {fn} {body_buggy}
''', f'''package ingest

{extra}
func {fn} {body_fixed}
'''


def add_007():
    put_pair(7, 'internal/ingest/retry_transition.go', *state_file('RetryTransition', '(current string, retryOK bool) string { if retryOK { return "retrying" }; return current }', '(current string, retryOK bool) string { if retryOK && current == "retrying" { return "succeeded" }; return current }'))
    put_pair(7, 'internal/ingest/transition_table.go', *state_file('CanMove', '(from, to string) bool { allowed := map[string]map[string]bool{"pending":{"running":true}, "running":{"retrying":true}}; return allowed[from][to] }', '(from, to string) bool { allowed := map[string]map[string]bool{"pending":{"running":true}, "running":{"retrying":true,"failed":true}, "retrying":{"succeeded":true,"failed":true}}; return allowed[from][to] }'))
    put_pair(7, 'internal/ingest/pending_view.go', *state_file('PendingStates', '(states []string) []string { out:=[]string{}; for _, state:=range states { if state=="pending" || state=="running" { out=append(out,state) } }; return out }', '(states []string) []string { out:=[]string{}; for _, state:=range states { if state=="pending" || state=="running" || state=="retrying" { out=append(out,state) } }; return out }'))
    put_pair(7, 'internal/ingest/state_audit.go', *state_file('AuditFinalState', '(workerState string) string { if workerState == "succeeded" { return "retrying" }; return workerState }', '(workerState string) string { switch workerState { case "succeeded", "failed": return workerState; default: return "in_progress" } }'))
    put_same(7, 'internal/ingest/state_machine_test.go', r'''
package ingest

import "testing"

func TestIngestRetryReachesTerminalState(t *testing.T) { if got := RetryTransition("retrying", true); got != "succeeded" { t.Fatalf("state=%s", got) } }
func TestIngestRejectsIllegalTransition(t *testing.T) { if !CanMove("retrying", "succeeded") || CanMove("succeeded", "running") { t.Fatal("transition table mismatch") } }
func TestIngestPendingViewIncludesRetrying(t *testing.T) { got:=PendingStates([]string{"pending","retrying","succeeded"}); if len(got)!=2 || got[1]!="retrying" { t.Fatalf("states=%v",got) } }
func TestIngestAuditRecordsFinalState(t *testing.T) { if got:=AuditFinalState("succeeded"); got!="succeeded" { t.Fatalf("audit=%s",got) } }
''')


def add_008():
    put_pair(8, 'internal/query/options.go', r'''
package query

type Options struct { Labels map[string]string }
func NewOptions() *Options { return &Options{} }
func (o *Options) AddLabel(k,v string) { o.Labels[k]=v }
''', r'''
package query

type Options struct { Labels map[string]string }
func NewOptions() *Options { return &Options{Labels: map[string]string{}} }
func (o *Options) AddLabel(k,v string) { if o.Labels==nil { o.Labels=map[string]string{} }; o.Labels[k]=v }
''')
    put_pair(8, 'internal/query/policy.go', r'''
package query

type Policy interface { Enabled() bool }
type RulePolicy struct { enabled bool }
func (p *RulePolicy) Enabled() bool { return p.enabled }
func PolicyEnabled(p Policy) bool { if p == nil { return false }; return p.Enabled() }
''', r'''
package query

import "reflect"

type Policy interface { Enabled() bool }
type RulePolicy struct { enabled bool }
func (p *RulePolicy) Enabled() bool { return p != nil && p.enabled }
func PolicyEnabled(p Policy) bool { if p == nil { return false }; v:=reflect.ValueOf(p); if v.Kind()==reflect.Pointer && v.IsNil(){return false}; return p.Enabled() }
''')
    put_pair(8, 'internal/query/facets.go', r'''
package query

type Facets struct { Values map[string][]string }
func (f *Facets) Add(field,value string) { f.Values[field]=append(f.Values[field],value) }
''', r'''
package query

type Facets struct { Values map[string][]string }
func (f *Facets) Add(field,value string) { if f.Values==nil { f.Values=map[string][]string{} }; f.Values[field]=append(f.Values[field],value) }
''')
    put_pair(8, 'internal/query/validation_chain.go', r'''
package query

type Validator interface { Validate(Request) error }
type ValidationChain struct { Items map[string]Validator }
func (v *ValidationChain) Add(name string, item Validator) { v.Items[name]=item }
''', r'''
package query

type Validator interface { Validate(Request) error }
type ValidationChain struct { Items map[string]Validator }
func (v *ValidationChain) Add(name string, item Validator) { if v.Items==nil { v.Items=map[string]Validator{} }; if item!=nil { v.Items[name]=item } }
''')
    put_same(8, 'internal/query/nil_paths_test.go', r'''
package query

import "testing"

func noPanic(t *testing.T, fn func()) { t.Helper(); defer func(){ if v:=recover(); v!=nil { t.Fatalf("panic: %v",v) } }(); fn() }
func TestQueryZeroValueOptionsAreUsable(t *testing.T) { noPanic(t, func(){ o:=NewOptions(); o.AddLabel("a","b"); if o.Labels["a"]!="b" {t.Fatal("missing label")} }) }
func TestQueryTypedNilPolicyIsRejected(t *testing.T) { var concrete *RulePolicy; var policy Policy=concrete; noPanic(t, func(){ if PolicyEnabled(policy) {t.Fatal("typed nil enabled")} }) }
func TestQueryFacetMapInitializes(t *testing.T) { noPanic(t, func(){ var f Facets; f.Add("brand","acme"); if len(f.Values["brand"])!=1 {t.Fatal("missing facet")} }) }
func TestQueryValidationCannotBeBypassed(t *testing.T) { noPanic(t, func(){ var c ValidationChain; c.Add("required", nil); if len(c.Items)!=0 {t.Fatal("nil validator stored")} }) }
''')


def context_bridge_file(fn):
    buggy = f'''package httpapi

import "context"

func {fn}(ctx context.Context, downstream func(context.Context) error) error {{
    detached := context.Background()
    return downstream(detached)
}}
'''
    fixed = f'''package httpapi

import "context"

func {fn}(ctx context.Context, downstream func(context.Context) error) error {{
    if err := ctx.Err(); err != nil {{
        return err
    }}
    return downstream(ctx)
}}
'''
    return buggy, fixed


def add_009():
    for rel, fn in [('cancel_bridge.go','BridgeCancellation'),('deadline_bridge.go','BridgeDeadline'),('context_slot.go','BridgeContextSlot'),('trace_context.go','RequestContextForLog')]:
        buggy, fixed = context_bridge_file(fn)
        put_pair(9, f'internal/transport/httpapi/{rel}', buggy, fixed)
    put_same(9, 'internal/transport/httpapi/context_bridge_test.go', r'''
package httpapi

import (
    "context"
    "errors"
    "testing"
    "time"
)

func assertBridge(t *testing.T, bridge func(context.Context, func(context.Context) error) error) { t.Helper(); ctx,cancel:=context.WithCancel(context.Background()); cancel(); called:=false; err:=bridge(ctx,func(got context.Context) error{called=true; return got.Err()}); if !errors.Is(err,context.Canceled)||called {t.Fatalf("err=%v called=%v",err,called)} }
func TestHTTPClientCancelStopsSearch(t *testing.T) { assertBridge(t, BridgeCancellation) }
func TestHTTPDeadlineReachesHandler(t *testing.T) { ctx,cancel:=context.WithTimeout(context.Background(),time.Millisecond); defer cancel(); seen:=false; err:=BridgeDeadline(ctx,func(got context.Context) error{_,seen=got.Deadline(); return nil}); if err!=nil||!seen {t.Fatalf("err=%v seen=%v",err,seen)} }
func TestHTTPRequestContextIsNotReused(t *testing.T) { assertBridge(t, BridgeContextSlot) }
func TestHTTPTraceUsesCurrentContext(t *testing.T) { assertBridge(t, RequestContextForLog) }
''')


def config_error_file(fn, sentinel, label):
    buggy = f'''package config

import (
    "errors"
    "fmt"
)

var {sentinel} = errors.New("{label}")
func {fn}(path string) error {{
    text := fmt.Sprintf("{label} %s: %v", path, {sentinel})
    return errors.New(text)
}}
'''
    fixed = f'''package config

import (
    "errors"
    "fmt"
)

var {sentinel} = errors.New("{label}")
func {fn}(path string) error {{
    if path == "" {{ path = "<default>" }}
    inner := fmt.Errorf("{label} %s: %w", path, {sentinel})
    return fmt.Errorf("configuration load: %w", inner)
}}
'''
    return buggy, fixed


def add_010():
    for rel, fn, sentinel, label in [
        ('open_error.go','WrapOpenError','ErrConfigOpen','config open failed'),
        ('scan_error.go','WrapScanError','ErrConfigScan','config scan failed'),
        ('validation_error.go','WrapValidationError','ErrConfigValidation','config validation failed'),
        ('classify_error.go','ClassifyConfigError','ErrConfigClassify','config classification failed')]:
        buggy, fixed = config_error_file(fn, sentinel, label)
        put_pair(10, f'internal/config/{rel}', buggy, fixed)
    put_same(10, 'internal/config/error_chain_test.go', r'''
package config

import (
    "errors"
    "testing"
)
func TestConfigPreservesOpenError(t *testing.T) { if !errors.Is(WrapOpenError("missing"),ErrConfigOpen){t.Fatal("open error detached")} }
func TestConfigPreservesScanError(t *testing.T) { if !errors.Is(WrapScanError("bad"),ErrConfigScan){t.Fatal("scan error detached")} }
func TestConfigPreservesValidationError(t *testing.T) { if !errors.Is(WrapValidationError("bad"),ErrConfigValidation){t.Fatal("validation error detached")} }
func TestConfigMainClassifiesSentinel(t *testing.T) { if !errors.Is(ClassifyConfigError("bad"),ErrConfigClassify){t.Fatal("classification error detached")} }
''')


def update_metadata():
    plan_path = ROOT / '_shared' / 'bug_plan.json'
    plan = json.loads(plan_path.read_text(encoding='utf-8'))
    for bug in plan['bugs']:
        n = int(bug['record'])
        bug['core_files'] = FILES[n]
        bug['core_symbols'] = SYMBOLS[n]
    plan_path.write_text(json.dumps(plan, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    for n, files in FILES.items():
        fp = ROOT / DATE / f'{REPO}__{n:03d}' / 'bug_fingerprint.json'
        data = json.loads(fp.read_text(encoding='utf-8'))
        data['core_files'] = files
        data['core_symbols'] = SYMBOLS[n]
        fp.write_text(json.dumps(data, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')


for n in range(1, 11):
    globals()[f'add_{n:03d}']()
update_metadata()
