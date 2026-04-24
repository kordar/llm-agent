package agent

type Agent struct {
	llm    LLM
	router *ModelRouter
	tools  *ToolRegistry
	memory Memory
	cfg    Config
}

func NewAgent(llm LLM, memory Memory, opts ...Option) *Agent {
	a := &Agent{
		llm:    llm,
		router: NewModelRouter(),
		tools:  NewToolRegistry(),
		memory: memory,
		cfg:    DefaultConfig(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(a)
		}
	}
	return a
}

func (a *Agent) RegisterTool(t Tool) error {
	return a.tools.Register(t)
}

func (a *Agent) Router() *ModelRouter {
	return a.router
}

type Option func(*Agent)

func WithConfig(cfg Config) Option {
	return func(a *Agent) {
		a.cfg = cfg
	}
}

func WithRouter(r *ModelRouter) Option {
	return func(a *Agent) {
		if r != nil {
			a.router = r
		}
	}
}

