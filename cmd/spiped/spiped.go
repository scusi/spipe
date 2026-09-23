// spiped - a go implementation of the original spiped command from
// https://github.com/Tarsnap/spiped (see spiped(1)), using the spipe
// protocol library from https://github.com/dchest/spipe
//
// Command line compatible with the original spiped:
// spiped {-e | -d} -s <source socket> -t <target socket> -k <key file> [-F]
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"log/syslog"
	"math"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/dchest/spipe"
)

const version = "spiped 1.0 (scusi/spipe go implementation, command line compatible with spiped 1.6.x)"

// childEnv marks the re-executed daemon child process (used for daemonizing).
const childEnv = "SPIPED_DAEMON_CHILD"

var (
	flagEncrypt      bool
	flagDecrypt      bool
	sourceSock       string
	targetSock       string
	keyFile          string
	bindAddr         string
	flagWaitDNS      bool
	flagFast         bool
	flagRequirePFS   bool
	foreground       bool
	disableKeepAlive bool
	maxConns         int
	timeoutSpec      string
	pidFile          string
	rtimeSpec        string
	noReResolve      bool
	useSyslog        bool
	userSpec         string
	showVersion      bool
)

var (
	key         []byte
	connTimeout time.Duration
	sem         chan struct{}
	wg          sync.WaitGroup
)

// maxBytesPerSession is the maximum of bytes to be allowed to be transfered per session.
// otherwise security considerations will not hold true.
// See: https://github.com/Tarsnap/spiped/blob/master/DESIGN.md
var maxBytesPerSession = math.Pow(float64(2), float64(64))

func init() {
	flag.BoolVar(&flagEncrypt, "e", false, "take unencrypted connections from the source socket and send encrypted connections to the target socket")
	flag.BoolVar(&flagDecrypt, "d", false, "take encrypted connections from the source socket and send unencrypted connections to the target socket")
	flag.StringVar(&sourceSock, "s", "", "address to listen on for incoming connections (host.name:port, [ip.v4.ad.dr]:port, [ipv6::addr]:port or /absolute/path/to/unix/socket)")
	flag.StringVar(&targetSock, "t", "", "address to connect to (host.name:port, [ip.v4.ad.dr]:port, [ipv6::addr]:port or /absolute/path/to/unix/socket)")
	flag.StringVar(&keyFile, "k", "", "file to read the shared key from, use '-' to read from standard input")
	flag.StringVar(&bindAddr, "b", "", "bind the outgoing address (port number optional, left to the operating system if omitted)")
	flag.BoolVar(&flagWaitDNS, "D", false, "wait for DNS: keep retrying if resolving the source address or binding fails")
	flag.BoolVar(&flagFast, "f", false, "use fast/weak handshaking (not supported by the spipe library, accepted but ignored)")
	flag.BoolVar(&flagRequirePFS, "g", false, "require perfect forward secrecy (not supported by the spipe library, accepted but ignored)")
	flag.BoolVar(&foreground, "F", false, "run in the foreground, do not daemonize")
	flag.BoolVar(&disableKeepAlive, "j", false, "disable transport layer keep-alives")
	flag.IntVar(&maxConns, "n", 100, "limit on the number of simultaneous connections (0 means no limit)")
	flag.StringVar(&timeoutSpec, "o", "5", "connection timeout in seconds for connecting to the target and protocol handshakes (0 means no timeout)")
	flag.StringVar(&pidFile, "p", "", "file to which the process id should be written (default: source socket with .pid appended, only when daemonizing)")
	flag.StringVar(&rtimeSpec, "r", "60", "re-resolve the target address every rtime seconds (addresses are re-resolved on every outgoing connection)")
	flag.BoolVar(&noReResolve, "R", false, "disable target address re-resolution (resolve the target once at start)")
	flag.BoolVar(&useSyslog, "syslog", false, "send warnings to syslog instead of stderr (no effect if -F is used)")
	flag.StringVar(&userSpec, "u", "", "after binding the source socket, change to user, :group or user:group")
	flag.BoolVar(&showVersion, "v", false, "print the version number and exit")
	flag.Usage = usage
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  %s {-e | -d} -s <source socket> -t <target socket> -k <key file>\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "        [-DFj] [-b <bind address>] [-n <max # connections>]\n")
	fmt.Fprintf(os.Stderr, "        [-o <connection timeout>] [-p <pidfile>] [-r <rtime> | -R]\n")
	fmt.Fprintf(os.Stderr, "        [--syslog] [-u <user> | <:group> | <user:group>]\n")
	fmt.Fprintf(os.Stderr, "  %s -v\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Socket addresses must be in one of the following formats:\n")
	fmt.Fprintf(os.Stderr, "  /absolute/path/to/unix/socket\n")
	fmt.Fprintf(os.Stderr, "  host.name:port\n")
	fmt.Fprintf(os.Stderr, "  [ip.v4.ad.dr]:port\n")
	fmt.Fprintf(os.Stderr, "  [ipv6::addr]:port\n\n")
	flag.PrintDefaults()
}

