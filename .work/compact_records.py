from pathlib import Path

ROOT=Path('/Users/hutu/Desktop/go_projects/work_projects/013-distributed-search-service'); DATE='2026-08-21'; REPO='distributed-search-service'
def put(n, rel, bad, good):
    for base,text in ((ROOT/DATE/f'{REPO}__{n:03d}'/'env',bad),(ROOT/'_gold'/f'{REPO}__{n:03d}',good)):
        p=base/rel; p.parent.mkdir(parents=True,exist_ok=True); p.write_text(text.lstrip(),encoding='utf-8')

def err(pkg, fn, sent, label, wrap):
    if wrap:
        body=f'''func {fn}(id string) error {{
    if id == "" {{ id = "<unknown>" }}
    wrapped := fmt.Errorf("{label} %s: %w", id, {sent})
    return fmt.Errorf("boundary: %w", wrapped)
}}'''
    else:
        body=f'''func {fn}(id string) error {{
    message := fmt.Sprintf("{label} %s: %v", id, {sent})
    detached := errors.New(message)
    return fmt.Errorf("boundary: %v", detached)
}}'''
    return f'''package {pkg}
import ("errors"; "fmt")
var {sent} = errors.New("{label}")
{body}
'''

for n, pkg, specs in [
    (1,'repository',[('error_get.go','WrapCollectionMissing','ErrCollectionMissing','collection missing'),('error_create.go','WrapCollectionConflict','ErrCollectionConflict','collection conflict'),('error_update.go','WrapCollectionVersion','ErrCollectionVersion','collection version mismatch'),('error_list.go','WrapCollectionPage','ErrCollectionPage','collection page invalid')]),
    (10,'config',[('open_error.go','WrapOpenError','ErrConfigOpen','config open failed'),('scan_error.go','WrapScanError','ErrConfigScan','config scan failed'),('validation_error.go','WrapValidationError','ErrConfigValidation','config validation failed'),('classify_error.go','ClassifyConfigError','ErrConfigClassify','config classification failed')]),
]:
    for rel,fn,sent,label in specs: put(n,f'internal/{pkg}/{rel}',err(pkg,fn,sent,label,False),err(pkg,fn,sent,label,True))

def ctx(fn, good):
    helper = 'walk' + fn
    body=(f'''func {fn}(ctx context.Context, steps int, apply func(int) error) error {{
    if ctx == nil {{ ctx = context.Background() }}
    return {helper}(ctx, steps, apply)
}}''' if good else f'''func {fn}(ctx context.Context, steps int, apply func(int) error) error {{
    detached := context.Background()
    return {helper}(detached, steps, apply)
}}''')
    return 'package storage\nimport "context"\n'+body+f'''\nfunc {helper}(ctx context.Context, steps int, apply func(int) error) error {{
    for i:=0; i<steps; i++ {{ select {{ case <-ctx.Done(): return ctx.Err(); default: }}; if err:=apply(i); err!=nil{{return err}} }}
    return ctx.Err()
}}\n'''
for rel,fn in [('context_replay.go','ReplayWithContext'),('context_append.go','AppendWithContext'),('context_recovery.go','RecoverWithContext'),('context_copy.go','CopyWithContext')]: put(2,'internal/storage/'+rel,ctx(fn,False),ctx(fn,True))

def snap(typ,val,method,good):
    if good:
        body=f'''type {typ} struct {{ mu sync.RWMutex; values []{val}; generation uint64 }}
func (s *{typ}) Store(v {val}) {{ s.mu.Lock(); s.values=append(s.values,v); s.generation++; s.mu.Unlock() }}
func (s *{typ}) {method}() []{val} {{ s.mu.RLock(); defer s.mu.RUnlock(); out:=make([]{val},len(s.values)); copy(out,s.values); return out }}
func (s *{typ}) Generation() uint64 {{ s.mu.RLock(); defer s.mu.RUnlock(); return s.generation }}'''
    else:
        body=f'''type {typ} struct {{ mu sync.RWMutex; values []{val} }}
func (s *{typ}) Store(v {val}) {{ s.mu.Lock(); s.values=append(s.values,v); s.mu.Unlock() }}
func (s *{typ}) {method}() []{val} {{ return s.values }}
func (s *{typ}) Generation() uint64 {{ return uint64(len(s.values)) }}'''
    return 'package index\nimport "sync"\n'+body+'\n'
