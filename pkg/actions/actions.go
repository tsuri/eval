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
			CommitSha: "1937df925312572ef64753e9bf20a208de98b8be",
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

func NewGenerateAction() *pbaction.Action {
	config := &pbaction.GenerateConfig{}

	anyConfig, err := anypb.New(config)
	if err != nil {
		fmt.Println("Error")
	}
	return &pbaction.Action{
		Kind: "generate",
		Outputs: []*pbchannel.Channel{{
			Name: "images",
			// TODO type should be 'Bag'
			Type: &pbypes.Type{
				Impl: &pbypes.Type_Atomic{pbypes.Type_STRING},
			},
		}},
		CmdTarget: "//actions/generate",
		Config:    anyConfig,
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

func generateActionDigest(action *pbaction.Action) (string, error) {
	generateConfig := pbaction.GenerateConfig{}
	if err := action.Config.UnmarshalTo(&generateConfig); err != nil {
		return "", err
	}
	h := sha256.New()
	// here we should hash in all inputs, which requires to recursively hash all graph precursors.
	// so it is something that should be called from engine.
	h.Write([]byte(fmt.Sprintf("%v", generateConfig.Id)))
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// ActonDigest should return separate hashes for:
// - the full action (implied by the following, but still worth returning separately
// - a separate hash for ech output

// we should have a full hash (giving an handle to where the action stores results
// and a bucket hash for geting to a list of potentially equivalent actions
func ActionDigest(action *pbaction.Action) (string, error) {
	digest := map[string]func(*pbaction.Action) (string, error){
		"build-image": buildImageActionDigest,
		"generate":    generateActionDigest,
	}

	if f, present := digest[action.Kind]; present {
		return f(action)
	} else {
		return "", fmt.Errorf("digest: unknown action type %s", action.Kind)
	}
}
