package app

const (
	lighthouseActionStart  = "Start Server"
	lighthouseActionStatus = "Server Status"
	lighthouseActionLogs   = "View Logs"
	lighthouseActionStop   = "Stop Server"
	lighthouseActionBack   = "Back"
)

func LighthouseMenuItems() []string {
	return []string{lighthouseActionStart, lighthouseActionStatus, lighthouseActionLogs, lighthouseActionStop, lighthouseActionBack}
}
