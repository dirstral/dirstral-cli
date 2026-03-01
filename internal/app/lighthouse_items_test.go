package app

import (
	"reflect"
	"testing"

	"github.com/alibilge/dirstral-cli/internal/host"
)

func TestLighthouseItemsForHealthStopped(t *testing.T) {
	items := lighthouseItemsForHealth(host.HealthInfo{Alive: false})
	got := itemValues(items)
	want := []string{lighthouseActionStart, lighthouseActionStatus, lighthouseActionLogs, lighthouseActionBack}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected stopped lighthouse items: got %v want %v", got, want)
	}
}

func TestLighthouseItemsForHealthRunningPrioritizesStop(t *testing.T) {
	items := lighthouseItemsForHealth(host.HealthInfo{Alive: true})
	got := itemValues(items)
	want := []string{lighthouseActionStop, lighthouseActionStatus, lighthouseActionLogs, lighthouseActionBack}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected running lighthouse items: got %v want %v", got, want)
	}
}

func itemValues(items []MenuItem) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, item.Value)
	}
	return values
}
