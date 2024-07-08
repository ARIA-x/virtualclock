package VClockMessageTypes

type TestData struct {
	ID  string `json:"id"`
	Num int    `json:"num"`
}

type Register struct {
	Name  string `json:"name"`
	Num   int    `json:"num"`
	Cycle int    `json:"cycle"`
}

type AckRegister struct {
	Name string `json:"name"`
	ID   []int  `json:"id"`
}

type Ack struct {
	Msg string `json:"msg"`
}

type Instantiate struct {
	Name  string `json:"name"`
	Token []int  `json:"Token"`
}

type AckInstantiate struct {
	Instances []RegisteredInstance
}

type RegisteredInstance struct {
	Token int `json:"Token"`
	Id    int `json:"Id"`
}

type State struct {
	Clock int `json:"Clock"`
	Id    int `json:"Id"`
}