for rel,typ,val,method in [('snapshot_view.go','SnapshotView','string','Copy'),('document_cache.go','DocumentCache','string','Snapshot'),('segment_catalog.go','SegmentCatalog','int','List'),('generation_stats.go','GenerationStats','uint64','Values')]: put(3,'internal/index/'+rel,snap(typ,val,method,False),snap(typ,val,method,True))

def sl(fn,good):
    helper = 'appendKept' + fn
    body=(f'''func {fn}(in []string, keep func(string) bool) []string {{
    capacity := len(in)
    out := make([]string, 0, capacity)
    return {helper}(out, in, keep)
}}''' if good else f'''func {fn}(in []string, keep func(string) bool) []string {{
    capacity := cap(in)
    out := in[:0:capacity]
    return {helper}(out, in, keep)
}}''')
    return 'package analyzer\n'+body+f'''\nfunc {helper}(out, in []string, keep func(string) bool) []string {{ for _,v:=range in {{if keep(v){{out=append(out,v)}}}}; return out }}\n'''
for rel,fn in [('token_filter.go','FilterTokens'),('synonym_expand.go','ExpandSynonyms'),('stopword_filter.go','RemoveStopwords'),('pipeline_snapshot.go','SnapshotPipeline')]: put(4,'internal/analyzer/'+rel,sl(fn,False),sl(fn,True))

def shard(rel,bad,good): put(5,'internal/shard/'+rel,bad,good)
shard('result_producer.go','''package shard
func ProduceResults(values []int, failAt int) <-chan int {
 out:=make(chan int); go func(){ for i,v:=range values { if i==failAt{return}; out<-v }; close(out) }(); return out
}
''','''package shard
func ProduceResults(values []int, failAt int) <-chan int {
 out:=make(chan int); go func(){ defer close(out); for i,v:=range values { if i==failAt{return}; out<-v } }(); return out
}
''')
shard('error_collector.go','''package shard
func CollectShardError(err error) (<-chan error, <-chan struct{}) {
 out:=make(chan error); done:=make(chan struct{}); go func(){ out<-err; close(out); close(done) }(); return out,done
}
''','''package shard
func CollectShardError(err error) (<-chan error, <-chan struct{}) {
 out:=make(chan error,1); done:=make(chan struct{}); out<-err; close(out); close(done); return out,done
}
''')
shard('wait_coordinator.go','''package shard
import "sync"
func StartShardWork(gate <-chan struct{}, work func()) bool {
 var wg sync.WaitGroup; go func(){ <-gate; wg.Add(1); defer wg.Done(); work() }(); wg.Wait(); return true
}
''','''package shard
import "sync"
func StartShardWork(gate <-chan struct{}, work func()) bool {
 var wg sync.WaitGroup; wg.Add(1); go func(){ defer wg.Done(); <-gate; work() }(); wg.Wait(); return true
}
''')
shard('stream_consumer.go','''package shard
import "context"
func ConsumeStream(ctx context.Context, input <-chan int) <-chan int {
 out:=make(chan int); go func(){ for { select { case v:=<-input: out<-v; case <-ctx.Done(): return } } }(); return out
}
''','''package shard
import "context"
func ConsumeStream(ctx context.Context, input <-chan int) <-chan int {
 out:=make(chan int); go func(){ defer close(out); for { select { case v,ok:=<-input: if !ok{return}; out<-v; case <-ctx.Done(): return } } }(); return out
}
''')

put(6,'internal/maintenance/lease_batch.go','''package maintenance
func ProcessLeases(count int, acquire func() func()) { for i:=0;i<count;i++ { release:=acquire(); defer release() } }
''','''package maintenance
func ProcessLeases(count int, acquire func() func()) { for i:=0;i<count;i++ { func(){ release:=acquire(); defer release() }() } }
''')
put(6,'internal/maintenance/operation_cleanup.go','''package maintenance
func RunWithCleanup(operation,cleanup func() error)(err error){ defer func(){err=cleanup()}(); return operation() }
''','''package maintenance
import "errors"
func RunWithCleanup(operation,cleanup func() error)(err error){ defer func(){err=errors.Join(err,cleanup())}(); return operation() }
''')
put(6,'internal/maintenance/rollback_guard.go','''package maintenance
func RunWithRollback(operation,rollback func() error) error { if err:=operation(); err!=nil{return err}; return nil }
''','''package maintenance
import "errors"
func RunWithRollback(operation,rollback func() error)(err error){ defer func(){if err!=nil{err=errors.Join(err,rollback())}}(); return operation() }
''')
put(6,'internal/maintenance/close_result.go','''package maintenance
func ResolveCloseResult(operation,closeResource func() error)(err error){ defer func(){err=closeResource()}(); return operation() }
''','''package maintenance
import "errors"
func ResolveCloseResult(operation,closeResource func() error)(err error){ defer func(){err=errors.Join(err,closeResource())}(); return operation() }
''')

