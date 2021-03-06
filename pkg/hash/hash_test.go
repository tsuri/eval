package hash_test

import (
	"eval/pkg/hash"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
)

type Channel struct {
	//	Name string
	Type string
}

type CommitPoint struct {
	Branch         string
	CommitSHA      string
	CommitRequired string
}

type Action struct {
	CommitPoint
	Inputs  map[string]Channel
	Outputs map[string]Channel
	Config  map[string]Channel
}

type BuildImageConfig struct {
	CommitPoint  CommitPoint
	BazelTargets []string
}

type BuildImageAction struct {
	Action
	BuildImageConfig
}

type CustomActionConfig struct {
	Cmd string
}

type CustomAction struct {
	Action
	CustomActionConfig
}

func (c *CommitPoint) Hash() []byte {
	return []byte{}
}

func TestMisc(t *testing.T) {
	bi := BuildImageAction{}

	v := 1
	fmt.Printf("Result: %v\n", hash.Hash(v))
	//nb := hash.BuildConfig{}
	fmt.Printf("Result: %v\n", hash.Hash(bi.Action))
}

func TestMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	o := "foo"

	m := NewMockHasher(ctrl)
	m.EXPECT().Hash(gomock.Eq(o)).Return([]byte("foo"))

	m.Hash(o)
}
