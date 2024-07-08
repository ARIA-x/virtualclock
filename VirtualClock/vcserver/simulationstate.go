package main

import (
	"errors"
	"fmt"
)

type State int

const (
	None State = iota
	Ready
	Run
	Done
	Complete
)

type Instance struct {
	token     int
	id        int
	clock     int
	state     State
	cycle     int
	nextclock int
}

// TODO: 配列を map に変える

type Simulator struct {
	name      string
	instances []Instance
}

type Simulation struct {
	name       string
	idseed     int
	simulators []Simulator
}

func (simulation *Simulation) StateTransition(simulatorName string, instanceID int, clock int, state State) {
	simulator, err := simulation.searchSimulator(simulatorName)
	if err != nil {
		fmt.Print(err.Error())
		return
	} else {
		instance, err := simulator.searchInstance(instanceID)
		if err != nil {
			fmt.Print(err.Error())
			return
		} else if instance.state == Complete {
			fmt.Printf("Instance(%d) has already completed its simulation. (trying to change the state (%s)", instanceID, StateToString(state))
		} else {
			instance.clock = clock
			instance.state = state
		}
	}
}

// （現在利用されていません：削除予定）
func (simulation *Simulation) isStateAll(globalclock int, state State) bool {
	for _, simulator := range simulation.simulators {
		for _, instance := range simulator.instances {
			// シミュレーションを終了している場合や、同期対象外の時刻であればスキップ
			if instance.state == Complete || (globalclock != instance.nextclock) {
				fmt.Println("skipped-------")
				fmt.Println(globalclock, instance.clock, instance.cycle)
				continue
			}
			if instance.clock != globalclock || instance.state != state {
				return false
			}
		}
	}
	return true
}

func StateToString(state State) string {
	switch state {
	case None:
		return "None"
	case Ready:
		return "Ready"
	case Run:
		return "Run"
	case Done:
		return "Done"
	case Complete:
		return "Complete"
	default:
		return "Unknown state"
	}
}

func (simulation *Simulation) isReadyAll(globalclock int) bool {
	for _, simulator := range simulation.simulators {
		for _, instance := range simulator.instances {
			// シミュレーションを終了している場合はスキップ
			if instance.state == Complete {
				continue
			}
			if instance.clock != globalclock || instance.state != Ready {
				return false
			}
		}
	}
	return true
}

func (simulation *Simulation) isDoneAll(globalclock int) bool {
	for _, simulator := range simulation.simulators {
		for _, instance := range simulator.instances {
			// シミュレーションを終了している場合や、同期対象外の時刻であればスキップ
			if instance.state == Complete || (globalclock != instance.nextclock) {
				fmt.Println("skipped-------")
				fmt.Println(globalclock, instance.clock, instance.cycle, instance.nextclock)
				continue
			}
			if instance.clock != globalclock || instance.state != Done {
				return false
			}
		}
	}
	return true
}

func (simulation *Simulation) isCompleteAll() bool {
	for _, simulator := range simulation.simulators {
		for _, instance := range simulator.instances {
			fmt.Printf("State %s, (ID: %d) \n", StateToString(instance.state), instance.id)
			if instance.state != Complete {
				return false
			}
		}
	}
	return true
}

func (simulation *Simulation) searchSimulator(simulatorName string) (*Simulator, error) {
	for i, simulator := range simulation.simulators {
		if simulator.name == simulatorName {
			return &simulation.simulators[i], nil
		}
	}
	return &Simulator{}, errors.New("simulator not found")
}

func (simulation *Simulation) ReadyToRun() bool {
	//すべてのシミュレータの実行準備が完了したかを確認する
	for _, sim := range simulation.simulators {
		if sim.isActivatedAll() {
			continue
		} else {
			return false
		}
	}
	return true
}

func (simulator *Simulator) searchInstance(instanceID int) (*Instance, error) {
	for i, instance := range simulator.instances {
		if instance.id == instanceID {
			return &simulator.instances[i], nil
		}
	}
	return &Instance{}, errors.New("instance not found")
}

func (simulator *Simulator) isActivatedAll() bool {
	return len(simulator.instances) != 0
	/*
		for _, instance := range simulator.instances {
			if instance.token == -1 {
				return false
			}
		}
	*/
}

func (simulator *Simulator) Activate(token int) (*Instance, error) {
	for i, instance := range simulator.instances {
		if instance.token == -1 {
			simulator.instances[i].token = token
			simulator.instances[i].id = i
			return &simulator.instances[i], nil
		}
	}
	return &Instance{}, errors.New("instance not found")
}

func (Simulation *Simulation) GenerateID() int {
	ret := simulation.idseed
	simulation.idseed++
	return ret
}

func (simulation *Simulation) updateNextClock(globalclock int) {
	for i := range simulation.simulators {
		// nasty... should improve
		for j := range simulation.simulators[i].instances {
			var instance = simulation.simulators[i].instances[j]
			if globalclock == simulation.simulators[i].instances[j].nextclock {
				simulation.simulators[i].instances[j].nextclock += instance.cycle
			}
		}
	}
}
