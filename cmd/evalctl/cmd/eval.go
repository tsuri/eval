package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"

	pb "eval/proto/engine"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	conn *grpc.ClientConn
)

func getConnection(address string) (*grpc.ClientConn, error) {
	cert, err := tls.LoadX509KeyPair(
		"/data/eval/certificates/clientCertificates/grpc-client.crt",
		"/data/eval/certificates/clientCertificates/grpc-client.key")
	if err != nil {
		log.Fatalf("tls.LoadX509KeyPair err: %v", err)
	}

	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile("/data/eval/certificates/certificatesChain/grpc-root-ca-and-grpc-server-ca-and-grpc-client-ca-chain.crt")
	if err != nil {
		log.Fatalf("ioutil.ReadFile err: %v", err)
	}

	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		log.Fatalf("certPool.AppendCertsFromPEM err")
	}

	c := credentials.NewTLS(&tls.Config{
		Certificates:       []tls.Certificate{cert},
		ServerName:         "ingress.local",
		InsecureSkipVerify: true,
		RootCAs:            certPool,
	})

	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(c))
	if err != nil {
		log.Fatalf("CANNOT CONNECT")
	}
	return conn, err

	// b, err := ioutil.ReadFile("/data/eval/certificates/serverCA/grpc-server-ca.crt")
	// if err != nil {
	// 	log.Fatalf("Cannot read CA certificate")
	// }
	// cp := x509.NewCertPool()
	// if !cp.AppendCertsFromPEM(b) {
	// 	return nil, errors.New("credentials: failed to append certificates")
	// }
	// config := &tls.Config{
	// 	//		InsecureSkipVerify: true,
	// 	RootCAs: cp,
	// }
	// // config := &tls.Config{
	// // 	InsecureSkipVerify: false,
	// // 	RootCAs:            cp,
	// // }
	// conn, err := grpc.Dial(address, grpc.WithTransportCredentials(credentials.NewTLS(config)))
	// if err != nil {
	// 	log.Fatalf("CANNOT CONNECT")
	// }
	// return conn, err
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
		//		conn, err = grpc.Dial("golang2021.conf42.com:443", grpc.WithInsecure())
		conn, err = getConnection("golang2021.conf42.com:443")
		if err != nil {
			log.Fatalf("did not connect: %s", err)
		}
		defer conn.Close()

		c := pb.NewEngineServiceClient(conn)
		response, err := c.Eval(context.Background(), &pb.EvalRequest{Number: n})
		if err != nil {
			log.Fatalf("Error when calling Eval: %s", err)
		}
		log.Printf("Response from server: %s", response.Number)

	},
}

func init() {
	rootCmd.AddCommand(evalCmd)
}
