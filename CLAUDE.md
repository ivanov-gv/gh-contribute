# General

## Project Structure Convention

- `cmd/` — Executable entry points. One subdirectory per binary. Each contains only a `main.go` with minimal wiring (
  config loading, dependency init, server start).
- `internal/app/` — Core application/business logic. Orchestrates services.
- `internal/client/` — External API and system clients. One subpackage per client (e.g. `git/`, `github/`, `auth/`).
  Clients handle all communication with external systems and APIs.
- `internal/config/` — Configuration loading from environment variables.
- `internal/server/` — HTTP server, routing, request/response mapping.
- `internal/service/` — Domain services. One package per concern (e.g. `parser/`, `render/`, `name/`). Services may have
  subpackages for internal structure.
- `internal/model/` — Data types and constants. One subpackage per domain (e.g. `timetable/`, `message/`, `callback/`).
  No business logic here.
- `internal/utils/` — Generic reusable helpers not tied to any domain. One subpackage per concern (e.g. `format/`).
- `internal/.../utils` - Reusable helpers tied to a domain
- `gen/` — Generated code. Do not edit manually. Includes generated data files and mockery-generated test mocks (
  configured in `.mockery.yaml`).
- `test/` — Integration tests (separated from unit tests which live next to source files in `internal/`).
- `deploy/` — Deployment configuration (Dockerfile, etc.).
- `docs/` — Documentation and resources.
- `.env.example` — Template for required environment variables. Copy to `.env` and fill in values. `.env` is gitignored
  and loaded by `Makefile` via `include .env`.

> **Rule**: every folder directly under `internal/` must be a category (a type of structure such as `service/`,
> `client/`, `model/`), never an individual package. Individual packages live one level deeper inside their category
> folder. Do not add new packages directly under `internal/`.

## File Naming Conventions

- `<package_name>.go` — Main file of a package. Contains the primary type and its core logic (e.g.
  `blacklist/blacklist.go`, `callback/callback.go`).
- `const.go` — Package-level constants and variable declarations.
- `errors.go` — Sentinel errors for the package (`var ErrSomething = errors.New(...)`).
- `mapper.go` — Conversion functions between types of different layers/domains. Placed in the package that owns the
  conversion (e.g. `server/mapper.go` converts between Telegram API types and internal `model/message` types;
  `date/mapper.go` converts between `time.Time` and `int64`).
- `*_test.go` — Unit tests, next to the source file they test.

## Control Flow and Layers

**Layers** (`main.go` → business logic):

1. `cmd/<binary>/main.go` — Loads config, creates the app, starts the server. Minimal wiring only.
2. `internal/config/` — Reads environment variables into a config struct.
3. `internal/server/` — `RunServer()` starts the server for connecting with the outer world and wires handlers (e.g.
   http server and handlers)
4. `internal/app/` — `NewApp()` initializes all services and business logic. The `App` struct holds all service
   dependencies. Orchestrates services to fulfill the request. Knows about business logic but not about transport or
   external API types.
5. `internal/service/`) — Each service is a focused unit doing one thing (finding routes, resolving names, rendering
   messages, etc.). Services receive and return internal model types. Services do not call each other — the app layer
   coordinates them.

In between layers are:

- `internal/model/` — Pure data types and constants. No logic, no dependencies on other layers. Shared across all layers
  as the common language.
- `.../mapper.go` - live at layer boundaries. They convert between external types and internal model types. Mappers are
  always named as `<Source>To<Target>`, `from<Source>` (for external → internal mappers), `to<Target>` (for internal →
  external mappers).

## Code Structure

### Service pattern

Services are structs with unexported fields, created via `New<ServiceName>(...)` constructors that accept dependencies
as arguments and return a pointer:

```go
func NewPathFinder(deps ...) *PathFinder {
return &PathFinder{...}
}

type PathFinder struct {
field1 Type1 // unexported fields
field2 Type2
}
```

Services with no state still follow the struct pattern (`type BlackListService struct{}`).

### Naming

- **Underscore prefix for shadowed builtins**: When a local variable would shadow an import or builtin, prefix with
  `_` (e.g. `_config`, `_app`, `_timetable`, `_message`, `_callback`).
- **Import aliases**: When a package name collides with a local variable or another import, use `<purpose>_<package>`
  alias (e.g. `callback_model "...model/callback"`, `model_render "...model/render"`).
- **Type IDs as distinct types**: Use named types for IDs (`type StationId int`, `type TrainId int`) to get compile-time
  safety.
- **Map type aliases**: Define map type aliases when the map signature is long or used often (
  `type StationIdToStationMap map[StationId]Stop`, `type TrainIdSet map[TrainId]struct{}`).
