package VClockDataTypes

type Configuration struct {
	GroupName  string `json:"groupname"`
	Name       string `json:"name"`
	ClockCycle int    `json:"clockcycle"`
}

type SimulationStructure struct {
	GroupName string          `json:"groupname"`
	Sims      []SimulatorInfo `json:"sims"`
}

type SimulatorInfo struct {
	Name string `json:"name"`
}
