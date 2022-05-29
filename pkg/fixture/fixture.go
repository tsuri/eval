package fixture

////////// TYPES
type Field struct {
	Name string
	Type Type
}

type Map struct {
	Fields []Field
}

func (t *Map) Underlying() Type { return t }

type Bag struct {
	Len  int
	Type Type
}

type BasicKind int

const (
	Invalid BasicKind = iota

	Int
	Float
	String
)

type BuiltinType struct {
	Kind BasicKind
	Name string
}

func (t *BuiltinType) Underlying() Type { return t }

type Type interface {
	Underlying() Type
}

////////// ACTIONS

type CommitPoint struct {
	Name           string
	Branch         string
	CommitSHA      string
	CommitRequired string
}

type BazelTarget string

type Channel struct {
	Type Type
}

type Action struct {
	Inputs  map[string]Channel
	Outputs map[string]Channel
	Config  map[string]Channel

	CommitPoint
	Implementation BazelTarget
}

type GenerateAction struct {
	Action
}

type ProcessAction struct {
	Action
}

type AggregateAction struct {
	Action
}

type ModelTrainAction struct {
	Action
}

type AnalyzeAction struct {
	Action
}

type SummarizeAction struct {
	Action
}

type ImageBuildAction struct {
	Action
}

func NewGenerateAction(commit string) *GenerateAction {
	return &GenerateAction{
		Action: Action{
			CommitPoint: CommitPoint{},
			Inputs: map[string]Channel{
				"imageset_uuid": Channel{},
			},
			Outputs: map[string]Channel{
				"imageset": Channel{},
			},
		},
	}
}

func NewProcessAction(commit string, input string) *ProcessAction {
	return &ProcessAction{}
}

func NewAggregateAction(commit string, input string) *AggregateAction {
	return &AggregateAction{}
}

func NewModelTrainAction(commit string, input string) *ModelTrainAction {
	return &ModelTrainAction{}
}

func NewAnalyzeAction(commit string, input string) *AnalyzeAction {
	return &AnalyzeAction{}
}

func NewSummarizeAction(commit string, inputA string, inputB string) *SummarizeAction {
	return &SummarizeAction{}
}

func NewImageBuildAction() *ImageBuildAction {
	return &ImageBuildAction{}
}

var AG = map[string]any{
	"compare.baseline.train.features.generate":  NewGenerateAction("golden"),
	"compare.baseline.train.features.process":   NewProcessAction("golden", "compare.baseline.train.features.generate.images"),
	"compare.baseline.train.features.aggregate": NewAggregateAction("golden", "compare.baseline.train.features.process.out"),
	"compare.baseline.train.model_train":        NewModelTrainAction("golden", "compare.baseline.train.features.aggregate.out"),
	"compare.baseline.analyze":                  NewAnalyzeAction("golden", "compare.baseline.train.model_train.out"),
	"compare.exp.train.features.generate":       NewGenerateAction("dev"),
	"compare.exp.train.features.process":        NewProcessAction("dev", "compare.exp.train.features.generate.images"),
	"compare.exp.train.features.aggregate":      NewAggregateAction("dev", "compare.exp.train.features.process.out"),
	"compare.exp.train.model_train":             NewModelTrainAction("dev", "compare.exp.train.features.aggregate.out"),
	"compare.exp.analyze":                       NewAnalyzeAction("dev", "compare.exp.train.model_train.out"),
	"compare.summarize":                         NewSummarizeAction("golden", "compare.baseline.anlyze.out", "compare.baseline.anlyze.out"),
	"image.build":                               NewImageBuildAction(),
}
