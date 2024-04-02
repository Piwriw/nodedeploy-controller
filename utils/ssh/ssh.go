package utils_ssh

import (
	"bytes"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"k8s.io/klog/v2"
	"os"
	"runtime"
	"strings"
)

// Client ssh客户端，支持scp的.
type Client struct {
	server string
	conn   *ssh.Client
	passwd string
}

func NewClient(host string, port string, user, password string) (*Client, error) {
	conf := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	if password != "" {
		conf.Auth = []ssh.AuthMethod{ssh.Password(password)}
	}
	server := fmt.Sprintf("%s:%s", host, port)
	conn, err := ssh.Dial("tcp", server, conf)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, server: server, passwd: password}, nil
}
func (c *Client) Exec(ctx context.Context, cmd string) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			buf := make([]byte, 1<<16)
			runtime.Stack(buf, true)
			klog.Errorf("panic err:%v", err)
			klog.Errorf("panic stack:%v", string(buf))
		}
	}()
	session, err := c.conn.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	stdin, err := session.StdinPipe()
	if err != nil {
		return "", err
	}
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, c.passwd+"\n")
	}()
	var bufErr, bufOut bytes.Buffer

	session.Stdout = &bufOut
	session.Stderr = &bufErr
	errChan := make(chan error, 1)
	go func() {
		errChan <- session.Run("sudo -S " + cmd)
	}()

	select {
	case err = <-errChan:
		if err != nil {
			return "", errors.Wrapf(err, bufOut.String()+bufErr.String())
		}
	case <-ctx.Done():
		return "", ctx.Err()
	}

	return strings.TrimSpace(bufOut.String() + bufErr.String()), nil
}

// ExecPipe 执行命令并设置输出流.
func (c *Client) ExecPipe(cmdStr string, setPipe func(stdOut, stdErr io.Reader)) error {
	defer func() {
		if err := recover(); err != nil {
			buf := make([]byte, 1<<16)
			// 获取所有goroutine的stacktrace, 如果只获取当前goroutine的stacktrace, 第二个参数需要为 `false`
			runtime.Stack(buf, true)
			klog.Errorf("panic err:%v", err)
			klog.Errorf("panic stack:%v", string(buf))
		}
	}()

	session, err := c.conn.NewSession()
	if err != nil {
		return errors.Wrapf(err, c.server)
	}
	defer session.Close()

	stdErr, err := session.StderrPipe()
	if err != nil {
		return err
	}

	stdOut, err := session.StdoutPipe()
	if err != nil {
		return err
	}

	setPipe(stdOut, stdErr)

	return session.Run(cmdStr)
}

// Upload 上传文件.
func (c *Client) Upload(ctx context.Context, src, dest string) error {
	sftp, err := sftp.NewClient(c.conn)
	if err != nil {
		return err
	}
	defer sftp.Close()

	st, err := os.Stat(src)
	if err != nil {
		return errors.Wrapf(err, "stat file:%v", src)
	}

	out, err := sftp.Create(dest)
	if err != nil {
		return errors.Wrapf(err, "create dest file:%v", dest)
	}
	defer out.Close()

	in, err := os.Open(src)
	if err != nil {
		return errors.Wrapf(err, "open src file:%v", src)
	}
	defer in.Close()

	errChan := make(chan error, 1)
	go func() {
		_, err := io.Copy(out, in)
		errChan <- errors.Wrapf(err, "io copy src:%v, dest:%v", src, dest)
	}()

	select {
	case err = <-errChan:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	cmd := fmt.Sprintf("chmod %o %v", st.Mode(), dest)
	if _, err = c.Exec(ctx, cmd); err != nil {
		return errors.Wrapf(err, "cmd:%v", cmd)
	}

	return nil
}
