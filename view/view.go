package view

import (
	"github.com/dustin/go-humanize"
	"github.com/nsf/libtorgo/torrent"
	"text/tabwriter"
	"path/filepath"
	"strings"
	"bufio"
	"flag"
	"fmt"
	"os"
)

type view_mode int

const (
	view_basic view_mode = iota
	view_short
	view_long
)

// colors
var (
	color_red          = "\033[0;31m"
	color_red_bold     = "\033[1;31m"
	color_green        = "\033[0;32m"
	color_green_bold   = "\033[1;32m"
	color_yellow       = "\033[0;33m"
	color_yellow_bold  = "\033[1;33m"
	color_blue         = "\033[0;34m"
	color_blue_bold    = "\033[1;34m"
	color_magenta      = "\033[0;35m"
	color_magenta_bold = "\033[1;35m"
	color_cyan         = "\033[0;36m"
	color_cyan_bold    = "\033[1;36m"
	color_white        = "\033[0;37m"
	color_white_bold   = "\033[1;37m"
	color_none         = "\033[0m"
)

func clear_colors() {
	color_red          = ""
	color_red_bold     = ""
	color_green        = ""
	color_green_bold   = ""
	color_yellow       = ""
	color_yellow_bold  = ""
	color_blue         = ""
	color_blue_bold    = ""
	color_magenta      = ""
	color_magenta_bold = ""
	color_cyan         = ""
	color_cyan_bold    = ""
	color_white        = ""
	color_white_bold   = ""
	color_none         = ""
}

var tabstable = []string{
	"",
	"    ",
	"        ",
	"            ",
	"                ",
	"                    ",
	"                        ",
}

func tabs(n int) string {
	if n < 0 {
		return ""
	}

	if n < len(tabstable) {
		return tabstable[n]
	}

	return strings.Repeat("    ", n)
}

type view_tool_context struct {
	mode view_mode
	out func(...interface{})
	recursive bool
}

func (ctx *view_tool_context) error_file_or_dir(name string, err error) {
	ctx.out(color_white_bold, name, color_none, " (error: ", err, ")\n")
}

func (ctx *view_tool_context) show_short(filename string, mi *torrent.MetaInfo) {
	var name, length string
	var nfiles int
	switch info := mi.Info.(type) {
	case torrent.SingleFile:
		name = info.Name
		length = humanize.IBytes(uint64(info.Length))
		nfiles = 1
	case torrent.MultiFile:
		name = info.Name
		total_size := int64(0)
		for _, f := range info.Files {
			total_size += f.Length
		}
		length = humanize.IBytes(uint64(total_size))
		nfiles = len(info.Files)
	}

	ctx.out(color_white_bold, filename, color_none, "\n",
		"    ", color_yellow, name, color_none,
		" (", color_cyan, length, color_none, " in ", nfiles)
	if nfiles == 1 {
		ctx.out(" file)\n")
	} else {
		ctx.out(" files)\n")
	}
}

func (ctx *view_tool_context) show_basic(filename string, mi *torrent.MetaInfo) {
	// torrent file name
	ctx.out(color_white_bold, filename, color_none, "\n")

	var name string
	switch info := mi.Info.(type) {
	case torrent.SingleFile:
		name = info.Name
	case torrent.MultiFile:
		name = info.Name
	}

	// torrent name
	ctx.out(color_green_bold, "\tname\t", color_none,
		color_yellow, name, color_none, "\n")

	// tracker url
	ctx.out(color_green_bold, "\ttracker url\t", color_none,
		mi.AnnounceList[0][0], "\n")

	// created by
	ctx.out(color_green_bold, "\tcreated by\t", color_none,
		mi.CreatedBy, "\n")

	// created on
	ctx.out(color_green_bold, "\tcreated on\t", color_none,
		color_magenta, mi.CreationDate, color_none, "\n")

	switch info := mi.Info.(type) {
	case torrent.SingleFile:
		ctx.out(color_green_bold, "\tfile name\t", color_none,
			info.Name, "\n")
		ctx.out(color_green_bold, "\tfile size\t", color_none,
			color_cyan, humanize.IBytes(uint64(info.Length)), color_none, "\n")
	case torrent.MultiFile:
		total_size := int64(0)
		for _, f := range info.Files {
			total_size += f.Length
		}
		ctx.out(color_green_bold, "\tnum files\t", color_none,
			len(info.Files), "\n")
		ctx.out(color_green_bold, "\ttotal size\t", color_none,
			color_cyan, humanize.IBytes(uint64(total_size)), color_none, "\n")
	}

	ctx.out("\n")
}

