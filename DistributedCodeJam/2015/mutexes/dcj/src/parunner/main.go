package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"text/tabwriter"
	"time"
)

const MaxInstances = 100

var nInstances = flag.Int("n", 1, fmt.Sprintf("Number of instances; must be from the [1,%d] range", MaxInstances))
var stdoutHandling = flag.String("stdout", "contest", "Stdout handling: contest, all, tagged, files")
var stderrHandling = flag.String("stderr", "all", "Stderr handling: all, tagged, files")
var filesPrefix = flag.String("prefix", "", "Filename prefix for files generated by -stdout=files and -stderr=files")
var warnRemaining = flag.Bool("warn_unreceived", true, "Warn about messages that remain unreceived after instance's termination")
var stats = flag.Bool("print_stats", false, "Print per-instance statistics")
var traceCommunications = flag.Bool("trace_comm", false, "Print out a trace of all messages exchanged")

var binaryPath string

func writeFile(streamType string, i int, r io.Reader) error {
	binaryDir, binaryFile := filepath.Split(binaryPath)
	if idx := strings.LastIndex(binaryFile, "."); idx != -1 {
		binaryFile = binaryFile[:idx]
	}
	basename := filepath.Join(binaryDir, binaryFile)
	if *filesPrefix != "" {
		basename = *filesPrefix
	}
	filename := fmt.Sprintf("%s.%s.%d", basename, streamType, i)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [flags] binary_to_run\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `Output handling modes:
  contest: Fail if more than one instance write any output. Redirect the output to the standard output of this program.
  all: Redirect all the instances' outputs to the corresponding output of this program.
  tagged: Redirect all the instances' outputs to the corresponding output of this program, while prefixing each line with instance number.
  files: Store output of each instance in a separate file.
`)
}

func main() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	flag.Usage = Usage
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Specify the binary name\n")
		flag.Usage()
		os.Exit(1)
	}
	var err error
	binaryPath, err = filepath.Abs(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot find absolute path of the binary: %v\n", err)
		os.Exit(1)
	}

	if *nInstances < 1 || *nInstances > MaxInstances {
		fmt.Fprintf(os.Stderr, "Number of instances should be from [1,%d], but %d was given\n", MaxInstances, *nInstances)
		flag.Usage()
		os.Exit(1)
	}

	var writeStdout func(int, io.Reader) error
	contestStdout := &ContestStdout{Output: os.Stdout}
	switch *stdoutHandling {
	case "contest":
		// This is handled specially (without a pipe) below.
	case "all":
	case "tagged":
		writeStdout = func(i int, r io.Reader) error { return TagStream(fmt.Sprintf("STDOUT %d: ", i), os.Stdout, r) }
	case "files":
		writeStdout = func(i int, r io.Reader) error { return writeFile("stdout", i, r) }
	default:
		fmt.Fprintf(os.Stderr, "Invalid stdout handling mode: %s", *stdoutHandling)
		flag.Usage()
		os.Exit(1)
	}
	var writeStderr func(int, io.Reader) error
	switch *stderrHandling {
	case "all":
	case "tagged":
		writeStderr = func(i int, r io.Reader) error { return TagStream(fmt.Sprintf("STDERR %d: ", i), os.Stderr, r) }
	case "files":
		writeStdout = func(i int, r io.Reader) error { return writeFile("stderr", i, r) }
	default:
		fmt.Fprintf(os.Stderr, "Inalid stderr handling mode: %s", *stdoutHandling)
		flag.Usage()
		os.Exit(1)
	}

	stdinPipe, err := NewFilePipe()
	if err != nil {
		log.Fatal(err)
	}
	defer stdinPipe.Release()
	go func() {
		_, err := io.Copy(stdinPipe, os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		err = stdinPipe.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()
	progs := make([]*exec.Cmd, *nInstances)
	var wg sync.WaitGroup
	closeAfterWait := []io.Closer{}
	for i := range progs {
		cmd := exec.Command(binaryPath)
		w, err := cmd.StdinPipe()
		if err != nil {
			log.Fatal(err)
		}
		go func() {
			// We don't care about errors from the writer (we expect broken pipe if the process has exited
			// before reading all of its input), but we do care about errors when reading from the filepipe.
			if _, err := io.Copy(WrapWriter(w), stdinPipe.Reader()); err != nil {
				if _, ok := err.(WriterError); !ok {
					log.Fatal(err)
				}
			}
			w.Close()
		}()
		makeFromWrite := func(writeProc func(int, io.Reader) error, w io.Writer) io.Writer {
			if writeProc == nil {
				return w
			}
			pr, pw := io.Pipe()
			closeAfterWait = append(closeAfterWait, pw)
			i := i
			wg.Add(1)
			go func() {
				err := writeProc(i, pr)
				if err != nil {
					// All the errors we can get are not caused by instances' invalid behaviour, but
					// by system issues (can't create a file, broken pipe on real stdout/err, etc.)
					log.Fatal(err)
				}
				wg.Done()
			}()
			return pw
		}
		if *stdoutHandling == "contest" {
			cmd.Stdout = contestStdout.NewWriter(i)
		} else {
			cmd.Stdout = makeFromWrite(writeStdout, os.Stdout)
		}
		cmd.Stderr = makeFromWrite(writeStderr, os.Stderr)
		progs[i] = cmd
	}
	commLog := ioutil.Discard
	if *traceCommunications {
		commLog = os.Stderr
	}
	instances, err := RunInstances(progs, commLog)
	for _, f := range closeAfterWait {
		f.Close()
	}
	wg.Wait()
	if er, ok := err.(ErrRemainingMessages); ok {
		if *warnRemaining {
			m := make(map[int][]int)
			for _, p := range er.RemainingMessages {
				m[p.To] = append(m[p.To], p.From)
			}
			fmt.Fprintf(os.Stderr, "Warning: following instances had some messages left after they've terminated:\n")
			for dest, srcs := range m {
				fmt.Fprintf(os.Stderr, "Instance %d did not receive message from instances: ", dest)
				for _, src := range srcs {
					fmt.Fprintf(os.Stderr, "%d ", src)
				}
				fmt.Fprintln(os.Stderr)
			}
		}
		err = nil
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var maxTime time.Duration
	var lastInstance int
	for i, instance := range instances {
		if instanceTime := instance.TimeRunning + instance.TimeBlocked; instanceTime >= maxTime {
			maxTime = instanceTime
			lastInstance = i
		}
	}
	fmt.Fprintf(os.Stderr, "Duration: %v (longest running instance: %d)\n", maxTime, lastInstance)
	if *stats {
		w := tabwriter.NewWriter(os.Stderr, 2, 1, 1, ' ', 0)
		io.WriteString(w, "Instance\tTotal time\tCPU time\tTime spent waiting\tSent messages\tSent bytes\n")
		for i, instance := range instances {
			fmt.Fprintf(w, "%d\t%v\t%v\t%v\t%d\t%d\n", i, instance.TimeRunning+instance.TimeBlocked, instance.TimeRunning, instance.TimeBlocked, instance.MessagesSent, instance.MessageBytesSent)
		}
		w.Flush()
	}
}
