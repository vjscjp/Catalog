package mysql

import (
	"testing"

	log "github.com/CiscoCloud/shipped-bootstrap/Godeps/_workspace/src/github.com/CiscoCloud/shipped-common/logging"
)

func TestMysql(t *testing.T) {
	if err := mysql(); err != nil {
		log.Error.Printf("MySQL database test failed: %s", err.Error())
		t.Fail()
	}
}
