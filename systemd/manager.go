package systemd

import (
	"fmt"
	"github.com/coreos/go-systemd/dbus"
	"log"
	"path/filepath"
)

const (
	UnitsDirectory          = "/etc/systemd/system/"
	MultiUserUnitsDirectory = "/etc/systemd/system/multi-user.target.wants/"
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
		UnitsDir: UnitsDirectory,
		State:    nil,
	}
	return &mgr, nil
}

func setupUnit(target string, conn *dbus.Conn) {
	// Blindly stop the unit in case it is running
	conn.StopUnit(target, "replace")

}

func linkUnit(target string, conn *dbus.Conn) {
	abs := "/home/core/" + target
	fixture := []string{abs}
	changes, err := conn.LinkUnitFiles(fixture, false, true)
	if err != nil {
		log.Fatalf("linkunit  failed  %v", err)
	}

	if len(changes) < 1 {
		log.Fatalf("Expected one change, got %v", changes)
	}

	runPath := filepath.Join(UnitsDirectory, target)
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

func (m *SystemdUnitManager) Enable(files []string) {
	m.EnableUnitFiles(files)
}

func (m *SystemdUnitManager) Disable(files []string) {
	m.DisableUnitFiles(files)
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
	}
	fmt.Printf("Stopped systemd unit %s(%s)", name, stat)

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

func (m *SystemdUnitManager) daemonReload() error {
	fmt.Println("Instructing systemd to reload units")
	return m.systemd.Reload()
}

func (m *SystemdUnitManager) EnableUnitFiles(files []string) {
	install, changes, err := m.systemd.EnableUnitFiles(files, false, true)
	if err != nil {
		log.Fatalf("Error failed enable %v", err)
	}

	if install != false {
		fmt.Println("Install was true")
	}

	if len(changes) < 1 {
		log.Fatalf("Expected one change, got %v", changes)
	}
	if err = m.daemonReload(); err != nil {
		fmt.Println("reload failed")
	}
}

func (m *SystemdUnitManager) DisableUnitFiles(files []string) {
	dChanges, err := m.systemd.DisableUnitFiles(files, false)
	if err != nil {
		log.Fatalf("Error failed disable %v", err)
	}

	if len(dChanges) != 1 {
		log.Fatalf("Changes should include the path, %v", dChanges)
	}
	if dChanges[0].Destination != "" {
		log.Fatalf("Change destination should be empty, %+v", dChanges[0])
	}
}