func (ctx *view_tool_context) show_long(filename string, mi *torrent.MetaInfo) {
	// torrent file name
	ctx.out(color_white_bold, filename, color_none, "\n")

	// announce groups
	ctx.out(color_green_bold, tabs(1), "announce groups", color_none, "\n")
	for i, ag := range mi.AnnounceList {
		ctx.out(color_yellow_bold, tabs(2), i, color_none, "\n")
		for _, a := range ag {
			ctx.out(tabs(3), a, "\n")
		}
	}

	// created on
	ctx.out(color_green_bold, tabs(1), "created on", color_none, "\n",
		tabs(2), mi.CreationDate, "\n")

	// comment
	if mi.Comment != "" {
		ctx.out(color_green_bold, tabs(1), "comment", color_none, "\n",
			tabs(2), mi.Comment, "\n")
	}

	// created by
	if mi.CreatedBy != "" {
		ctx.out(color_green_bold, tabs(1), "created by", color_none, "\n",
			tabs(2), mi.CreatedBy, "\n")
	}

	// encoding
	if mi.Encoding != "" {
		ctx.out(color_green_bold, tabs(1), "encoding", color_none, "\n",
			tabs(2), mi.Encoding, "\n")
	}

	// url list
	if len(mi.URLList) > 0 {
		ctx.out(color_green_bold, tabs(1), "webseed urls", color_none, "\n")
		for _, url := range mi.URLList {
			ctx.out(tabs(2), url, "\n")
		}
	}

	switch info := mi.Info.(type) {
	case torrent.SingleFile:
		ctx.out(color_green_bold, tabs(1), "name", color_none, " (single file)\n",
			tabs(2), color_yellow, info.Name, color_none, "\n")
		ctx.out(color_green_bold, tabs(1), "length", color_none, "\n",
			tabs(2), color_cyan, humanize.IBytes(uint64(info.Length)), color_none, "\n")
	case torrent.MultiFile:
		ctx.out(color_green_bold, tabs(1), "name", color_none, " (multiple files)\n",
			tabs(2), color_yellow, info.Name, color_none, "\n")
		ctx.out(color_green_bold, tabs(1), "files", color_none, " (", len(info.Files), ")\n")
		for _, f := range info.Files {
			ctx.out(tabs(2), filepath.Join(f.Path...),
				" (", color_cyan, humanize.IBytes(uint64(f.Length)), color_none, ")\n")
		}
	}

	ctx.out("\n")
}

func (ctx *view_tool_context) show_file(filename string) {
	mi, err := torrent.LoadFromFile(filename)
	if err != nil {
		ctx.error_file_or_dir(filename, err)
		return
	}

	if !ctx.recursive {
		_, filename = filepath.Split(filename)
	}
	switch ctx.mode {
	case view_short:
		ctx.show_short(filename, mi)
	case view_long:
		ctx.show_long(filename, mi)
	case view_basic:
		ctx.show_basic(filename, mi)
	}
}

func (ctx *view_tool_context) show_dir(dirname string) {
	walker := func(path string, info os.FileInfo, err error) error {
		// skip directories in a non-recursive mode
		if info.IsDir() && !ctx.recursive && path != dirname {
			return filepath.SkipDir
		}

		// skip bad files
		if err != nil {
			return nil
		}

		if filepath.Ext(path) == ".torrent" {
			ctx.show_file(path)
		}
		return nil
	}
	filepath.Walk(dirname, walker)
}

func Tool() {
	var ctx view_tool_context
	var (
		no_colors bool
		short bool
		basic bool
		long bool
	)

	fs := flag.NewFlagSet("view tool", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s view [<options>] [<file or dir...>]\n\n",
			os.Args[0])
		fmt.Fprintln(os.Stderr, "Options:")
		tabber := tabwriter.NewWriter(os.Stderr, 0, 0, 4, ' ', 0)
		fs.VisitAll(func (f *flag.Flag) {
			fmt.Fprintf(tabber, "  -%s\t%s\n", f.Name, f.Usage)
		})
		tabber.Flush()
	}

	fs.BoolVar(&ctx.recursive, "r", false, "inspect directories recursively")
	fs.BoolVar(&no_colors, "n", false,
		"don't use terminal colors")
	fs.BoolVar(&short, "s", false,
		"short output, two lines per file, brief info")
	fs.BoolVar(&basic, "b", true,
		"basic output, a couple of lines per file (default)")
	fs.BoolVar(&long, "l", false,
		"long output, prints every bit of information")
	fs.Parse(os.Args[2:])

	if no_colors {
		clear_colors()
	}

	switch {
	case short:
		tabber := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
		defer tabber.Flush()
		ctx.out = func(args ...interface{}) {
			fmt.Fprint(tabber, args...)
		}
		ctx.mode = view_short
	case long:
		stdout := bufio.NewWriter(os.Stdout)
		defer stdout.Flush()
		ctx.out = func(args ...interface{}) {
			fmt.Fprint(stdout, args...)
		}
		ctx.mode = view_long
	default:
		tabber := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
		defer tabber.Flush()
		ctx.out = func(args ...interface{}) {
			fmt.Fprint(tabber, args...)
		}
		ctx.mode = view_basic
	}

	if fs.NArg() == 0 {
		ctx.show_dir(".")
	}

	for _, arg := range fs.Args() {
		fi, err := os.Stat(arg)
		if err != nil {
			ctx.error_file_or_dir(arg, err)
			continue
		}

		if fi.IsDir() {
			ctx.show_dir(arg)
		} else {
			ctx.show_file(arg)
		}
	}
}
