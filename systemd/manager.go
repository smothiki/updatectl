package systemd

import (
	"fmt"
	"github.com/coreos/go-systemd/dbus"
	"log"
	"os"
	"path/filepath"
)

const (
	RunUnitsDirectory = "/run/systemd/system/"
	EtcUnitsDirectory = "/etc/systemd/system/"
)

type systemdUnitstate struct {
	LoadState   string
	ActiveState string
	SubState    string
}

type SystemdUnitManager struct {
	systemd  *dbus.Conn
	UnitsDir string
	State    *systemdUnitstate
}

func NewSystemdUnitManager() (*SystemdUnitManager, error) {
	systemd, err := dbus.New()
	if err != nil {
		return nil, err
	}

	mgr := SystemdUnitManager{
		systemd:  systemd,
		UnitsDir: RunUnitsDirectory,
		State:    nil,
	}
	return &mgr, nil
}

func setupUnit(target string, conn *dbus.Conn) {
	// Blindly stop the unit in case it is running
	conn.StopUnit(target, "replace")

	// Blindly remove the symlink in case it exists
	targetRun := filepath.Join(RunUnitsDirectory, target)
	os.Remove(targetRun)
	targetRun = filepath.Join(EtcUnitsDirectory, target)
	os.Remove(targetRun)
}

func linkUnit(target string, conn *dbus.Conn) {
	abs := "/home/core/" + target
	fixture := []string{abs}
	changes, err := conn.LinkUnitFiles(fixture, true, true)
	if err != nil {
		log.Fatalf("linkunit  failed  %v", err)
	}

	if len(changes) < 1 {
		log.Fatalf("Expected one change, got %v", changes)
	}

	runPath := filepath.Join(RunUnitsDirectory, target)
	if changes[0].Filename != runPath {
		log.Fatal("Unexpected target filename")
	}
}

func (m *SystemdUnitManager) Start(name string) {
	m.startUnit(name)
}

func (m *SystemdUnitManager) Stop(name string) {
	m.stopUnit(name)
}

func (m *SystemdUnitManager) GetUnitState(name string) (*systemdUnitstate, error) {
	info, err := m.systemd.GetUnitProperties(name)
	if err != nil {
		return nil, err
	}
	us := systemdUnitstate{
		LoadState:   info["LoadState"].(string),
		ActiveState: info["ActiveState"].(string),
		SubState:    info["SubState"].(string),
	}
	return &us, nil
}

func (m *SystemdUnitManager) startUnit(name string) {
	setupUnit(name, m.systemd)
	linkUnit(name, m.systemd)
	job, err := m.systemd.StartUnit(name, "replace")
	if err != nil {
		log.Fatalf("Failed to start systemd unit %s: %v", name, err)
	}
	fmt.Printf("Started systemd unit %s(%s)", name, job)

	if job != "done" {
		log.Fatal("Job is not done:", job)
	}

	units, err := m.systemd.ListUnits()

	var unit *dbus.UnitStatus
	for _, u := range units {
		if u.Name == name {
			unit = &u
		}
	}

	if unit == nil {
		log.Fatalf("Test unit not found in list")
	}

	if unit.ActiveState != "active" {
		log.Fatalf("Test unit not active")
	}

}

func (m *SystemdUnitManager) stopUnit(name string) {
	stat, err := m.systemd.StopUnit(name, "replace")
	if err != nil {
		log.Fatalf("Failed to stop systemd unit %s: %v", name, err)
	} else {
		log.Fatalf("Stopped systemd unit %s(%s)", name, stat)
	}
	units, err := m.systemd.ListUnits()
	var unit *dbus.UnitStatus
	unit = nil
	for _, u := range units {
		if u.Name == name {
			unit = &u
		}
	}

	if unit != nil {
		log.Fatalf("Test unit found in list, should be stopped")
	}
}
