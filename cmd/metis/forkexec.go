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
	Exit   int
	Output string
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

	ready  chan struct{} // closed when the server's {"ready":true} line arrives
	done   chan struct{} // closed when the reader loop exits
	stderr bytes.Buffer  // capped diagnostics for start-failure messages (guarded by mu)
}

const stderrCap = 64 << 10

// protoLine is the union of the server's stdout lines: the one-shot ready handshake and
// per-request responses.
type protoLine struct {
	Ready  bool   `json:"ready"`
	ID     *int   `json:"id"`
	Exit   int    `json:"exit"`
	Output string `json:"output"`
}

type forkReq struct {
	ID     int               `json:"id"`
	Module string            `json:"module"`
	Cwd    string            `json:"cwd"`
	Env    map[string]string `json:"env"`
}

// startForkServer launches `uv run --project <root> python -m metis.forkserver` and wires
// the reader goroutines. The server env = ambient + the default single-thread BLAS pins
// (metis#48; names the operator exported are already excluded by blasPins — an explicit
// choice wins). Forked step children inherit this env; per-step METIS_* vars travel in
// requests, never here.
func startForkServer(root string, pins []string) (*forkServer, error) {
	cmd := exec.Command("uv", "run", "--project", root, "python", "-m", "metis.forkserver")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), pins...)
	// Own process GROUP: `uv run` spawns python as a child (no exec), and the server forks
	// step children — group-kill is the only way to reap the whole tree on a hung shutdown
	// (and a test's mid-flight kill). Normal shutdown stays graceful (stdin EOF → drain);
	// Ctrl-C on `metis run` closes the stdin pipe, so detached servers still self-exit.
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
			ch <- forkResp{Exit: pl.Exit, Output: pl.Output}
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

// execute runs one step request to completion. errServerUnavailable = nothing dispatched,
// legacy fallback is safe; errDispatchedLost = the child may be running, error the step; any
// response — even exit != 0 — is a real step outcome.
func (s *forkServer) execute(module, cwd string, env map[string]string) (forkResp, error) {
	<-s.ready
	s.mu.Lock()
	if s.dead != nil { // nothing dispatched yet — safe to fall back
		defer s.mu.Unlock()
		return forkResp{}, s.dead
	}
	s.nextID++
	id := s.nextID
	ch := make(chan forkResp, 1)
	s.pending[id] = ch
	b, err := json.Marshal(forkReq{ID: id, Module: module, Cwd: cwd, Env: env})
	if err == nil {
		_, err = s.stdin.Write(append(b, '\n'))
	}
	if err != nil {
		// The write was ATTEMPTED — a partial line may have reached a dying server. Treat as
		// dispatched (conservative: no legacy re-run against a possibly-forked child).
		delete(s.pending, id)
		s.mu.Unlock()
		return forkResp{}, fmt.Errorf("%w: write request: %v", errDispatchedLost, err)
	}
	s.mu.Unlock()
	resp, ok := <-ch
	if !ok { // channel closed by readLoop's death path — request was in flight
		s.mu.Lock()
		defer s.mu.Unlock()
		return forkResp{}, fmt.Errorf("%w: %v", errDispatchedLost, s.dead)
	}
	return resp, nil
}

func (s *forkServer) shutdown() {
	_ = s.stdin.Close() // EOF → server drains in-flight children and exits
	select {
	case <-s.done:
	case <-time.After(60 * time.Second): // a hung drain (e.g. a wedged child) — group-kill
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
//   - (resp, true, nil)  — a real step outcome (even exit != 0);
//   - (_, false, nil)    — nothing dispatched (broken/unstartable server) → legacy is SAFE;
//   - (_, false, err)    — dispatched-and-lost → the caller must ERROR the step (I1: the
//     forked child may still be running; a legacy re-run would double-execute into stepDir).
func (p *serverPool) execute(spec wrapperSpec, cwd string, env map[string]string) (forkResp, bool, error) {
	p.mu.Lock()
	if p.broken[spec.root] {
		p.mu.Unlock()
		return forkResp{}, false, nil
	}
	s := p.servers[spec.root]
	if s == nil {
		var err error
		s, err = startForkServer(spec.root, p.pins)
		if err != nil {
			p.broken[spec.root] = true
			p.mu.Unlock()
			p.noticeOnce("root:"+spec.root, fmt.Sprintf("start failed for %s (%v) — legacy exec for this root", spec.root, err))
			return forkResp{}, false, nil
		}
		p.servers[spec.root] = s
	}
	p.mu.Unlock()

	resp, err := s.execute(spec.module, cwd, env)
	if err != nil {
		p.mu.Lock()
		p.broken[spec.root] = true
		p.mu.Unlock()
		if errors.Is(err, errDispatchedLost) {
			p.noticeOnce("root:"+spec.root, fmt.Sprintf("server for %s died mid-flight (%v) — erroring the step; later leaves use legacy", spec.root, err))
			return forkResp{}, false, err
		}
		p.noticeOnce("root:"+spec.root, fmt.Sprintf("server for %s unavailable (%v) — legacy exec for this root", spec.root, err))
		return forkResp{}, false, nil
	}
	return resp, true, nil
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
