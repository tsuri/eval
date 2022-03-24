package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

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
	conn, err := grpc.DialContext(ctx, endpoint, grpc.WithTransportCredentials(c))
	if err != nil {
		return nil, err
	}
	return conn, err

}