func main() {
	flag.Parse()

	if showVersion {
		fmt.Println(version)
		return
	}

	if flagEncrypt == flagDecrypt {
		fmt.Fprintln(os.Stderr, "spiped: exactly one of -e or -d must be given")
		flag.Usage()
		os.Exit(1)
	}
	if sourceSock == "" || targetSock == "" || keyFile == "" {
		fmt.Fprintln(os.Stderr, "spiped: -s, -t and -k are required")
		flag.Usage()
		os.Exit(1)
	}
	if flagFast {
		log.Println("spiped: warning: -f is not supported by the spipe library and will be ignored")
	}
	if flagRequirePFS {
		log.Println("spiped: warning: -g is not supported by the spipe library and will be ignored")
	}

	var err error
	isChild := os.Getenv(childEnv) == "1"
	connTimeout, err = parseSeconds(timeoutSpec)
	if err != nil {
		die(err)
	}
	if _, err = parseSeconds(rtimeSpec); err != nil {
		die(err)
	}

	if isChild && keyFile == "-" {
		// The daemonized child cannot re-read the key from stdin,
		// the parent passes it through a pipe (fd 4).
		key, err = io.ReadAll(os.NewFile(4, "keypipe"))
	} else {
		key, err = readKey(keyFile)
	}
	if err != nil {
		die(err)
	}

	srcNetwork, srcAddr, err := parseSockAddr(sourceSock)
	if err != nil {
		die(err)
	}
	tgtNetwork, tgtAddr, err := parseSockAddr(targetSock)
	if err != nil {
		die(err)
	}

	// Bind the source socket before daemonizing, so the parent process only
	// returns once spiped is ready to accept connections (like the original).
	var ln net.Listener
	if isChild {
		f := os.NewFile(3, "listener")
		ln, err = net.FileListener(f)
		if err != nil {
			die(err)
		}
	} else {
		ln, err = listenWithRetry(srcNetwork, srcAddr, flagWaitDNS)
		if err != nil {
			die(err)
		}
	}

	if !foreground && !isChild {
		if err = daemonize(ln, key); err != nil {
			die(err)
		}
	}

	if !foreground {
		if pidFile == "" {
			pidFile = defaultPidFile(sourceSock)
		}
		if err = writePidFile(pidFile); err != nil {
			die(err)
		}
		if useSyslog {
			switchToSyslog()
		}
	}

	if userSpec != "" {
		if err = dropPrivileges(userSpec); err != nil {
			die(err)
		}
	}

	dialer := &net.Dialer{Timeout: connTimeout}
	if disableKeepAlive {
		dialer.KeepAlive = -1
	}
	if tgtNetwork == "tcp" && bindAddr != "" {
		laddr, err := parseBindAddr(bindAddr)
		if err != nil {
			die(err)
		}
		dialer.LocalAddr = laddr
	}

	target := tgtAddr
	if noReResolve && tgtNetwork == "tcp" {
		target, err = resolveTargetWithRetry(tgtAddr, flagWaitDNS)
		if err != nil {
			die(err)
		}
	}

	if maxConns > 0 {
		sem = make(chan struct{}, maxConns)
	}

	// On SIGTERM stop accepting new connections and exit once all active
	// connections are finished.
	done := make(chan struct{})
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	go func() {
		<-sigterm
		log.Println("spiped: received SIGTERM, waiting for connections to finish")
		close(done)
		ln.Close()
	}()

	log.Printf("spiped: forwarding from %s to %s", srcAddr, target)

acceptLoop:
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-done:
				break acceptLoop
			default:
			}
			log.Printf("spiped: accept error: %v", err)
			continue
		}
		// Stop accepting further connections while the limit is reached.
		if sem != nil {
			sem <- struct{}{}
		}
		wg.Add(1)
		go handleConnection(conn, dialer, tgtNetwork, target)
	}
	wg.Wait()
	log.Println("spiped: shut down")
}

