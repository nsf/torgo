package make

import (
	"os"
	"fmt"
	"flag"
	"time"
	"bytes"
	"bufio"
	"strings"
	"runtime"
	"path/filepath"
	"text/tabwriter"
	"github.com/nsf/libtorgo/torrent"
	"github.com/dustin/go-humanize"
)

//----------------------------------------------------------------------------
// sampler (for average speed measurements)
//----------------------------------------------------------------------------

type sampler []int64

func (sp *sampler) add_sample(sample int64) {
	s := *sp
	if len(s) < cap(s) {
		s = append(s, sample)
	} else {
		n := len(s) - 1
		copy(s[:n], s[1:len(s)])
		s[n] = sample
	}
	*sp = s
}

func (sp *sampler) average() int64 {
	if len(*sp) == 0 {
		return 0
	}

	sum := int64(0)
	for _, s := range *sp {
		sum += s
	}
	return sum / int64(len(*sp))
}

//----------------------------------------------------------------------------
// progress reporter
//----------------------------------------------------------------------------

type progress_reporter interface {
	begin()
	report(done, total int64)
	end()
}

//----------------------------------------------------------------------------
// simple progress reporter
//----------------------------------------------------------------------------

type simple_progress_reporter struct {
	lastipercents int
}

func (this *simple_progress_reporter) begin() {
	this.lastipercents = 0
	fmt.Print("Hashing contents:   0%")
}

func (this *simple_progress_reporter) report(done, total int64) {
	percents := float64(done) / float64(total) * 100
	ipercents := int(percents)
	if ipercents == this.lastipercents {
		return
	}
	this.lastipercents = ipercents
	fmt.Printf("\rHashing contents: %3d%%", ipercents)
}

func (this *simple_progress_reporter) end() {
	fmt.Print("\rHashing contents: 100%")
	fmt.Print("\rHashing contents: Done.\n")
}

//----------------------------------------------------------------------------
// advanced progress reporter
//----------------------------------------------------------------------------

type advanced_progress_reporter struct {
	sampler sampler
	start_time time.Time
	last_time time.Time
	last_done int64
	out *bufio.Writer
	total int64
}

func (this *advanced_progress_reporter) begin() {
	this.sampler = make(sampler, 0, 5)
	this.last_time = time.Now()
	this.start_time = this.last_time
	this.last_done = 0
	this.out = bufio.NewWriter(os.Stdout)
}

// 1000KiB  1000MiB/s 00:00:00 [#################------------------]  51%
func (this *advanced_progress_reporter) report(done, total int64) {
	this.total = total

	now := time.Now()
	delta_time := now.Sub(this.last_time)
	this.last_time = now

	time_since_start := now.Sub(this.start_time)

	delta_done := done - this.last_done
	this.last_done = done

	speed := int64(float64(delta_done) / delta_time.Seconds())
	this.sampler.add_sample(speed)

	done_h := humanize.IBytes(uint64(done))

	w := get_terminal_width()

	// reserve one space in the beginning
	w -= 1
	this.out.WriteString(" ")

	// reserve space for bytes progress counter:
	// 1000KiB (7) + ' ' (1)
	w -= 8
	fmt.Fprintf(this.out, "%7s ", done_h)

	// reserve space for speed value: 1000KiB/s + ' ' (10)
	w -= 10
	fmt.Fprintf(this.out, "%7s/s ",
		humanize.IBytes(uint64(this.sampler.average())))

	// reserve space for time since start: 00:00:00 (8) + ' ' (1)
	w -= 9
	var t time.Time
	t = t.Add(time_since_start)
	fmt.Fprint(this.out, t.Format("15:04:05 "))

	// and now the progress bar, reserve space for [] (2) and for ' ' + 100%
	// (5) == 7
	w -= 7
	this.out.WriteString("[")
	donecharw := int(float64(done) / float64(total) * float64(w))
	for i := 0; i < donecharw; i++ {
		this.out.WriteByte('#')
	}
	for i := 0; i < w - donecharw; i++ {
		this.out.WriteByte('-')
	}
	this.out.WriteString("] ")
	fmt.Fprintf(this.out, "%3d%%\r",
		int(float64(done) / float64(total) * 100))
	this.out.Flush()
}

func (this *advanced_progress_reporter) end() {
	total_h := humanize.IBytes(uint64(this.total))
	time_since_start := time.Since(this.start_time)

	w := get_terminal_width()
	// reserve one space in the beginning
	w -= 1
	this.out.WriteString(" ")

	// reserve space for bytes progress counter:
	// 1000KiB (7) + ' ' (1)
	w -= 8
	fmt.Fprintf(this.out, "%7s ", total_h)

	// reserve space for speed value: 1000KiB/s + ' ' (10)
	w -= 10
	fmt.Fprintf(this.out, "%7s/s ",
		humanize.IBytes(uint64(this.sampler.average())))

	// reserve space for time since start: 00:00:00 (8) + ' ' (1)
	w -= 9
	var t time.Time
	t = t.Add(time_since_start)
	fmt.Fprint(this.out, t.Format("15:04:05 "))

	// and now the progress bar, reserve space for [] (2) and for ' ' + 100%
	// (5) == 7
	w -= 7
	this.out.WriteString("[")
	for i := 0; i < w; i++ {
		this.out.WriteByte('=')
	}
	this.out.WriteString("] 100%\n")
	this.out.Flush()
}

