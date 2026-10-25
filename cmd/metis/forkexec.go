package main

// forkexec.go — the metis#44 warm fork-server executor seam. The step-AUTHORING contract is
// untouched: a step-type is still a bash wrapper on the step path. This file recognizes the
// two-repo wrapper convention, and when it matches, routes the leaf through a warm
// `python -m metis.forkserver` (one per project root — metis's and kbench's venvs differ)
// instead of spawning `uv run → fresh python → import pandas/sklearn` (~1s measured tax) per
// step. A non-matching wrapper or a server that fails to start falls back to the legacy
// subprocess LOUDLY (once per uses-type / root) — the speedup degrades, never correctness.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync"
	"syscall"
	"time"
)

// wrapperSpec is the parse of a convention-conforming step wrapper: the uv project root the
// wrapper resolves ($0/../..) and the step module it hands to metis.trace.
type wrapperSpec struct {
	root   string
	module string
}

var (
	// The exact two-repo convention (steps/*/*, kbench#5-unified): both lines must match —
	// the ROOT resolution (so our root derivation mirrors the wrapper's own semantics) and
	// the exec line (which names the module).
	wrapperRootRe = regexp.MustCompile(`(?m)^ROOT="\$\(cd "\$\(dirname "\$0"\)/\.\./\.\." && pwd\)"$`)
	wrapperExecRe = regexp.MustCompile(`(?m)^exec uv run --project "\$ROOT" python -m metis\.trace ([A-Za-z0-9_.]+)\s*$`)
)

// parseWrapper decides forkability: a wrapper is forkable iff it follows the convention
// exactly AND its derived root looks like a uv project. Anything else → legacy exec.
func parseWrapper(exe string) (wrapperSpec, bool) {
	abs, err := filepath.Abs(exe)
	if err != nil {
		return wrapperSpec{}, false
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return wrapperSpec{}, false
	}
	if !wrapperRootRe.Match(b) {
		return wrapperSpec{}, false
	}
	m := wrapperExecRe.FindSubmatch(b)
	if m == nil {
		return wrapperSpec{}, false
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(abs))) // steps/<layer>/<name> → repo root
	if _, err := os.Stat(filepath.Join(root, "pyproject.toml")); err != nil {
		return wrapperSpec{}, false
	}
	return wrapperSpec{root: root, module: string(m[1])}, true
}

// errServerUnavailable marks "the fork-server for this root can't serve" (start failure or
// died) — the caller falls back to legacy exec. Distinct from a STEP failure, which is real.
var errServerUnavailable = errors.New("fork-server unavailable")

type forkResp struct {
	Launched     bool     // false: the server refused before fork (e.g. step.log would not open)
	ExitKnown    bool     // the server reaped the child and observed its exit status
	Exit         int      // valid when ExitKnown
	Output       string   // the log tail (metis#72)
	Joined       bool     // the server saw EOF on the child's pipe: exited and drained
	LogTruncated bool     // step.log hit its cap
	LogError     string   // the first step.log write error (output kept flowing)
	Error        string   // a server-side failure handling this request
	Runtime      *float64 // seconds, fork to exit; absent when the exit was not observed
}

// forkProtocol is the server protocol this client speaks (metis.forkserver PROTOCOL). An older
// server is refused at its ready line, before anything is dispatched: it would ignore log_path
// and cancel, so the step falls back to the legacy path instead.
const forkProtocol = 2

func checkForkProtocol(v int) error {
	if v < forkProtocol {
		return fmt.Errorf("%w: protocol %d, want %d (a server older than this metis)", errServerUnavailable, v, forkProtocol)
	}
	return nil
}

// forkServer is one warm server process (one uv project root). Response routing is by
// request id; a dead server fails all pending and future requests with errServerUnavailable.
type forkServer struct {
	stdin io.WriteCloser
	cmd   *exec.Cmd

	mu      sync.Mutex
	pending map[int]chan forkResp
	nextID  int
	dead    error // sticky: set when the reader loop exits

	ready    chan struct{} // closed when the server's {"ready":true} line arrives
	protocol int           // from the ready line (written before ready closes)
	done     chan struct{} // closed when the reader loop exits
	stderr   bytes.Buffer  // capped diagnostics for start-failure messages (guarded by mu)
}

const stderrCap = 64 << 10

