package cmd

import (
	"context"
	"fmt"
	"log"
	"strconv"

	pb "eval/proto/engine"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var (
	conn *grpc.ClientConn
)

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
		conn, err = grpc.Dial("localhost:55555", grpc.WithInsecure())
		if err != nil {
			log.Fatalf("did not connect: %s", err)
		}
		defer conn.Close()

		c := pb.NewEngineServiceClient(conn)
		print(c)
		response, err := c.Eval(context.Background(), &pb.EvalRequest{Number: n})
		if err != nil {
			log.Fatalf("Error when calling SayHello: %s", err)
		}
		log.Printf("Response from server: %s", response.Number)

	},
}

func init() {
	rootCmd.AddCommand(evalCmd)
}