- **Enum pattern**: Use `iota` for internal enums. Use typed string constants for values that appear in serialized data.
- **Sentinel errors**: Defined in `errors.go` as package-level `var Err... = errors.New(...)`.
- **Only meaningful names**: prefer using readable names, understandable without context. `fileIterator` instead of
  `iter`, `func ParseTimetable(additionalRoutesHttpPaths ...string)` instead of `func P(paths ...string)`

### Model structs

Models use flat structs with a `Type` field + optional data fields per variant instead of interfaces:

```go
type Callback struct {
Type             Type
UpdateData       UpdateData       // populated when Type == UpdateType
ReverseRouteData ReverseRouteData // populated when Type == ReverseRouteType
}
```

Consumers switch on the `Type` field and read the corresponding data.

### Logic structure

Use laconic and precise comments throughout the code for faster understanding.
It's always easier to read comments, than plain unfamiliar code, especially on code reviews. Example:

```go
package p

// ReadSomeImportantInfo reads *this* from *that* for *specific* purpose with *these* details
func ReadSomeImportantInfo() {
	// read from *that*
	...
	// convert to *this*
	...
	// validation for the *specific* purpose
	...
	// add *these* details
	...
}
```

Less nesting is better. 

Bad example:

```go
func BadFunction(user User, data []int) error {
	if user.IsActive {
		if len(data) > 0 {
			avg := calculateAverage(data)
			if avg > 50 {
				err := storeResult(avg)
				if err != nil {
					return err // Deeply nested return
				}
				return nil
			} else {
				return errors.New("average too low") // Another nested return
			}
		} else {
			return errors.New("no data to process")
		}
	} else {
		return errors.New("user is inactive")
	}
}
```

Good example:

```go
func GoodFunction(user User, data []int) error {
	// Use a guard clause for the 'IsActive' check
	if !user.IsActive {
		return errors.New("user is inactive")
	}

	// Use a guard clause for the 'len(data)' check
	if len(data) == 0 {
		return errors.New("no data to process")
	}

	avg := calculateAverage(data)

	// Use a guard clause for the 'avg' check
	if avg <= 50 {
		return errors.New("average too low")
	}

	// Process the main logic without deep nesting
	err := storeResult(avg)
	if err != nil { // Handle the potential error with an early return
		return err
	}

	return nil
}

```

### Other

- Use `github.com/samber/lo` for functional collection operations (`lo.Map`, `lo.Filter`, `lo.Must`, `lo.Flatten`, etc.)
  instead of writing manual loops when the intent is clearer with a functional style.
- Use generics for reusable utility functions operating on maps/slices (see `internal/utils/`). Also use generic
  functions in service code when the pattern is clear (e.g. `GetMessage[T any](map, key) T`).
- Define interfaces at the consumer side, not the provider side.

## Error handling

In general, every error has to have:

- **Pointer** to the place where the error occurred
- **Context** providing details about what happened
- **Result** that has to be read and understood by a human or agent

### Errors

Pattern for errors:

```go
package A

func B(s string) error { /* ... */ }

func A() error {
	parameter := "important input parameter"
	err := B(parameter)
	if err != nil {
		// Errorf with the function name, parameters in [] and %w
		return fmt.Errorf("B [parameter='%s']: %w", parameter, err)
	}
	return nil
}

```

According to the rule:

1. Pointer: "B ..." - points to the B function call
2. Context: "\[parameter='%s']"
3. Result: "... %w"

If a function is called more than once in the same scope, then make those calls and errors distinguishable.

### Logging

Pattern for logging:

```go
package main

func main() {
	const logfmt = "main: " // constant logfmt with the function name
	_app, err := app.NewApp()
	if err != nil {
		// log with logfmt + the function name and context passed
		log.Fatal().Str("some more context", "if needed").Err(fmt.Errorf(logfmt+"app.NewApp: %w", err)).Send()
	}
}
```

According to the rule:

1. Pointer: logfmt + the function name
2. Context: .Str("some more context", "if needed")
3. Result: Fatal + err

Details:

- For logging use `github.com/rs/zerolog/log` by default, if nothing else mentioned.

## Testing

### Framework

Use `github.com/stretchr/testify` for assertions:

- `assert` — for non-fatal checks (test continues on failure, reports all issues at once).
- `require` — for fatal checks where the rest of the test makes no sense if this fails (e.g. nil pointer would panic).

### Unit tests

Stored next to the source file they test, in the same package (white-box testing). This allows testing unexported
functions directly.

Run with: `go test -count=1 -race ./internal/...`

Structure:

- Name tests as `Test<FunctionOrBehavior>` (e.g. `TestUnify`, `TestFindDirectPaths`, `TestGenerateRoute`).
- Use `t.Run(name, func(t *testing.T) {...})` for subtests when iterating over cases (languages, inputs, etc.).
- Define test constants at the top of the test file for reusable test data.
- Use helper functions prefixed with the context (e.g. `renderTestDirectRoutes(...)`) to build test fixtures inline
  rather than loading from files.
