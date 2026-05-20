package testhelpers

import (
	"context"
	"net"
	"os"
	"time"
)

const dockerSocketDialTimeout = 200 * time.Millisecond

// DockerAvailable сообщает, доступен ли локальный Docker daemon.
func DockerAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), dockerSocketDialTimeout)
	defer cancel()
	dialer := net.Dialer{Timeout: dockerSocketDialTimeout}
	for _, sock := range []string{"/var/run/docker.sock", os.ExpandEnv("$HOME/.docker/run/docker.sock")} {
		conn, err := dialer.DialContext(ctx, "unix", sock)
		if err == nil {
			_ = conn.Close()
			return true
		}
	}
	return false
}
