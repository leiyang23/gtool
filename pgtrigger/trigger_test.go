package pgtrigger

import (
	"fmt"
	"testing"
	"time"
)

var trigger = Trigger{
	ConnStr: "postgres://postgres:ztesoft123@10.10.236.160:18200/zdcp?sslmode=disable",
	Schema:  "alarm",
	Table:   "alarm_item",
}

func TestSetTrigger(t *testing.T) {

	err := trigger.Set()
	if err != nil {
		t.Error(err)
	}
}

func TestListen(t *testing.T) {

	cb11 := func(d Msg) {
		fmt.Printf("11 %p\n", &d)
	}

	cb12 := func(d Msg) {
		fmt.Printf("12 %p\n", &d)
	}

	err := trigger.Listen([]func(d Msg){cb11, cb12})
	if err != nil {
		t.Error(err)
	}

	for trigger.State() >= 0 {
		time.Sleep(1 * time.Second)
	}
}
