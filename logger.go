package rmhttp

// A Logger receives a message and optional args with the intention of logging the message
// and args with an importance level defined by which method is called.
//
// The variadic args should be entered as pairs, with the odd numbered acting as keys for
// the next value. Adding an odd number of arguments here will result in unpredictable
// behaviour.
//
// This interface can be safely satisfied by the standard library slog logger.
type Logger interface {
	Debug(string, ...any)
	Info(string, ...any)
	Warn(string, ...any)
	Error(string, ...any)
}
