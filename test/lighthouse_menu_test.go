package test

import (
	"reflect"
	"testing"

	"github.com/alibilge/dirstral-cli/internal/app"
)

func TestLighthouseMenuItemsOrder(t *testing.T) {
	want := []string{"Start Server", "Server Status", "Stop Server", "Back"}
	if got := app.LighthouseMenuItems(); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected lighthouse options: got %v want %v", got, want)
	}
}
