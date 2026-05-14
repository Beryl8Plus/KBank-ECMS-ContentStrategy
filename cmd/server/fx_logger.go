package main

import (
	"context"
	"fmt"

	"go.uber.org/fx/fxevent"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/infrastructure/logger"
)

// structuredFxLogger bridges fx's internal event system to the project's
// structured logger so that hook failures, provider errors, and shutdown
// events are displayed in the same JSON format as the rest of the service
// rather than fx's default "[Fx] TERMINATED" plain-text output.
//
// Only events that carry actionable information (errors, lifecycle state
// changes) are emitted; routine "providing X" debug lines are suppressed
// to keep logs clean in production.
type structuredFxLogger struct{}

func newStructuredFxLogger() fxevent.Logger { return &structuredFxLogger{} }

func (l *structuredFxLogger) LogEvent(event fxevent.Event) {
	ctx := context.Background()
	switch e := event.(type) {

	// --- Provider / supply errors ----------------------------------------

	case *fxevent.Provided:
		if e.Err != nil {
			l.errorf(ctx, "provider failed to register %s [module: %s]: %v",
				e.ConstructorName, e.ModuleName, e.Err)
		}

	case *fxevent.Decorated:
		if e.Err != nil {
			l.errorf(ctx, "decorator failed for %s [module: %s]: %v",
				e.DecoratorName, e.ModuleName, e.Err)
		}

	case *fxevent.Supplied:
		if e.Err != nil {
			l.errorf(ctx, "supplied value failed for type %s [module: %s]: %v",
				e.TypeName, e.ModuleName, e.Err)
		}

	case *fxevent.Replaced:
		if e.Err != nil {
			l.errorf(ctx, "replace failed for types %v [module: %s]: %v",
				e.OutputTypeNames, e.ModuleName, e.Err)
		}

	// --- Invoked functions (fx.Invoke errors) ----------------------------

	case *fxevent.Invoked:
		if e.Err != nil {
			l.fatalf(ctx, "fx.Invoke failed — %s [module: %s]: %v",
				e.FunctionName, e.ModuleName, e.Err)
		}

	// --- Lifecycle hook events -------------------------------------------

	case *fxevent.OnStartExecuted:
		if e.Err != nil {
			l.fatalf(ctx, "OnStart hook failed — %s (registered by %s): %v",
				e.FunctionName, e.CallerName, e.Err)
		}

	case *fxevent.OnStopExecuted:
		if e.Err != nil {
			l.errorf(ctx, "OnStop hook failed — %s (registered by %s): %v",
				e.FunctionName, e.CallerName, e.Err)
		}

	// --- Application-level state -----------------------------------------

	case *fxevent.RolledBack:
		if e.Err != nil {
			l.fatalf(ctx, "startup rolled back — OnStart hooks were reverted: %v", e.Err)
		}

	case *fxevent.Started:
		if e.Err != nil {
			l.fatalf(ctx, "application failed to start: %v", e.Err)
		}

	case *fxevent.Stopped:
		if e.Err != nil {
			l.errorf(ctx, "application stopped with error: %v", e.Err)
		}

		// All other events (Providing, Decorating, Running, OnStartExecuting,
		// OnStopExecuting) carry no error information worth surfacing in
		// production logs; they are intentionally ignored here.
	}
}

func (l *structuredFxLogger) errorf(ctx context.Context, format string, args ...any) {
	logger.LSystem(ctx, entity.SystemLog{
		Service: "CMS-DELIVERY", Level: "ERROR",
		Message: fmt.Sprintf(format, args...),
	})
}

func (l *structuredFxLogger) fatalf(ctx context.Context, format string, args ...any) {
	logger.LSystem(ctx, entity.SystemLog{
		Service: "CMS-DELIVERY", Level: "FATAL",
		Message: fmt.Sprintf(format, args...),
	})
}