// protoLine is the union of the server's stdout lines: the one-shot ready handshake and
// per-request responses.
type protoLine struct {
	Ready        bool     `json:"ready"`
	Protocol     int      `json:"protocol"`
	ID           *int     `json:"id"`
	Launched     bool     `json:"launched"`
	ExitKnown    bool     `json:"exit_known"`
	Exit         int      `json:"exit"`
	Output       string   `json:"output"`
	Joined       bool     `json:"joined"`
	LogTruncated bool     `json:"log_truncated"`
	LogError     string   `json:"log_error"`
	Error        string   `json:"error"`
	Runtime      *float64 `json:"runtime"`
}

type forkReq struct {
	ID        int               `json:"id"`
	Module    string            `json:"module"`
	Cwd       string            `json:"cwd"`
	Env       map[string]string `json:"env"`
	LogPath   string            `json:"log_path,omitempty"` // metis#72: the server streams into it
	LogCap    int               `json:"log_cap,omitempty"`
	LogMarker string            `json:"log_marker,omitempty"`
	Grace     float64           `json:"grace,omitempty"`      // seconds, for the server-owned join
	JoinBound float64           `json:"join_bound,omitempty"` // seconds
}

// forkCancel asks the server to signal an unreaped child's process group (metis#72). The server
// is the child's only reaper, so this is the only safe way to signal it: Go never signals a
// forkserver pid itself.
type forkCancel struct {
	Cancel int    `json:"cancel"`
	Signal string `json:"signal"`
}

// forkOpts are one forked step's lifecycle parameters.
type forkOpts struct {
	timeout   time.Duration // zero: none
	grace     time.Duration
	joinBound time.Duration
	logPath   string
}

// forkRun is what one forked step's lifecycle observed: the first terminal event, the response
// (if one arrived), the last signal requested, and when the request was dispatched.
type forkRun struct {
	resp    forkResp
	got     bool
	event   endEvent
	signal  string
	started time.Time
}

// errCancelledBeforeDispatch: the run was cancelled while the step waited for the server to be
// ready. Nothing was dispatched, so the step is cancelled before launch, never a legacy re-run.
var errCancelledBeforeDispatch = errors.New("cancelled before dispatch")

// startForkServer launches `uv run --project <root> python -m metis.forkserver` and wires
// the reader goroutines. The server env = ambient + the default single-thread BLAS pins
// (metis#48; names the operator exported are already excluded by blasPins — an explicit
// choice wins). Forked step children inherit this env; per-step METIS_* vars travel in
// requests, never here.
func startForkServer(root string, pins []string) (*forkServer, error) {
	cmd := exec.Command("uv", "run", "--project", root, "python", "-m", "metis.forkserver")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), pins...)
	// Own process GROUP: `uv run` spawns python as a child (no exec), so group-kill is the only
	// way to stop uv and the server together on a hung shutdown (and a test's mid-flight kill).
	// It does NOT reach forked step children: since metis#72 each leads its own group, stopped
	// only through the server's cancel while the server lives; a forced kill orphans them (and a
	// step in flight then ends cleanup-unconfirmed). Normal shutdown stays graceful (stdin EOF →
	// drain, every waiter bounded); Ctrl-C on `metis run` closes the stdin pipe, so detached
	// servers still self-exit.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	s := &forkServer{
		stdin: stdin, cmd: cmd,
		pending: map[int]chan forkResp{},
		ready:   make(chan struct{}),
		done:    make(chan struct{}),
	}
	go s.drainStderr(stderr)
	go s.readLoop(stdout)
	return s, nil
}

func (s *forkServer) drainStderr(r io.Reader) {
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			s.mu.Lock()
			if s.stderr.Len() < stderrCap {
				s.stderr.Write(buf[:n])
			}
			s.mu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

// readLoop routes response lines to their pending channels. Exit (EOF/parse trouble) marks
// the server dead and fails everything pending — callers fall back or error, never hang.
func (s *forkServer) readLoop(stdout io.Reader) {
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 1<<20), 4<<20) // responses carry tail-capped step output
	readyClosed := false
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var pl protoLine
		if err := json.Unmarshal(line, &pl); err != nil {
			continue // tolerate non-protocol noise on stdout (e.g. a chatty uv)
		}
		if pl.Ready && !readyClosed {
			readyClosed = true
			s.protocol = pl.Protocol
			close(s.ready)
			continue
		}
		if pl.ID == nil {
			continue
		}
		s.mu.Lock()
		ch := s.pending[*pl.ID]
		delete(s.pending, *pl.ID)
		s.mu.Unlock()
		if ch != nil {
			ch <- forkResp{Launched: pl.Launched, ExitKnown: pl.ExitKnown, Exit: pl.Exit, Output: pl.Output,
				Joined: pl.Joined, LogTruncated: pl.LogTruncated, LogError: pl.LogError, Error: pl.Error, Runtime: pl.Runtime}
		}
	}
	err := sc.Err()
	waitErr := s.cmd.Wait()
	s.mu.Lock()
	if err == nil {
		err = waitErr
	}
	if err == nil {
		err = errors.New("fork-server exited")
	}
	s.dead = fmt.Errorf("%w: %v; stderr:\n%s", errServerUnavailable, err, s.stderr.String())
	for id, ch := range s.pending {
		delete(s.pending, id)
		close(ch)
	}
	s.mu.Unlock()
	if !readyClosed {
		close(s.ready) // unblock ready-waiters; they read s.dead
	}
	close(s.done)
}

