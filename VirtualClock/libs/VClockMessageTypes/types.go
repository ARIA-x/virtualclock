package VClockMessageTypes

type TestData struct {
	ID  string `json:"id"`
	Num int    `json:"num"`
}

type GroupInfo struct {
	GroupName string          `json:"name"`
	Sims      []SimulatorInfo `json:"sims"`
}

type MyInfo struct {
	MyName string `json:"name"`
}

type SimulatorInfo struct {
	Name     string `json:"name"`
	Nthreads int    `json:"nthreads"`
}

type Ack struct {
	Msg string `json:"msg"`
}

type Instantiate struct {
	Name  string `json:"name"`
	Token int    `json:"Token"`
}

type AckInstantiate struct {
	Token int `json:"Token"`
	Id    int `json:"Id"`
}

type State struct {
	Clock int `json:"Clock"`
	Id    int `json:"Id"`
}
