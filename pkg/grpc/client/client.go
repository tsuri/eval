package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

const (
	baseDir    = "/data/eval/certificates"
	caCert     = "ca.crt"
	clientCert = "tls.crt"
	clientKey  = "tls.key"
)

func Connect(service string) (*grpc.ClientConn, error) {
	var conn *grpc.ClientConn
	conn, err := NewConnection(service,
		filepath.Join(baseDir, caCert),
		filepath.Join(baseDir, clientCert),
		filepath.Join(baseDir, clientKey))
	// error not propagated
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	return conn, nil
}

func NewConnection(endpoint string,
	certFile string,
	clientCertFile string,
	clientKeyFile string) (*grpc.ClientConn, error) {

	host, _, err := net.SplitHostPort(endpoint)
	if err != nil {
		return nil, err
	}

	cert, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, err
	}

	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		return nil, errors.New("invalid CA certificate")
	}

	c := credentials.NewTLS(&tls.Config{
		Certificates:       []tls.Certificate{cert},
		ServerName:         host,
		InsecureSkipVerify: true,
		RootCAs:            certPool,
	})

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	ctx = metadata.AppendToOutgoingContext(ctx, "user", "USER", "pass", "PASS")
	return grpc.DialContext(ctx, endpoint, grpc.WithTransportCredentials(c))
}

// maybe I should use interceptors so that we don't need to explicitely call this function
func WithRequesterInfo(parent context.Context) context.Context {
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("cannot get hostname: %s", err)
	}
	if os.Geteuid() == 0 {
		log.Fatal("cannot execute as root")
	}
	user, err := user.Current()
	if err != nil {
		log.Fatalf("cannot get username: %s", err)
	}

	return metadata.AppendToOutgoingContext(parent, "user", user.Username, "hostname", hostname)
}
