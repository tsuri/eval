package actions

import (
	"crypto/sha256"
	pbAction "eval/proto/action"
	pbAGraph "eval/proto/agraph"
	pbChannel "eval/proto/channel"
	pbTypes "eval/proto/types"
	"fmt"

	"google.golang.org/protobuf/types/known/anypb"
)

func actionBuildImage(config *pbAction.BuildImageConfig) *pbAction.Action {
	anyConfig, err := anypb.New(config)
	if err != nil {
		fmt.Println("Error")
	}
	return &pbAction.Action{
		Kind: "build-image",
		Name: "build",
		Outputs: []*pbChannel.Channel{{
			Name: "info",
			Type: &pbTypes.Type{
				Impl: &pbTypes.Type_Atomic{pbTypes.Type_STRING},
			},
		}},
		Config: anyConfig,
	}
}

func AGraphBuildImage(config *pbAction.BuildImageConfig) *pbAGraph.AGraph {
	actions := []*pbAction.Action{
		actionBuildImage(config),
	}

	return &pbAGraph.AGraph{
		Name:    "image",
		Actions: actions,
	}
}

func buildImageActionDigest(action *pbAction.Action) (string, error) {
	buildConfig := pbAction.BuildImageConfig{}
	if err := action.Config.UnmarshalTo(&buildConfig); err != nil {
		return "", err
	}
	h := sha256.New()
	// here we should hash in all inputs, which requires to recursively hash all graph precursors.
	// so it is something that should be called from engine.
	h.Write([]byte(fmt.Sprintf("%v", buildConfig.BaseImage)))
	h.Write([]byte(fmt.Sprintf("%v", buildConfig.CommitPoint)))
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// ActonDigest should return separate hashes for:
// - the full action (implied by the following, but still worth returning separately
// - a separate hash for ech output

func ActionDigest(action *pbAction.Action) (string, error) {
	digest := map[string]func(*pbAction.Action) (string, error){
		"build-image": buildImageActionDigest,
	}

	if f, present := digest[action.Kind]; present {
		return f(action)
	} else {
		return "", fmt.Errorf("digest: unknown action type %s", action.Kind)
	}
}
