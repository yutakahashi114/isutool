package trace

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"syscall"
	"time"
)

func (t *Tracer) StartProfileCPU(duration time.Duration) error {
	f, err := os.Create(t.filePrefix + "_cpu.pprof")
	if err != nil {
		return err
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		return err
	}
	go func() {
		time.Sleep(duration)
		pprof.StopCPUProfile()
		f.Close()
		cmd := exec.Command("go", "tool", "pprof", "-http", ":9090", "-no_browser", t.filePrefix+"_cpu.pprof")
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		err := cmd.Start()
		if err != nil {
			log.Println(err)
			return
		}
		time.Sleep(5 * time.Second)
		res, err := http.DefaultClient.Get("http://localhost:9090/ui/flamegraph")
		if err != nil {
			log.Println(err)
			return
		}
		defer res.Body.Close()
		f, err := os.Create(t.filePrefix + "_cpu.html")
		if err != nil {
			log.Println(err)
			return
		}
		defer f.Close()
		io.Copy(f, res.Body)
		err = syscall.Kill(cmd.Process.Pid, syscall.SIGINT)
		if err != nil {
			log.Println(err)
		}
		err = cmd.Wait()
		if err != nil {
			log.Println(err)
		}
	}()
	return nil
}

func (t *Tracer) StartProfileMem(duration time.Duration) error {
	f, err := os.Create(t.filePrefix + "_mem.mprof")
	if err != nil {
		return err
	}
	go func() {
		defer f.Close()
		time.Sleep(duration)
		runtime.GC()
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Println(err)
		}
	}()
	return nil
}

func (t *Tracer) StartProfileTrace(duration time.Duration) error {
	f, err := os.Create(t.filePrefix + "_trace.out")
	if err != nil {
		return err
	}
	trace.Start(f)
	go func() {
		time.Sleep(duration)
		trace.Stop()
		f.Close()
	}()
	return nil
}
