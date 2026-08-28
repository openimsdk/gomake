package util

import (
	"github.com/openimsdk/tools/utils/datautil"
	"github.com/shirou/gopsutil/v4/process"
)

func ProcessesByExePath() (map[string][]*process.Process, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, err
	}

	processMap := make(map[string][]*process.Process)
	for _, p := range processes {
		exePath, err := p.Exe()
		if err != nil {
			continue
		}
		exePath = NormalizeExePath(exePath)
		processMap[exePath] = append(processMap[exePath], p)
	}
	return processMap, nil
}

func ProcessCountByExePath() (map[string]int, error) {
	processMap, err := ProcessesByExePath()
	if err != nil {
		return nil, err
	}

	countMap := make(map[string]int, len(processMap))
	for exePath, processes := range processMap {
		countMap[exePath] = len(processes)
	}
	return countMap, nil
}

func PIDsByExePath() (map[string][]int, error) {
	processMap, err := ProcessesByExePath()
	if err != nil {
		return nil, err
	}

	pidMap := make(map[string][]int, len(processMap))
	for exePath, processes := range processMap {
		pidMap[exePath] = datautil.Slice(processes, func(e *process.Process) int { return int(e.Pid) })
	}
	return pidMap, nil
}
