package kubedialer

import "github.com/gotomicro/ego/core/elog"

type Logger interface {
	Debugf(format string, fields ...any)
	Infof(format string, fields ...any)
	Warnf(format string, fields ...any)
	Errorf(format string, fields ...any)
}

type defaultLogger struct{}

func (d *defaultLogger) Debugf(format string, fields ...any) {
	elog.Debugf(format, fields...)
}

func (d *defaultLogger) Errorf(format string, fields ...any) {
	elog.Errorf(format, fields...)
}

func (d *defaultLogger) Infof(format string, fields ...any) {
	elog.Infof(format, fields...)
}

func (d *defaultLogger) Warnf(format string, fields ...any) {
	elog.Warnf(format, fields...)
}

var _ Logger = (*defaultLogger)(nil)
