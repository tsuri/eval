package protoflags

import (
	"fmt"

	"github.com/emicklei/proto"
)

func FlagsFromProtobuf(proto *proto.Message) {
	fmt.Print("Inspecting protobuf")
}