// errDispatchedLost marks "the request was (possibly) handed to the server before it died" —
// the forked child may STILL BE RUNNING and writing into stepDir, so the caller must ERROR
// the step, never re-run it via legacy (double-execution would corrupt the step's outputs).
// Close-review I1: only a PRE-dispatch failure is safe to fall back on.
var errDispatchedLost = errors.New("fork-server died with the request in flight")

// execute runs one step request to its first terminal event: its response, its timeout, or the
// run's cancellation (metis#72). On a timeout or cancel it asks the server to TERM the child's
// group, then KILL after grace, and waits for the response within a bound that covers the
// server's own post-exit join (join, grace, join). errServerUnavailable: nothing dispatched,
// legacy fallback is safe. errCancelledBeforeDispatch: the ready wait was cancelled, nothing
// dispatched. errDispatchedLost: the server died with the request in flight, the child may
// still run. A response, even exit != 0, is a real outcome; one that lands after a timeout or
// cancel does not change the event (the first observed wins).
func (s *forkServer) execute(ctx context.Context, module, cwd string, env map[string]string, o forkOpts) (forkRun, error) {
	select {
	case <-s.ready:
	case <-ctx.Done():
		return forkRun{}, errCancelledBeforeDispatch
	}
	if err := checkForkProtocol(s.protocol); err != nil {
		return forkRun{}, err // nothing dispatched: fall back to legacy
	}
	s.mu.Lock()
	if s.dead != nil { // nothing dispatched yet — safe to fall back
		defer s.mu.Unlock()
		return forkRun{}, s.dead
	}
	s.nextID++
	id := s.nextID
	ch := make(chan forkResp, 1)
	s.pending[id] = ch
	b, err := json.Marshal(forkReq{ID: id, Module: module, Cwd: cwd, Env: env, LogPath: o.logPath,
		LogCap: stepLogCap, LogMarker: truncationMarker, Grace: o.grace.Seconds(), JoinBound: o.joinBound.Seconds()})
	run := forkRun{started: time.Now(), event: endExited} // dispatch is the launch on this path
	if err == nil {
		_, err = s.stdin.Write(append(b, '\n'))
	}
	if err != nil {
		// The write was ATTEMPTED — a partial line may have reached a dying server. Treat as
		// dispatched (conservative: no legacy re-run against a possibly-forked child).
		delete(s.pending, id)
		s.mu.Unlock()
		return run, fmt.Errorf("%w: write request: %v", errDispatchedLost, err)
	}
	s.mu.Unlock()
	defer func() { // a request left unanswered (cancel bound, lost server) leaves no entry behind
		s.mu.Lock()
		delete(s.pending, id)
		s.mu.Unlock()
	}()

	var deadline <-chan time.Time
	if o.timeout > 0 {
		timer := time.NewTimer(o.timeout)
		defer timer.Stop()
		deadline = timer.C
	}
	receive := func(resp forkResp, ok bool) error {
		if !ok { // closed by readLoop's death path — the request was in flight
			s.mu.Lock()
			defer s.mu.Unlock()
			return fmt.Errorf("%w: %v", errDispatchedLost, s.dead)
		}
		run.resp, run.got = resp, true
		return nil
	}
	select {
	case resp, ok := <-ch:
		return run, receive(resp, ok)
	case <-deadline:
		run.event = endDeadline
	case <-ctx.Done():
		run.event = endCancelled
	}
	for _, step := range []struct {
		sig  string
		wait time.Duration
	}{{"TERM", o.grace}, {"KILL", 2*o.joinBound + o.grace + time.Second}} {
		run.signal = step.sig
		if err := s.cancel(id, step.sig); err != nil {
			return run, fmt.Errorf("%w: cancel: %v", errDispatchedLost, err)
		}
		select {
		case resp, ok := <-ch:
			return run, receive(resp, ok)
		case <-time.After(step.wait):
		}
	}
	return run, nil // no response within the bound: the join is unconfirmed
}