//----------------------------------------------------------------------------

type announce_groups [][]string

func (ag *announce_groups) String() string {
	var buf bytes.Buffer
	fmt.Fprint(&buf, "[")
	for i, g := range *ag {
		if i != 0 {
			fmt.Fprint(&buf, ", ")
		}
		fmt.Fprint(&buf, "[", strings.Join(g, ", "), "]")
	}
	fmt.Fprint(&buf, "]")
	return buf.String()
}

func (ag *announce_groups) Set(value string) error {
	// ignore empty strings
	if value == "" {
		return nil
	}

	// don't care about empty strings, the torrent.Builder will clean them up
	group := strings.Split(value, ",")
	*ag = append(*ag, group)
	return nil
}

type webseeds []string

func (ws *webseeds) String() string {
	return "[" + strings.Join(*ws, ", ") + "]"
}

func (ws *webseeds) Set(value string) error {
	if value == "" {
		return nil
	}
	newseeds := strings.Split(value, ",")
	*ws = append(*ws, newseeds...)
	return nil
}

func Tool() {
	var (
		announce_groups announce_groups
		comment string
		piece_length int64
		name string
		output string
		private bool
		nworkers int
		verbose bool
		webseeds webseeds
	)

	fs := flag.NewFlagSet("make tool", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr,
			"Usage: %s make -a <url>[,<url>] [<options>] <file or dir...>\n\n",
			os.Args[0])
		fmt.Fprintln(os.Stderr, "Options:")
		tabber := tabwriter.NewWriter(os.Stderr, 0, 0, 2, ' ', 0)
		fs.VisitAll(func (f *flag.Flag) {
			fmt.Fprintf(tabber, "  -%s\t%s\n", f.Name, f.Usage)
		})
		tabber.Flush()
	}

	fs.Var(&announce_groups, "a",
		"announce URL group (comma separated), additional -a adds backup trackers")
	fs.StringVar(&comment, "c", "",
		"add commentary to the metainfo")
	fs.Int64Var(&piece_length, "l", 256*1024,
		"piece length (default is 256KiB)")
	fs.StringVar(&name, "n", "",
		"name of the torrent (default is automatically determined)")
	fs.StringVar(&output, "o", "",
		"filename of the created .torrent file (default is <name>.torrent)")
	fs.BoolVar(&private, "p", false,
		"set the private flag")
	fs.IntVar(&nworkers, "j", 0,
		fmt.Sprintf("use N workers for SHA1 hashing (default is %d)",
			runtime.NumCPU()))
	fs.BoolVar(&verbose, "v", false,
		"be verbose")
	fs.Var(&webseeds, "w",
		"add WebSeed URLs (comma separated), additional -w adds more URLs")

	fs.Parse(os.Args[2:])

	// check some mandatory arguments
	if len(announce_groups) == 0 {
		fmt.Fprintln(os.Stderr, "You must specify at least one announce URL group")
		os.Exit(1)
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "You must specify at least one input file")
		os.Exit(1)
	}

	// set default nworkers if it's <= 0
	if nworkers <= 0 {
		nworkers = runtime.NumCPU()
	}

	// override GOMAXPROCS only if user doesn't wish to do that himself
	if os.Getenv("GOMAXPROCS") == "" {
		runtime.GOMAXPROCS(nworkers+1)
	}

	// prepare builder for building the .torrent file
	var b torrent.Builder
	for _, ag := range announce_groups {
		b.AddAnnounceGroup(ag)
	}
	b.SetComment(comment)
	b.SetPieceLength(piece_length)
	b.SetName(name)
	b.SetPrivate(private)
	for _, ws := range webseeds {
		b.AddWebSeedURL(ws)
	}

	// add files to the builder queue
	walker := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading \"%s\": %s\n", path, err)
			return nil
		}

		b.AddFile(path)
		return nil
	}
	for _, arg := range fs.Args() {
		fi, err := os.Stat(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading \"%s\": %s\n", arg, err)
			continue
		}

		if fi.IsDir() {
			filepath.Walk(arg, walker)
		} else {
			b.AddFile(arg)
		}
	}

	// get terminal width and select progress reporter
	var reporter progress_reporter
	termw := get_terminal_width()
	if termw < 40 {
		reporter = &simple_progress_reporter{}
	} else {
		reporter = &advanced_progress_reporter{}
	}

	// create the batch
	batch, err := b.Submit()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating a batch: %s\n", err)
		os.Exit(1)
	}

	// prepare file for .torrent output
	if output == "" {
		if name == "" {
			output = batch.DefaultName() + ".torrent"
		} else {
			output = name + ".torrent"
		}
	}
	f, err := os.Create(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating a file: %s\n", err)
		os.Exit(1)
	}
	defer f.Close()

	// START!
	reporter.begin()
	completion, status := batch.Start(f, nworkers)
	for {
		select {
		case hashed := <-status:
			reporter.report(hashed, batch.TotalSize())
			time.Sleep(250 * time.Millisecond)
		case err := <-completion:
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nerror: %s\n", err)
				os.Exit(1)
			}
			reporter.end()
			return
		}
	}
}