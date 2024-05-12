package nixcli

import (
	"cmp"
	"slices"

	"golang.org/x/exp/maps"
)

type cmdPathInfoOutput []cmdPathInfoOutputStorePath

type cmdPathInfoOutputStorePath struct {
	Deriver          string   `json:"deriver"`
	NarHash          string   `json:"narHash"`
	NarSize          int      `json:"narSize"`
	Path             string   `json:"path"`
	References       []string `json:"references"`
	RegistrationTime int      `json:"registrationTime"`
	Valid            bool     `json:"valid"`
}

type cmdDerivationShowOutput map[string]cmdDerivationShowOutputDerivation

type cmdDerivationShowOutputDerivation struct {
	Args      []string          `json:"args"`
	Builder   string            `json:"builder"`
	Env       map[string]string `json:"env"`
	InputDrvs map[string]struct {
		DynamicOutputs map[string]string `json:"dynamicOutputs"`
		Outputs        []string          `json:"outputs"`
	} `json:"inputDrvs"`
	InputSrcs []string `json:"inputSrcs"`
	Name      string   `json:"name"`
	Outputs   map[string]struct {
		Path          string `json:"path"`
		HashAlgorithm string `json:"hashAlgo"`
		Hash          string `json:"hash"`
	} `json:"outputs"`
	System string `json:"system"`
}

func (o cmdDerivationShowOutputDerivation) outputPath() string {
	if o.Outputs == nil {
		return ""
	}

	if out, exists := o.Outputs["out"]; exists {
		return out.Path
	}

	keys := maps.Keys(o.Outputs)
	if len(keys) == 0 {
		return ""
	}

	slices.SortStableFunc(keys, cmp.Compare[string])
	return o.Outputs[keys[0]].Path
}

type cmdBuildDerivationOutput []cmdBuildDerivationOutputDerivation

type cmdBuildDerivationOutputDerivation struct {
	DrvPath   string            `json:"drvPath"`
	Outputs   map[string]string `json:"outputs"`
	StartTime int               `json:"startTime"`
	StopTime  int               `json:"stopTime"`
}

func (o cmdBuildDerivationOutputDerivation) outputPath() string {
	if o.Outputs == nil {
		return ""
	}

	if path, exists := o.Outputs["out"]; exists {
		return path
	}

	keys := maps.Keys(o.Outputs)
	if len(keys) == 0 {
		return ""
	}

	slices.SortStableFunc(keys, cmp.Compare[string])
	return o.Outputs[keys[0]]
}
