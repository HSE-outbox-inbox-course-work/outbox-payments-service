package sl

import "log/slog"

type Key = string

const (
	KeyError Key = "error"
)

func Error(err error) slog.Attr {
	return slog.String(KeyError, err.Error())
}
