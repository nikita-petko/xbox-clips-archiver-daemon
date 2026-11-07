package http

import "github.com/golang/glog"

type glogAdapter struct{}

func (g *glogAdapter) Printf(format string, v ...any) {
	glog.V(10000).Infof(format, v...)
}
