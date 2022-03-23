package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"strconv"
	"time"

	pb "eval/proto/engine"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	conn *grpc.ClientConn
)

func getConnection(endpoint string) (*grpc.ClientConn, error) {
	host, _, err := net.SplitHostPort(endpoint)
	if err != nil {
		log.Fatalf("bad endpoint err: %v", err)
	}

	cert, err := tls.LoadX509KeyPair(
		"/data/eval/C/evalctl.crt",
		"/data/eval/C/evalctl.key",
	)
	if err != nil {
		log.Fatalf("tls.LoadX509KeyPair err: %v", err)
	}

	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile("/data/eval/C/evalCA.crt")
	if err != nil {
		log.Fatalf("ioutil.ReadFile err: %v", err)
	}

	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		log.Fatalf("certPool.AppendCertsFromPEM err")
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
		log.Fatalf("CANNOT CONNECT")
	}
	return conn, err
}

// getCmd represents the get command
var evalCmd = &cobra.Command{
	Use:   "eval",
	Short: "causes the evaluation of a graph",
	Long:  `Something deeper here.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		n, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			log.Fatalf("bad argument: %s", err)
		}

		fmt.Println("Eval " + args[0])

		var conn *grpc.ClientConn
		conn, err = getConnection("engine.eval.net:443")
		if err != nil {
			log.Fatalf("did not connect: %s", err)
		}
		defer conn.Close()

		client := pb.NewEngineServiceClient(conn)
		response, err := client.Eval(context.Background(), &pb.EvalRequest{Number: n})
		if err != nil {
			log.Fatalf("Error when calling Eval: %s", err)
		}
		log.Printf("Response from server: %s", response.Number)

	},
}

func init() {
	rootCmd.AddCommand(evalCmd)
}
