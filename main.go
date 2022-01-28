package main

/*
#cgo LDFLAGS: -L${SRCDIR}/graftcp -lgraftcp

#include <stdlib.h>

static void *alloc_string_slice(int len) {
             return malloc(sizeof(char*)*len);
}

int client_main(int argc, char **argv);
*/
import "C"
import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/hmgle/graftcp/local"
)

const (
	maxArgsLen = 0xfff
	cmdLead    = "run"
	helpDoc    = "Example: viaproxy socks5://127.0.0.1:1080 run curl -L https://www.google.com\n"
)

var proxyPattern = regexp.MustCompile(`^(?P<proto>.+)(\s+|://)(?P<addr>.+:\d+)$`)

type Options struct {
	Use string `long:"use" description:"e.g. socks5://127.0.0.1:1080" required:"true"`
}

func reSubMatchMap(r *regexp.Regexp, str string) (map[string]string, error) {
	var err error = nil
	match := r.FindStringSubmatch(str)
	if match == nil {
		return nil, fmt.Errorf("invalid proxy spec: '%s'", str)
	}
	subMatchMap := make(map[string]string)
	for i, name := range r.SubexpNames() {
		if i != 0 {
			subMatchMap[name] = match[i]
		}
	}

	return subMatchMap, err
}

func graftcpClientMain(args []string) int {
	argc := C.int(len(args))

	argv := (*[maxArgsLen]*C.char)(C.alloc_string_slice(argc))
	defer C.free(unsafe.Pointer(argv))

	for i, arg := range args {
		argv[i] = C.CString(arg)
		defer C.free(unsafe.Pointer(argv[i]))
	}

	returnValue := C.client_main(argc, (**C.char)(unsafe.Pointer(argv)))
	return int(returnValue)
}

func showHelpAndExit() {
	fmt.Printf(helpDoc)
	os.Exit(0)
}

func main() {
	var err error
	var proxySpec string = ""
	var targetCmd []string
	var cmdFollowing = false
	verbose := false

	proxyMode := "auto"
	socks5Addr := ""
	socks5User := ""
	socks5Pwd := ""
	httpProxyAddr := ""

	retCode := 0
	defer func() { os.Exit(retCode) }()

	if len(os.Args) == 1 {
		showHelpAndExit()
	}

	for i, v := range os.Args {
		if i == 0 {
			continue
		}

		if len(targetCmd) == 0 {
			if v == "-h" || v == "--help" {
				showHelpAndExit()
			}
			if v == "-v" || v == "--verbose" {
				verbose = true
				continue
			}
		}

		if v == cmdLead && len(targetCmd) == 0 {
			cmdFollowing = true
			continue
		}

		if cmdFollowing {
			targetCmd = append(targetCmd, v)
		} else {
			proxySpec = v
		}
	}

	if proxySpec == "" {
		fmt.Printf("No proxy specified.\n")
		os.Exit(1)
	}
	if len(targetCmd) == 0 {
		fmt.Printf("No command part found.\n")
		os.Exit(1)
	}

	m, err := reSubMatchMap(proxyPattern, proxySpec)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf(
			"Using %s proxy '%s' to run '%s'.\n",
			m["proto"], m["addr"], strings.Join(targetCmd, " "),
		)
	}

	if m["proto"] == "socks5" {
		proxyMode = "only_socks5"
		socks5Addr = m["addr"]
	} else if m["proto"] == "http" {
		proxyMode = "only_http_proxy"
		httpProxyAddr = m["addr"]
	}

	l := local.NewLocal(":0", socks5Addr, socks5User, socks5Pwd, httpProxyAddr)
	l.SetSelectMode(proxyMode)

	tmpDir, err := ioutil.TempDir("/tmp", "mgraftcp")
	if err != nil {
		log.Fatalf("ioutil.TempDir err: %s", err.Error())
	}
	defer os.RemoveAll(tmpDir)
	pipePath := tmpDir + "/mgraftcp.fifo"
	syscall.Mkfifo(pipePath, uint32(os.ModePerm))

	l.FifoFd, err = os.OpenFile(pipePath, os.O_RDWR, 0)
	if err != nil {
		log.Fatalf("os.OpenFile(%s) err: %s", pipePath, err.Error())
	}

	go l.UpdateProcessAddrInfo()
	ln, err := l.StartListen()
	if err != nil {
		log.Fatalf("l.StartListen err: %s", err.Error())
	}
	go l.StartService(ln)
	defer ln.Close()

	_, faddr := l.GetFAddr()

	var graftcpClientArgs []string
	graftcpClientArgs = append(graftcpClientArgs, os.Args[0])
	graftcpClientArgs = append(graftcpClientArgs, "-p", strconv.Itoa(faddr.Port), "-f", pipePath)
	graftcpClientArgs = append(graftcpClientArgs, targetCmd...)
	if verbose {
		fmt.Printf("graftcp client main args: %s\n", strings.Join(graftcpClientArgs, " "))
	}
	retCode = graftcpClientMain(graftcpClientArgs)
}
