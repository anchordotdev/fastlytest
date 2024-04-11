//go:build !wasip1 || nofastlyhostcalls

package fastlytest

import (
	"context"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml"
	"github.com/shirou/gopsutil/process"
)

type Viceroy struct {
	GoPath      string
	ViceroyPath string

	CmdArgs    []string
	ConfigPath string
	Env        []string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	tmpDir string
}

func NewViceroy(cfg Config) (*Viceroy, error) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		return nil, err
	}

	vicPath, err := exec.LookPath("viceroy")
	if err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "viceroy-runner")
	if err != nil {
		return nil, err
	}

	cfgFile, err := os.Create(filepath.Join(tmpDir, "config.toml"))
	if err != nil {
		return nil, err
	}

	if err := toml.NewEncoder(cfgFile).Encode(cfg); err != nil {
		return nil, err
	}
	if err := cfgFile.Close(); err != nil {
		return nil, err
	}

	return &Viceroy{
		GoPath:      goPath,
		ViceroyPath: vicPath,

		ConfigPath: cfgFile.Name(),
		Env:        wasmEnv(),

		tmpDir: tmpDir,
	}, nil
}

func (v *Viceroy) Cleanup() error {
	if v.tmpDir != "" {
		return os.RemoveAll(v.tmpDir)
	}
	return nil
}

func (v *Viceroy) GoTestPkg(ctx context.Context, pkgName string, testArgs ...string) *exec.Cmd {
	args := append(cmdArgs(pkgName), testArgs...)
	return v.GoTestCmd(ctx, args...)
}

func (v *Viceroy) GoTestCmd(ctx context.Context, testArgs ...string) *exec.Cmd {
	stdout := v.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}

	stderr := v.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	vicArgs := strings.Join([]string{v.ViceroyPath, "run", "--config", v.ConfigPath}, " ")

	args := []string{"test", "-exec", vicArgs}
	args = append(args, testArgs...)

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Path = v.GoPath
	cmd.Env = v.Env

	cmd.Stdin = v.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	return cmd
}

func wasmEnv() []string {
	env := []string{"GOARCH=wasm", "GOOS=wasip1"}
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, "GOARCH=") && !strings.HasPrefix(v, "GOOS=") {
			env = append(env, v)
		}
	}
	return env
}

func cmdArgs(pkgName string) []string {
	// lookup extra command line arguments from parent 'go test' process

	pproc, err := process.NewProcess(int32(os.Getppid()))
	if err != nil {
		panic(err)
	}
	cmdLine, err := pproc.Cmdline()
	if err != nil {
		panic(err)
	}

	// remove the 'test.' prefix from all flags, then re-parse with the
	// parent process' cmdline

	var flags flag.FlagSet
	flags.Usage = func() {}
	flags.SetOutput(io.Discard)

	flag.CommandLine.VisitAll(func(flg *flag.Flag) {
		if strings.HasPrefix(flg.Name, "test.") {
			flags.Var(flg.Value, strings.TrimPrefix(flg.Name, "test."), flg.Usage)
		}
	})

	for {
		if err = flags.Parse(strings.Fields(cmdLine)[2:]); err == nil {
			break
		}
		if strings.HasPrefix(err.Error(), "flag provided but not defined: -") {
			name := strings.TrimPrefix(err.Error(), "flag provided but not defined: -")
			flags.String(name, "", "")
		} else {
			panic(err)
		}
	}

	// collect flags and arguments

	var args []string
	flags.Visit(func(flg *flag.Flag) {
		args = append(args, "-"+flg.Name+"="+flg.Value.String())
	})

	for _, pkg := range flags.Args() {
		if strings.HasPrefix(pkg, "./"+pkgName) {
			pkg = "./" + strings.TrimPrefix(pkg, "./"+pkgName)
		}
		args = append(args, pkg)
	}

	return args
}
