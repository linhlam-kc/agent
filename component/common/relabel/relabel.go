package relabel

import (
	"fmt"

	"github.com/grafana/regexp"
	"github.com/hashicorp/hcl/v2"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/relabel"
	"github.com/rfratto/gohcl"
)

// Action is the relabelling action to be performed.
type Action string

// All possible Action values.
const (
	Replace   Action = "replace"
	Keep      Action = "keep"
	Drop      Action = "drop"
	HashMod   Action = "hashmod"
	LabelMap  Action = "labelmap"
	LabelDrop Action = "labeldrop"
	LabelKeep Action = "labelkeep"
	Lowercase Action = "lowercase"
	Uppercase Action = "uppercase"
)

var actions = map[Action]struct{}{
	Replace:   {},
	Keep:      {},
	Drop:      {},
	HashMod:   {},
	LabelMap:  {},
	LabelDrop: {},
	LabelKeep: {},
	Lowercase: {},
	Uppercase: {},
}

// String returns the string representation of the Action type.
func (a Action) String() string {
	if _, exists := actions[a]; exists {
		return string(a)
	}
	return "Action:" + string(a)
}

// MarshalText implements encoding.TextMarshaler for Action.
func (a Action) MarshalText() (text []byte, err error) {
	return []byte(a.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler for Action.
func (a *Action) UnmarshalText(text []byte) error {
	if _, exists := actions[Action(text)]; exists {
		*a = Action(text)
		return nil
	}
	return fmt.Errorf("unrecognized action type %q", string(text))
}

// Regexp encapsulates the Regexp type from Grafana's fork of the Go stdlib regexp package.
type Regexp struct {
	*regexp.Regexp
}

func newRegexp(s string) (Regexp, error) {
	re, err := regexp.Compile("^(?:" + s + ")$")
	return Regexp{re}, err
}

func mustNewRegexp(s string) Regexp {
	re, err := newRegexp(s)
	if err != nil {
		panic(err)
	}
	return re
}

// MarshalText implements encoding.TextMarshaler for Regexp.
func (re Regexp) MarshalText() (text []byte, err error) {
	if re.String() != "" {
		return []byte(re.String()), nil
	}
	return nil, nil
}

// UnmarshalText implements encoding.TextUnmarshaler for Regexp.
func (re *Regexp) UnmarshalText(text []byte) error {
	regex, err := regexp.Compile("^(?:" + string(text) + ")$")
	if err != nil {
		return err
	}

	*re = Regexp{regex}
	return nil
}

// Config describes a relabelling step to be applied on a target.
type Config struct {
	SourceLabels []string `hcl:"source_labels,optional"`
	Separator    string   `hcl:"separator,optional"`
	Regex        Regexp   `hcl:"regex,optional"`
	Modulus      uint64   `hcl:"modulus,optional"`
	TargetLabel  string   `hcl:"target_label,optional"`
	Replacement  string   `hcl:"replacement,optional"`
	Action       Action   `hcl:"action,optional"`
}

// DefaultRelabelConfig sets the default values of fields when decoding a RelabelConfig block.
var DefaultRelabelConfig = Config{
	Action:      Replace,
	Separator:   ";",
	Regex:       mustNewRegexp("(.*)"),
	Replacement: "$1",
}

var relabelTarget = regexp.MustCompile(`^(?:(?:[a-zA-Z_]|\$(?:\{\w+\}|\w+))+\w*)+$`)

// DecodeHCL implements gohcl.Decoder.
// This method is only called on blocks, not objects.
func (rc *Config) DecodeHCL(body hcl.Body, ctx *hcl.EvalContext) error {
	*rc = DefaultRelabelConfig

	type relabelConfig Config
	err := gohcl.DecodeBody(body, ctx, (*relabelConfig)(rc))
	if err != nil {
		return err
	}

	if rc.Action == "" {
		return fmt.Errorf("relabel action cannot be empty")
	}
	if rc.Modulus == 0 && rc.Action == HashMod {
		return fmt.Errorf("relabel configuration for hashmod requires non-zero modulus")
	}
	if (rc.Action == Replace || rc.Action == HashMod || rc.Action == Lowercase || rc.Action == Uppercase) && rc.TargetLabel == "" {
		return fmt.Errorf("relabel configuration for %s action requires 'target_label' value", rc.Action)
	}
	if (rc.Action == Replace || rc.Action == Lowercase || rc.Action == Uppercase) && !relabelTarget.MatchString(rc.TargetLabel) {
		return fmt.Errorf("%q is invalid 'target_label' for %s action", rc.TargetLabel, rc.Action)
	}
	if (rc.Action == Lowercase || rc.Action == Uppercase) && rc.Replacement != DefaultRelabelConfig.Replacement {
		return fmt.Errorf("'replacement' can not be set for %s action", rc.Action)
	}
	if rc.Action == LabelMap && !relabelTarget.MatchString(rc.Replacement) {
		return fmt.Errorf("%q is invalid 'replacement' for %s action", rc.Replacement, rc.Action)
	}
	if rc.Action == HashMod && !model.LabelName(rc.TargetLabel).IsValid() {
		return fmt.Errorf("%q is invalid 'target_label' for %s action", rc.TargetLabel, rc.Action)
	}

	if rc.Action == LabelDrop || rc.Action == LabelKeep {
		if rc.SourceLabels != nil ||
			rc.TargetLabel != DefaultRelabelConfig.TargetLabel ||
			rc.Modulus != DefaultRelabelConfig.Modulus ||
			rc.Separator != DefaultRelabelConfig.Separator ||
			rc.Replacement != DefaultRelabelConfig.Replacement {

			return fmt.Errorf("%s action requires only 'regex', and no other fields", rc.Action)
		}
	}

	return nil
}

// HCLToPromRelabelConfigs bridges the HCL-based configuration of relabeling
// steps to the Prometheus implementation.
func HCLToPromRelabelConfigs(rcs []*Config) []*relabel.Config {
	res := make([]*relabel.Config, len(rcs))
	for i, rc := range rcs {
		sourceLabels := make([]model.LabelName, len(rc.SourceLabels))
		for i, sl := range rc.SourceLabels {
			sourceLabels[i] = model.LabelName(sl)
		}

		res[i] = &relabel.Config{
			SourceLabels: sourceLabels,
			Separator:    rc.Separator,
			Modulus:      rc.Modulus,
			TargetLabel:  rc.TargetLabel,
			Replacement:  rc.Replacement,
			Action:       relabel.Action(rc.Action),
			Regex:        relabel.Regexp{Regexp: rc.Regex.Regexp},
		}
	}

	return res
}