func handleConnection(src net.Conn, dialer *net.Dialer, network, target string) {
	defer func() {
		wg.Done()
		if sem != nil {
			<-sem
		}
	}()
	defer src.Close()

	if tc, ok := src.(*net.TCPConn); ok && disableKeepAlive {
		tc.SetKeepAlive(false)
	}

	if flagDecrypt {
		// Source socket speaks spipe, wrap and handshake first.
		wrapped, err := spipeHandshake(src, key, false, connTimeout)
		if err != nil {
			log.Printf("spiped: handshake with %s failed: %v", src.RemoteAddr(), err)
			return
		}
		src = wrapped
	}

	var dst net.Conn
	if flagEncrypt {
		// Target socket speaks spipe, wrap and handshake after connecting.
		raw, err := dialer.Dial(network, target)
		if err != nil {
			log.Printf("spiped: cannot connect to target %s: %v", target, err)
			return
		}
		wrapped, err := spipeHandshake(raw, key, true, connTimeout)
		if err != nil {
			raw.Close()
			log.Printf("spiped: handshake with target %s failed: %v", target, err)
			return
		}
		dst = wrapped
	} else {
		var err error
		dst, err = dialer.Dial(network, target)
		if err != nil {
			log.Printf("spiped: cannot connect to target %s: %v", target, err)
			return
		}
	}
	defer dst.Close()

	log.Printf("spiped: connection from %s to %s", src.RemoteAddr(), target)
	tcp_con_forward(src, dst)
}

