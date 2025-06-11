package config

import (
	"strings"

	"github.com/Lavina-Tech-LLC/lavinagopackage/v2/conf"
)

var (
	Confs   Conf
	testing = false
)

type (
	Conf struct {
		ChatlyServiceId string
	}
)

func init() {
	arg := "conf/"
	if strings.Contains(conf.GetPath(), "tests") {
		arg = "../conf/"
	}
	Confs = conf.Get[Conf](arg)
}
