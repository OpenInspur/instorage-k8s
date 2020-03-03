package ssh

import (
	"bytes"
	"fmt"
	"net"
	"time"

	"github.com/golang/glog"
	"golang.org/x/crypto/ssh"
)

// Interface to allow mocking of ssh.Dial, for testing SSH
type sshDialer interface {
	Dial(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error)
}

// SSHDialer is a implementation of sshDialer
type SSHDialer struct{}

var _ sshDialer = &SSHDialer{}

func (d *SSHDialer) Dial(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	conn, err := net.DialTimeout(network, addr, config.Timeout)
	if err != nil {
		return nil, err
	}
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return nil, err
	}
	conn.SetReadDeadline(time.Time{})
	return ssh.NewClient(c, chans, reqs), nil
}

// TimeoutDialer wraps an sshDialer with a timeout around Dial(). The golang
// ssh library can hang indefinitely inside the Dial() call (see issue #23835).
// Wrapping all Dial() calls with a conservative timeout provides safety against
// getting stuck on that.
type TimeoutDialer struct {
	dialer  sshDialer
	timeout time.Duration
}

// 150 seconds is longer than the underlying default TCP backoff delay (127
// seconds). This timeout is only intended to catch otherwise uncaught hangs.
const sshDialTimeout = 150 * time.Second

var gTimeoutDialer sshDialer = &TimeoutDialer{&SSHDialer{}, sshDialTimeout}

func (d *TimeoutDialer) Dial(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	config.Timeout = d.timeout
	return d.dialer.Dial(network, addr, config)
}

//IExecutor is the interface to execute ssh command on remote end
type IExecutor interface {
	Execute(cmd string) (string, string, int, error)
}

//Executor use to execute a ssh command on the remote end
type Executor struct {
	client *ssh.Client
	addr   string
	user   string
	passwd string
}

//NewExecutor create a Executor object base on the argument
func NewExecutor(addr string, user string, passwd string) *Executor {
	return &Executor{
		client: nil,
		addr:   addr,
		user:   user,
		passwd: passwd,
	}
}

//Execute run the command on the remote end and return the respond information
func (exe *Executor) Execute(cmd string) (string, string, int, error) {
	glog.Infof("login %s execute cmd: %s", exe.addr, cmd)

	session, err := exe.getSession()
	if err != nil {
		return "", "", 0, fmt.Errorf("session get failed for %s", err)
	}
	defer session.Close()

	// Run the command.
	code := 0
	var bout, berr bytes.Buffer
	session.Stdout, session.Stderr = &bout, &berr
	err = session.Run(cmd)
	if err != nil {
		// Check whether the command failed to run or didn't complete.
		if exiterr, ok := err.(*ssh.ExitError); ok {
			// If we got an ExitError and the exit code is nonzero, we'll
			// consider the SSH itself successful (just that the command run
			// errored on the addr).
			if code = exiterr.ExitStatus(); code != 0 {
				err = nil
			}
		} else {
			// Some other kind of error happened (e.g. an IOError); consider the
			// SSH unsuccessful.
			err = fmt.Errorf("failed running `%s` on %s@%s for %v", cmd, exe.user, exe.addr, err)
		}
	}
	stdout, stderr := bout.String(), berr.String()

	glog.Infof("cmd execute finish out: %s", stdout)
	glog.Infof("cmd execute finish err: %s", stderr)
	glog.Infof("cmd execute finish code: %d", code)
	glog.Infof("cmd execute finish err: %v", err)
	return stdout, stderr, code, err
}

func (exe *Executor) getSession() (*ssh.Session, error) {
	var lastErr error
	for range []string{"chance", "chance", "chance"} {
		if exe.client == nil {
			// Setup the config, dial the server, and open a session.
			config := &ssh.ClientConfig{
				User:            exe.user,
				Auth:            []ssh.AuthMethod{ssh.Password(exe.passwd)},
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			}
			client, err := gTimeoutDialer.Dial("tcp", exe.addr, config)
			if err != nil {
				glog.Errorf("SSH dial to %s@%s failed for %v", exe.user, exe.addr, err)
				lastErr = err
				continue
			} else {
				exe.client = client
			}
		}

		session, err := exe.client.NewSession()
		if err != nil {
			glog.Errorf("SSH create session to %s@%s failed for %v", exe.user, exe.addr, err)
			lastErr = err
			exe.client = nil
			continue
		}

		return session, nil
	}

	return nil, lastErr
}