// spipeHandshake wraps a plain connection in an spipe connection and performs
// the protocol handshake, enforcing the connection timeout if one is set.
func spipeHandshake(raw net.Conn, key []byte, isClient bool, timeout time.Duration) (net.Conn, error) {
	var c *spipe.Conn
	if isClient {
		c = spipe.Client(key, raw)
	} else {
		c = spipe.Server(key, raw)
	}
	if timeout > 0 {
		raw.SetDeadline(time.Now().Add(timeout))
	}
	err := c.Handshake()
	if timeout > 0 {
		raw.SetDeadline(time.Time{})
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func listenWithRetry(network, addr string, wait bool) (net.Listener, error) {
	for {
		ln, err := net.Listen(network, addr)
		if err == nil {
			return ln, nil
		}
		if !wait {
			return nil, err
		}
		log.Printf("spiped: cannot bind %s: %v, retrying", addr, err)
		time.Sleep(time.Second)
	}
}

func resolveTargetWithRetry(addr string, wait bool) (string, error) {
	for {
		ta, err := net.ResolveTCPAddr("tcp", addr)
		if err == nil {
			return ta.String(), nil
		}
		if !wait {
			return "", err
		}
		log.Printf("spiped: cannot resolve %s: %v, retrying", addr, err)
		time.Sleep(time.Second)
	}
}

func parseSockAddr(addr string) (network, normAddr string, err error) {
	switch {
	case addr == "":
		return "", "", fmt.Errorf("empty socket address")
	case strings.HasPrefix(addr, "/"):
		return "unix", addr, nil
	default:
		return "tcp", addr, nil
	}
}

func parseBindAddr(bind string) (*net.TCPAddr, error) {
	if _, _, err := net.SplitHostPort(bind); err != nil {
		if ip := net.ParseIP(bind); ip != nil && ip.To4() == nil {
			bind = "[" + bind + "]:0"
		} else {
			bind = bind + ":0"
		}
	}
	return net.ResolveTCPAddr("tcp", bind)
}

func parseSeconds(spec string) (time.Duration, error) {
	if spec == "" {
		return 0, fmt.Errorf("empty value")
	}
	if d, err := time.ParseDuration(spec); err == nil {
		return d, nil
	}
	n, err := strconv.Atoi(spec)
	if err != nil {
		return 0, fmt.Errorf("invalid value %q", spec)
	}
	return time.Duration(n) * time.Second, nil
}

func readKey(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}

func defaultPidFile(source string) string {
	if filepath.IsAbs(source) {
		return source + ".pid"
	}
	return filepath.Base(source) + ".pid"
}

func writePidFile(path string) error {
	return os.WriteFile(path, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0644)
}

func switchToSyslog() {
	w, err := syslog.New(syslog.LOG_WARNING|syslog.LOG_DAEMON, "spiped")
	if err != nil {
		log.Printf("spiped: cannot connect to syslog: %v", err)
		return
	}
	log.SetOutput(w)
}

func dropPrivileges(spec string) error {
	userPart, groupPart, _ := strings.Cut(spec, ":")
	if userPart == "" && groupPart == "" {
		return fmt.Errorf("invalid -u argument %q", spec)
	}
	if groupPart != "" {
		g, err := lookupGroup(groupPart)
		if err != nil {
			return err
		}
		gid, err := strconv.Atoi(g.Gid)
		if err != nil {
			return err
		}
		if err = syscall.Setgid(gid); err != nil {
			return err
		}
	}
	if userPart != "" {
		u, err := lookupUser(userPart)
		if err != nil {
			return err
		}
		uid, err := strconv.Atoi(u.Uid)
		if err != nil {
			return err
		}
		if err = syscall.Setuid(uid); err != nil {
			return err
		}
	}
	return nil
}

func lookupUser(nameOrID string) (*user.User, error) {
	if _, err := strconv.Atoi(nameOrID); err == nil {
		return user.LookupId(nameOrID)
	}
	return user.Lookup(nameOrID)
}

func lookupGroup(nameOrID string) (*user.Group, error) {
	if _, err := strconv.Atoi(nameOrID); err == nil {
		return user.LookupGroupId(nameOrID)
	}
	return user.LookupGroup(nameOrID)
}

// daemonize re-executes this process with an inherited listener file
// descriptor (fd 3) and exits the parent. The source socket is already bound,
// so once the parent returns spiped is ready to accept connections. If the
// key was read from standard input, it is passed to the child through a pipe
// (fd 4), since the child cannot re-read it from standard input.
func daemonize(ln net.Listener, key []byte) error {
	f, err := listenerFile(ln)
	if err != nil {
		return err
	}
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Env = append(os.Environ(), childEnv+"=1")
	cmd.ExtraFiles = []*os.File{f}
	var keyWrite, keyReadFile *os.File
	if keyFile == "-" {
		keyRead, w, perr := os.Pipe()
		if perr != nil {
			f.Close()
			return perr
		}
		keyWrite = w
		keyReadFile = keyRead
		cmd.ExtraFiles = append(cmd.ExtraFiles, keyRead)
	}
	if err := cmd.Start(); err != nil {
		f.Close()
		return err
	}
	if keyReadFile != nil {
		keyReadFile.Close()
	}
	if keyWrite != nil {
		keyWrite.Write(key)
		keyWrite.Close()
	}
	log.Printf("spiped: daemonized as pid %d", cmd.Process.Pid)
	os.Exit(0)
	return nil
}

func listenerFile(ln net.Listener) (*os.File, error) {
	switch l := ln.(type) {
	case *net.TCPListener:
		return l.File()
	case *net.UnixListener:
		return l.File()
	}
	return nil, fmt.Errorf("unsupported listener type %T", ln)
}

func die(err error) {
	log.Fatalf("spiped: %v", err)
}

// Handles a connection pair and performs synchronization:
// ---spipe/TCP---> Spiped ---TCP/spipe---> Target
// <---spipe/TCP--- Spiped <---TCP/spipe--- Target
func tcp_con_forward(src net.Conn, dst net.Conn) {
	chan_to_dst := stream_copy(src, dst)
	chan_to_src := stream_copy(dst, src)
	select {
	case n := <-chan_to_dst:
		log.Printf("spiped: connection from %s closed, %.0f bytes transfered from source", src.RemoteAddr(), float64(n))
	case n := <-chan_to_src:
		log.Printf("spiped: connection from %s closed, %.0f bytes transfered to source", src.RemoteAddr(), float64(n))
	}
}

// Performs copy operation between streams, returns a channel which
// reports the number of bytes copied when the copy loop has finished.
func stream_copy(src io.Reader, dst io.Writer) <-chan int64 {
	buf := make([]byte, 1024)
	sync_channel := make(chan int64)
	go func() {
		var total int64
		defer func() {
			// Flush buffered packet data before closing spipe connections.
			if flusher, ok := dst.(interface{ Flush() error }); ok {
				flusher.Flush()
			}
			if con, ok := dst.(net.Conn); ok {
				con.Close()
			}
			sync_channel <- total
		}()
		for {
			// make sure we do not transfer more than 2^64 byte per session
			if float64(total) >= maxBytesPerSession {
				log.Println("spiped: transfered bytes have reached the maximum allowed, aborting")
				break
			}
			nBytes, err := src.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("spiped: read error: %v", err)
				}
				break
			}
			total += int64(nBytes)
			if _, err := dst.Write(buf[:nBytes]); err != nil {
				log.Printf("spiped: write error: %v", err)
				break
			}
		}
	}()
	return sync_channel
}