// cancel writes a signal request for an in-flight id.
func (s *forkServer) cancel(id int, sig string) error {
	b, err := json.Marshal(forkCancel{Cancel: id, Signal: sig})
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dead != nil {
		return s.dead
	}
	_, err = s.stdin.Write(append(b, '\n'))
	return err
}

func (s *forkServer) shutdown() {
	_ = s.stdin.Close() // EOF → server drains in-flight children and exits
	select {
	case <-s.done:
	case <-time.After(60 * time.Second): // a wedged server (its waiters are bounded) — group-kill uv
		// and the server; forked step children lead their own groups and are not reached here
		_ = syscall.Kill(-s.cmd.Process.Pid, syscall.SIGKILL)
		<-s.done
	}
}

// serverPool lazily starts one forkServer per project root and remembers roots whose start
// failed (so a broken layer degrades to legacy once, loudly, not per-leaf).
type serverPool struct {
	mu      sync.Mutex
	servers map[string]*forkServer
	broken  map[string]bool
	warned  map[string]bool
	out     io.Writer
	outMu   sync.Mutex
	pins    []string // metis#48: default leaf BLAS pins, applied to every server's spawn env
}

func newServerPool(out io.Writer, pins []string) *serverPool {
	return &serverPool{
		servers: map[string]*forkServer{},
		broken:  map[string]bool{},
		warned:  map[string]bool{},
		out:     out,
		pins:    pins,
	}
}

// noticeOnce prints one loud line per key — the escape-hatch visibility contract: falling
// back to legacy exec is fine, doing it silently is not.
func (p *serverPool) noticeOnce(key, msg string) {
	p.outMu.Lock()
	defer p.outMu.Unlock()
	if p.warned[key] {
		return
	}
	p.warned[key] = true
	fmt.Fprintf(p.out, "metis: forkserver: %s\n", msg)
}

// execute routes a parsed step through the root's warm server. Outcomes:
//   - (run, true, nil)  — dispatched; run holds what the lifecycle observed (a response, or none
//     within the bound after KILL: the join is unconfirmed, and the root is marked broken);
//   - (_, false, nil)   — nothing dispatched (broken/unstartable server) → legacy is SAFE;
//   - (_, false, errCancelledBeforeDispatch) — the run was cancelled before dispatch;
//   - (_, true, err)    — dispatched-and-lost → the caller must ERROR the step (I1: the forked
//     child may still be running; a legacy re-run would double-execute into stepDir).
func (p *serverPool) execute(ctx context.Context, spec wrapperSpec, cwd string, env map[string]string, o forkOpts) (forkRun, bool, error) {
	p.mu.Lock()
	if p.broken[spec.root] {
		p.mu.Unlock()
		return forkRun{}, false, nil
	}
	s := p.servers[spec.root]
	if s == nil {
		var err error
		s, err = startForkServer(spec.root, p.pins)
		if err != nil {
			p.broken[spec.root] = true
			p.mu.Unlock()
			p.noticeOnce("root:"+spec.root, fmt.Sprintf("start failed for %s (%v) — legacy exec for this root", spec.root, err))
			return forkRun{}, false, nil
		}
		p.servers[spec.root] = s
	}
	p.mu.Unlock()

	run, err := s.execute(ctx, spec.module, cwd, env, o)
	switch {
	case errors.Is(err, errCancelledBeforeDispatch):
		return forkRun{}, false, err
	case errors.Is(err, errDispatchedLost):
		p.markBroken(spec.root, fmt.Sprintf("server for %s died mid-flight (%v) — erroring the step; later leaves use legacy", spec.root, err))
		return run, true, err
	case err != nil:
		p.markBroken(spec.root, fmt.Sprintf("server for %s unavailable (%v) — legacy exec for this root", spec.root, err))
		return forkRun{}, false, nil
	case !run.got:
		p.markBroken(spec.root, fmt.Sprintf("server for %s did not answer a cancelled step within its bound — later leaves use legacy", spec.root))
	}
	return run, true, nil
}

func (p *serverPool) markBroken(root, msg string) {
	p.mu.Lock()
	p.broken[root] = true
	p.mu.Unlock()
	p.noticeOnce("root:"+root, msg)
}

// shutdown closes every server (EOF-drain). Deferred by runExperiment.
func (p *serverPool) shutdown() {
	p.mu.Lock()
	servers := make([]*forkServer, 0, len(p.servers))
	for _, s := range p.servers {
		servers = append(servers, s)
	}
	p.servers = map[string]*forkServer{}
	p.mu.Unlock()
	for _, s := range servers {
		s.shutdown()
	}
}