- Use `t.Log` to print intermediate results for debugging.

What to test in unit tests:

- Core logic and algorithms (pathfinding, name matching, rendering).
- Data integrity — validate that constants/maps have all expected keys, no empty values, no accidental duplicates.
- Edge cases — wrong input, fuzzy matching with typos, different alphabets.

### Integration tests

Stored in `test/` directory, separate package. Test the full request-response cycle by starting the actual HTTP server
and sending real HTTP requests to it.

Run with: `go test -count=1 -race ./test/...`

Structure:

- Start the server in a goroutine with a cancellable context.
- Use `assert.Eventually` to wait for the server to be ready.
- Send HTTP requests and assert on response status codes.
- Use subtests (`t.Run`) to group related scenarios within a single server lifecycle.

### Mocking

Use [mockery](https://github.com/vektra/mockery) for generating mocks from interfaces. Configuration in `.mockery.yaml`.
Generated mocks go to `gen/mocks/<package_name>/`. Use the `.EXPECT()` pattern for setting up expectations:

```go
mockClient.EXPECT().
RequestWithContext(mock.Anything, token, "sendMessage", mock.Anything, mock.Anything, mock.Anything).
Return([]byte("{}"), nil)
```

Use `mock.Anything` for arguments you don't care about in a particular test.

### Benchmarks

Use `Benchmark<Name>(b *testing.B)` for performance-sensitive code. Document results in comments above the benchmark
function for future reference.

## Documentation

### README.md (for humans)

README is the entry point for anyone reading the project. Structure:

1. **Title and one-liner** — Project name and a single sentence describing what it does.
2. **TL;DR / Table of contents** — For long READMEs, start with either a TL;DR summary or a clickable table of contents
   so readers can jump to relevant sections.
3. **Requirements / Goals** — What the project must do. Functional and non-functional requirements.
4. **Solution details** — How the project works: interface, backend, algorithms, deployment. Explain the "why" behind
   non-obvious design decisions.
5. **Ways to improve** — Known limitations and future directions.

Guidelines:

- Write for humans who have never seen the project before.
- Include diagrams, examples, and links to external resources where helpful.
- Keep sections self-contained — a reader should be able to understand a section without reading the entire document.
- Do not duplicate code or configuration that can be found in the source. Reference file paths instead.

### CLAUDE.md (for AI agents)

CLAUDE.md is the entry point for AI coding agents. It is loaded into the system prompt automatically. Structure:

1. **General conventions** — Project-agnostic rules that apply across all projects (structure, naming, error handling,
   testing, etc.). Put these first so they can be reused across repositories.
2. **Project-specific section** — Build commands, architecture details, and anything unique to this repository.

Guidelines:

- Be concise — every line consumes context window. Prefer terse rules over verbose explanations.
- Be prescriptive — write rules the agent can follow mechanically ("use X", "never do Y"), not vague advice
  ("consider using X").
- Include code examples for patterns that are hard to describe in words.
- Do not repeat information that is obvious from the code itself (e.g. listing every file).
- Keep under 200 meaningful lines if possible — long files get truncated.
- Update when conventions change. Outdated instructions are worse than no instructions.

### Code comments

See the "Logic structure" section in Code Structure above. Comments in code should be laconic and explain the "what"
and "why" at each step, not the "how" (the code itself shows the "how").

## Deployment

### Docker

All deployment configuration lives in `deploy/`. Use a multi-stage Dockerfile:

1. **Builder stage** — Full SDK image (e.g. `golang:1.23-alpine`). Install CA certificates, copy sources, build a
   static binary with `CGO_ENABLED=0`.
2. **Runtime stage** — Minimal base image (`scratch` or `distroless`). Copy only the binary and CA certs from the
   builder.
   No shell, no package manager, no extra attack surface.

### Environment configuration

Runtime configuration is passed via environment variables, not config files. This keeps the image immutable and
environment-agnostic.

- `.env.example` lists all variables with placeholders. Copy to `.env` for local development.
- `.env` is gitignored and loaded by `Makefile` via `include .env`.
- Secrets (API tokens, etc.) are stored in the cloud provider's secret manager, never in `.env.example` or code.

### Environments

Use the `ENVIRONMENT` variable to distinguish between environments:

- `PROD` — Production. Default behavior.
- `PREPROD` — Pre-production/staging. Enables extra warnings or debug features via post-handlers in the server layer
  (e.g. appending a test environment warning to every response). The app and service layers stay unaware of the
  environment — environment-specific behavior is injected at the server layer only.
