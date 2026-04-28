package agent

type Agent struct {
	llm        LLM
	router     *ModelRouter
	tools      ToolManager
	toolRouter *ToolRouter
	mem        AgentMemory
	cfg        Config
}

func NewAgent(llm LLM, opts ...Option) *Agent {
	a := &Agent{
		llm:    llm,
		router: NewModelRouter(),
		tools:  noopToolManager{},
		cfg:    DefaultConfig(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(a)
		}
	}
	a.toolRouter = NewToolRouter(a.llm, a.tools, a.cfg.ToolDecision)
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
		a.toolRouter = NewToolRouter(a.llm, a.tools, a.cfg.ToolDecision)
	}
}

func WithRouter(r *ModelRouter) Option {
	return func(a *Agent) {
		if r != nil {
			a.router = r
		}
	}
}

// WithToolManager injects external tool manager implementation.
func WithToolManager(m ToolManager) Option {
	return func(a *Agent) {
		if m != nil {
			a.tools = m
			a.toolRouter = NewToolRouter(a.llm, a.tools, a.cfg.ToolDecision)
		}
	}
}

// WithMemoryManager sets memory implementation for the agent.
func WithMemoryManager(m MemoryManager) Option {
	return func(a *Agent) {
		if m != nil {
			a.mem = m
		}
	}
}

// WithMemory sets AgentMemory implementation directly.
func WithMemory(m AgentMemory) Option {
	return func(a *Agent) {
		if m != nil {
			a.mem = m
		}
	}
}

// WithEnterpriseMemory is an alias of WithMemory for enterprise memory implementations.
func WithEnterpriseMemory(m AgentMemory) Option {
	return func(a *Agent) {
		if m != nil {
			a.mem = m
		}
	}
}
