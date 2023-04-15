package config

func TrialMode() {
	if Env.GemfastTrialMode == "true" {
		Env.AuthMode = "none"
		Env.FilterEnabled = "false"
		Env.MirrorEnabled = "false"
	}
}
