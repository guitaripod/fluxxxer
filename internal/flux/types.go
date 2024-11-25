package flux

type Input struct {
	Prompt             string `json:"prompt"`
	Seed               *int   `json:"seed,omitempty"`
	NumOutputs         int    `json:"num_outputs"`
	AspectRatio        string `json:"aspect_ratio"`
	OutputFormat       string `json:"output_format"`
	OutputQuality      int    `json:"output_quality"`
	DisableSafetyCheck bool   `json:"disable_safety_checker"`
}
