package app

const (
	lighthouseActionStart  = "Start Server"
	lighthouseActionStatus = "Server Status"
	lighthouseActionStop   = "Stop Server"
	lighthouseActionBack   = "Back"
)

func LighthouseMenuItems() []string {
	return []string{lighthouseActionStart, lighthouseActionStatus, lighthouseActionStop, lighthouseActionBack}
}
