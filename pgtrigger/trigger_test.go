package pgtrigger

import (
	"fmt"
	"testing"
	"time"
)

var trigger = Trigger{
	ConnStr: "postgres://postgres:123456@127.0.0.1:5432/qq?sslmode=disable",
	Schema:  "public",
	Table:   "alarm_item",
}

func TestSetTrigger(t *testing.T) {

	err := trigger.Set()
	if err != nil {
		t.Error(err)
	}
}

func TestListen(t *testing.T) {

	cb := func(d *Msg) {
		fmt.Println(22, *d)
	}

	err := trigger.Listen([]func(d *Msg){cb})
	if err != nil {
		t.Error(err)
	}

	for trigger.State() >= 0 {
		time.Sleep(1 * time.Second)
	}
}
