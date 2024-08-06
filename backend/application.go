package backend

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// THIS IS JUST A PROTOTYPE I KNOW IT'S BAD I SWEAR

const (
	osRead       = 04
	osWrite      = 02
	osEx         = 01
	osUserShift  = 6
	osGroupShift = 3
	osOthShift   = 0

	osUserR   = osRead << osUserShift
	osUserW   = osWrite << osUserShift
	osUserX   = osEx << osUserShift
	osUserRw  = osUserR | osUserW
	osUserRwx = osUserRw | osUserX

	osGroupR   = osRead << osGroupShift
	osGroupW   = osWrite << osGroupShift
	osGroupX   = osEx << osGroupShift
	osGroupRw  = osGroupR | osGroupW
	osGroupRwx = osGroupRw | osGroupX

	osOthR   = osRead << osOthShift
	osOthW   = osWrite << osOthShift
	osOthX   = osEx << osOthShift
	osOthRw  = osOthR | osOthW
	osOthRwx = osOthRw | osOthX

	osAllR   = osUserR | osGroupR | osOthR
	osAllW   = osUserW | osGroupW | osOthW
	osAllX   = osUserX | osGroupX | osOthX
	osAllRw  = osAllR | osAllW
	osAllRwx = osAllRw | osGroupX
)

type TorrentStatus uint8

const (
	Downloading TorrentStatus = iota
	Seeding
	Paused
	Stopped
)

type State struct {
	Hello string `json:"hello"`
}

const dirFileMode = os.ModeDir | (osUserRwx | osAllR)

func GetStoreDir() string {
	return os.Getenv("HOME") + string(os.PathSeparator) + ".gorrent"
}

func GetStoreFilePath() string {
	return filepath.Join(GetStoreDir(), "store.json")
}

func InitDir() (string, error) {
	dir := GetStoreDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.Mkdir(dir, dirFileMode)
		if err != nil {
			return "", err
		}
	}

	return dir, nil
}

func GetState() *State {
	var st State
	file, err := os.ReadFile(GetStoreFilePath())
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(file, &st)
	if err != nil {
		panic(err)
	}

	return &st
}

func (s *State) getPath() string {
	return GetStoreFilePath()
}

func (s *State) Write() error {
	jason, err := json.Marshal(s)
	if err != nil {
		return err
	}

	err = os.WriteFile(s.getPath(), jason, dirFileMode)
	if err != nil {
		return err
	}

	return nil
}

func InitState() State {
	dir, err := InitDir()
	if err != nil {
		panic(err)
	}

	var st State

	stateFilePath := filepath.Join(dir, "store.json")
	// if it doesn't exist
	if _, err := os.Stat(stateFilePath); os.IsNotExist(err) {
		jason, err := json.Marshal(st)
		if err != nil {
			panic(err)
		}

		err = os.WriteFile(stateFilePath, jason, dirFileMode)
		if err != nil {
			panic(err)
		}
		return st
	}
	// if it exists, read and return state
	file, err := os.ReadFile(stateFilePath)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(file, &st)
	if err != nil {
		panic(err)
	}

	return st
}
