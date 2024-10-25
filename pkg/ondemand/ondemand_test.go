package ondemand

import (
	"log"
	"testing"
)

func TestOnDemand(t *testing.T) {

	res, err := OnDemand("This is a ball")
	if err != nil {
		t.Error(err)

	}

	log.Println(res)

}