def state(n,rel,fn,bad,good): put(n,'internal/ingest/'+rel,f'package ingest\nfunc {fn}{bad}\n',f'package ingest\nfunc {fn}{good}\n')
state(7,'retry_transition.go','RetryTransition','(current string,retryOK bool) string { if retryOK{return "retrying"}; return current }','(current string,retryOK bool) string { if retryOK&&current=="retrying"{return "succeeded"}; return current }')
state(7,'transition_table.go','CanMove','(from,to string) bool { a:=map[string]map[string]bool{"pending":{"running":true},"running":{"retrying":true}}; return a[from][to] }','(from,to string) bool { a:=map[string]map[string]bool{"pending":{"running":true},"running":{"retrying":true,"failed":true},"retrying":{"succeeded":true,"failed":true}}; return a[from][to] }')
state(7,'pending_view.go','PendingStates','(states []string) []string { out:=[]string{}; for _,s:=range states {if s=="pending"||s=="running"{out=append(out,s)}}; return out }','(states []string) []string { out:=[]string{}; for _,s:=range states {if s=="pending"||s=="running"||s=="retrying"{out=append(out,s)}}; return out }')
state(7,'state_audit.go','AuditFinalState','(state string) string { if state=="succeeded"{return "retrying"}; return state }','(state string) string { if state=="succeeded"||state=="failed"{return state}; return "in_progress" }')

put(8,'internal/query/options.go','''package query
type Options struct{Labels map[string]string}
func NewOptions()*Options{return &Options{}}
func(o *Options)AddLabel(k,v string){o.Labels[k]=v}
''','''package query
type Options struct{Labels map[string]string}
func NewOptions()*Options{return &Options{Labels:map[string]string{}}}
func(o *Options)AddLabel(k,v string){if o.Labels==nil{o.Labels=map[string]string{}};o.Labels[k]=v}
''')
put(8,'internal/query/policy.go','''package query
type Policy interface{Enabled()bool}; type RulePolicy struct{enabled bool}
func(p *RulePolicy)Enabled()bool{return p.enabled}
func PolicyEnabled(p Policy)bool{if p==nil{return false};return p.Enabled()}
''','''package query
import "reflect"
type Policy interface{Enabled()bool}; type RulePolicy struct{enabled bool}
func(p *RulePolicy)Enabled()bool{return p!=nil&&p.enabled}
func PolicyEnabled(p Policy)bool{if p==nil{return false};v:=reflect.ValueOf(p);if v.Kind()==reflect.Pointer&&v.IsNil(){return false};return p.Enabled()}
''')
put(8,'internal/query/facets.go','''package query
type Facets struct{Values map[string][]string}
func(f *Facets)Add(k,v string){f.Values[k]=append(f.Values[k],v)}
''','''package query
type Facets struct{Values map[string][]string}
func(f *Facets)Add(k,v string){if f.Values==nil{f.Values=map[string][]string{}};f.Values[k]=append(f.Values[k],v)}
''')
put(8,'internal/query/validation_chain.go','''package query
type Validator interface{Validate(Request)error}; type ValidationChain struct{Items map[string]Validator}
func(v *ValidationChain)Add(k string,item Validator){v.Items[k]=item}
''','''package query
type Validator interface{Validate(Request)error}; type ValidationChain struct{Items map[string]Validator}
func(v *ValidationChain)Add(k string,item Validator){if v.Items==nil{v.Items=map[string]Validator{}};if item!=nil{v.Items[k]=item}}
''')

def bridge(fn,good):
    return 'package httpapi\nimport "context"\n'+(f'''func {fn}(ctx context.Context,next func(context.Context)error)error{{ if err:=ctx.Err();err!=nil{{return err}};return next(ctx) }}\n''' if good else f'''func {fn}(ctx context.Context,next func(context.Context)error)error{{ detached:=context.Background();return next(detached) }}\n''')
for rel,fn in [('cancel_bridge.go','BridgeCancellation'),('deadline_bridge.go','BridgeDeadline'),('context_slot.go','BridgeContextSlot'),('request_log_context.go','RequestContextForLog')]: put(9,'internal/transport/httpapi/'+rel,bridge(fn,False),bridge(fn,True))
