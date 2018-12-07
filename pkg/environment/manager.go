package environment

// Manager is a factory for Environments in which build actions are run. An
// Manager has access to platform properties passed to the command to be
// executed. This may allow the Manager to, for example, download container
// images or set up simulators/emulators.
type Manager interface {
	Acquire(platform map[string]string) (Environment, error)
}
