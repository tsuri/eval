package actions

import (
	"crypto/sha256"
	pbaction "eval/proto/action"
	pbchannel "eval/proto/channel"
	pbypes "eval/proto/types"
	"fmt"

	"google.golang.org/protobuf/types/known/anypb"
)

func NewBuildImageAction() *pbaction.Action {
	config := &pbaction.BuildImageConfig{
		ImageName: "eval",
		ImageTag:  "latest",
		BaseImage: "debian:buster",
		CommitPoint: &pbaction.CommitPoint{
			// use a 'name' of golden here and the replace before sending to the engine
			CommitSha: "3e771b63822fa1b8320251c972b9617180fde46a",
			Branch:    "main",
		},
		BazelTargets: []string{"//actions/wrapper"},
	}
	anyConfig, err := anypb.New(config)
	if err != nil {
		fmt.Println("Error")
	}
	return &pbaction.Action{
		Kind: "build-image",
		//		Name: fmt.Sprintf("%s.build", parent),
		Outputs: []*pbchannel.Channel{{
			Name: "info",
			Type: &pbypes.Type{
				Impl: &pbypes.Type_Atomic{pbypes.Type_STRING},
			},
		}},
		Config: anyConfig,
	}
}

func buildImageActionDigest(action *pbaction.Action) (string, error) {
	buildConfig := pbaction.BuildImageConfig{}
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

func ActionDigest(action *pbaction.Action) (string, error) {
	digest := map[string]func(*pbaction.Action) (string, error){
		"build-image": buildImageActionDigest,
	}

	if f, present := digest[action.Kind]; present {
		return f(action)
	} else {
		return "", fmt.Errorf("digest: unknown action type %s", action.Kind)
	}
}
