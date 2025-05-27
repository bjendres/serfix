package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"runtime"
)

const (
	helpFlagUsage     = "Help and usage instructions"
	forceFlagUsage    = "Force overwrite of destination file if it exists"
	readBufferDefault = 16 // 16M buffer, can be overwritten by --buffer-size 
)

var helpPtr = flag.Bool("help", false, helpFlagUsage)
var forcePtr = flag.Bool("force", false, forceFlagUsage)
var counter int = 0
var lexer = regexp.MustCompile(`s:\d+:\\?\".*?\\?\";`)
var re = regexp.MustCompile(`(s:)(\d+)(:\\?\")(.*?)(\\?\";)`)
var esc = regexp.MustCompile(`(\\"|\\'|\\\\|\\a|\\b|\\f|\\n|\\r|\\s|\\t|\\v|\\0)`)
var buffer_size_in_mb = readBufferDefault;

func init() {
	// Short flags too
	flag.BoolVar(helpPtr, "h", false, helpFlagUsage)
	flag.BoolVar(forcePtr, "f", false, forceFlagUsage)
	flag.StringVar(&bufferSize, "--buffer-size", '16M', forceFlagUsage)

}

func main() {
	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU)

	// APPLY read buffer size override via command line 'buffer-size' parameter 
	var buffer_size_prefix = "--buffer-size="
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, buffer_size_prefix) {
			// this is buffer_size override parameter
			if strings.HasSuffix(strings.ToLower(arg), "m") {
				buffer_size_in_mb, error := strconv.ParseInt(arg[:len(arg)], 10, 32)
				int_parameter, error := strconv.Atoi(arg[:len(arg)-1])
				if error != nil {
    				fmt.Println("Failed to parse buffer-size value:", error)
					return
				}
				buffer_size_in_mb = int_parameter
     		} else {
     			panic("The --buffer-size paramter requires a number followed by an 'M'")
     		}
		}
	}


	// Handle flags
	flag.Parse()

	args := flag.Args()

	if *helpPtr {
		PrintUsage()
		return
	}

	if len(args) > 0 {
		filename := fmt.Sprintf("%s", args[0])
		// Open provided file
		infile, err := os.Open(filename)
		if err != nil {
			fmt.Println(err)
			return
		}
		// close provided file on exit and check for its returned error
		defer infile.Close()

		newDestination := false
		outfilename := filename
		tempfilename := fmt.Sprintf("%s~", outfilename)
		if len(args) > 1 {
			newDestination = true
			outfilename = fmt.Sprintf("%s", args[1])
			tempfilename = fmt.Sprintf("%s~", outfilename)
			if !*forcePtr {
				if _, err := os.Stat(outfilename); err == nil {
					fmt.Println("Destination file already exists, aborting serfix.")
					return
				}
			}
		}

		// Open out file
		tempfile, err := os.Create(tempfilename)
		if err != nil {
			println(err)
		}
		// close out file
		defer tempfile.Close()

		r := bufio.NewReaderSize(infile, 1024 * 1024 * buffer_size_in_mb)

		line, err := r.ReadString('\n')
		for err == nil {
			tempfile.WriteString(lexer.ReplaceAllStringFunc(string(line), Replace))

			line, err = r.ReadString('\n')
		}
		if err != io.EOF {
			fmt.Println(err)
			return
		}

		// Close the in/out files
		if err := tempfile.Close(); err != nil {
			fmt.Println(err)
			return
		}
		if err := infile.Close(); err != nil {
			fmt.Println(err)
			return
		}

		if !newDestination {
			// Remove original file
			if err := os.Remove(filename); err != nil {
				fmt.Println(err)
				return
			}
		} else {
			// If destination exists and force flag is used, remove destination file
			if _, err := os.Stat(outfilename); err == nil {
				if !*forcePtr {
					if err := os.Remove(outfilename); err != nil {
						fmt.Println(err)
						return
					}
				}
			}
		}
		if err := os.Rename(tempfilename, outfilename); err != nil {
			fmt.Println(err)
			return
		}

	} else {
		r := bufio.NewReaderSize(os.Stdin, 1024 * 1024 * buffer_size_in_mb)

		line, isPrefix, err := r.ReadLine()
		for err == nil && !isPrefix {
			fmt.Println(lexer.ReplaceAllStringFunc(string(line), Replace))

			line, isPrefix, err = r.ReadLine()
		}
		if isPrefix {
			fmt.Println(errors.New("serfix: buffer size too small"))
			return
		}
		if err != io.EOF {
			fmt.Println(err)
			return
		}
	}
}

func Replace(matches string) string {
	parts := re.FindStringSubmatch(matches)
	str_len := len(parts[4]) - len(esc.FindAllString(parts[4], -1))
	return fmt.Sprintf("%s%d%s%s%s", parts[1], str_len, parts[3], parts[4], parts[5])
}

func PrintUsage() {
	fmt.Println("Usage: serfix [flags] filename [outfilename]")
	fmt.Println("Alt. Usage: cat filename | serfix")
	fmt.Println("")
	fmt.Println("\t -f, --force   \t\t\t Force overwrite of destination file if it exists.")
	fmt.Println("\t -h, --help    \t\t\t Print serfix help.")
	fmt.Println("\t --buffer-size \t\t\t adjust the serfix buffer size, default is 16M")
	fmt.Println("")
}
